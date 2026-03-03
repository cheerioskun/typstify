package typst

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/fsnotify/fsnotify"
)

// Compiler is a typst compiler wrapper.
type Compiler struct {
	workDir string
	tempDir string

	// lock to keep compile be executed sequencely.
	mux          sync.Mutex
	currentWatch *exec.Cmd

	// fsWatcher for Watch compile method
	fsWatcher *fsnotify.Watcher
}

type CompileParams struct {
	Options CompileCmdOptions

	// file to be compiled
	InputFile string
	// or use a io.Reader
	InputReader io.Reader

	// The filename of the output file, without extension name.
	// If typst generates multiple files, this is the prefix of the final filenames.
	OutFilename string
	OutDir      string
	// command stdout and stderr output writer.
	CmdOut io.Writer
}

// NewCompiler creates a compiler. pkgDir and cacheDir can be left empty to use the default.
func NewCompiler(workDir string) (*Compiler, error) {
	if _, err := exec.LookPath(executableName); err != nil {
		return nil, err
	}

	c := &Compiler{
		workDir: workDir,
	}

	c.mkTempDir()
	return c, nil
}

func (t *Compiler) commonOptions(params *CompileParams) []string {
	params.Options.RootDir = t.workDir

	options := params.Options.Build()

	args := []string{}

	// set INPUT and OUTPUT
	if params.InputFile == "" || params.InputFile == "-" {
		if params.InputReader == nil {
			panic("A reader should provided")
		}
		args = append(args, "-")
		// rewrite InputFile
		params.InputFile = "-"
	} else {
		args = append(args, params.InputFile)
	}

	outDir := params.OutDir
	if outDir == "" {
		outDir = t.tempDir
	} else {
		err := os.MkdirAll(outDir, 0755)
		if err != nil {
			log.Println("cannot mkdir: ", err)
			outDir = t.tempDir
		}
	}

	if params.Options.Format == PNG || params.Options.Format == SVG {
		// If multple image is generated, we are required to provide a outfile name with pattern `{n}`
		args = append(args, filepath.Join(outDir, fmt.Sprintf("%s{n}.%s", params.OutFilename, params.Options.Format)))
	} else {
		// pdf
		args = append(args, filepath.Join(outDir, fmt.Sprintf("%s.%s", params.OutFilename, params.Options.Format)))
	}

	return append(options, args...)
}

// compileAndCallback reads from input and compile the content into files with a supported output format.
func (t *Compiler) Compile(ctx context.Context, opts *CompileParams, callback func(files []string)) error {
	// t.mux.Lock()
	// defer t.mux.Unlock()

	ctx, cancelFunc := context.WithCancel(ctx)
	defer cancelFunc()

	err := t.compile(ctx, opts)
	if err != nil {
		return err
	}

	var outFiles = []string{}
	fileInfos, err := os.ReadDir(opts.OutDir)
	if err != nil {
		return err
	}

	for _, info := range fileInfos {
		if info.IsDir() || !strings.HasPrefix(info.Name(), opts.OutFilename) {
			continue
		}

		outFiles = append(outFiles, filepath.Join(opts.OutDir, info.Name()))
	}

	if len(outFiles) > 0 && callback != nil {
		callback(outFiles)
	}

	return nil
}

func (t *Compiler) compile(ctx context.Context, opts *CompileParams) error {
	args := []string{"compile"}
	args = append(args, t.commonOptions(opts)...)

	cmd := newCmd(ctx, args...)
	if opts.InputFile == "-" {
		// setup a stdin pipe to read from a reader.
		stdin, err := cmd.StdinPipe()
		if err != nil {
			return err
		}

		go func() {
			defer stdin.Close()
			io.Copy(stdin, opts.InputReader)
		}()
	}

	log.Println("executing command: ", cmd.String())

	cmd.Stdout = opts.CmdOut
	cmd.Stderr = opts.CmdOut
	return cmd.Run()
}

// Watch watches the workdir and compiles it on every change. Output files are also watched to
// notify the caller on every output change.
func (t *Compiler) Watch(ctx context.Context, opts *CompileParams, callback func(files []string)) error {
	if t.currentWatch != nil {
		return nil
	}

	t.mux.Lock()
	defer t.mux.Unlock()

	var err error
	if t.fsWatcher == nil {
		t.fsWatcher, err = fsnotify.NewWatcher()
		if err != nil {
			return err
		}
	}

	for _, target := range t.fsWatcher.WatchList() {
		if target == t.tempDir {
			return fmt.Errorf("%s is already being watched", target)
		}
	}

	// Start listening for events.
	go func() {
		for {
			select {
			case event, ok := <-t.fsWatcher.Events:
				if !ok {
					return
				}
				if event.Has(fsnotify.Write) || event.Has(fsnotify.Create) {
					callback([]string{event.Name})
				}
			case err, ok := <-t.fsWatcher.Errors:
				log.Println("fs watch error:", err)
				if !ok {
					return
				}
			}
		}
	}()

	// Add a watch target
	t.fsWatcher.Add(t.tempDir)
	// t.fsWatcher.Add(t.workDir)

	defer t.Unwatch()
	// Need to block the current goroutine to make fs notify work. If watch compile failed with error
	// we have to quit the entire watch flow.
	err = t.watchAndCompile(ctx, opts)
	if err != nil {
		return err
	}
	return nil
}

func (t *Compiler) Unwatch() error {
	if t.currentWatch != nil {
		err := t.currentWatch.Cancel()
		if err != nil {
			return err
		}
		t.currentWatch = nil
	}

	if t.fsWatcher == nil {
		return nil
	}

	t.fsWatcher.Remove(t.tempDir)
	defer func() {
		t.fsWatcher = nil
	}()
	return t.fsWatcher.Close()
}

func (t *Compiler) watchAndCompile(ctx context.Context, opts *CompileParams) error {
	args := []string{"watch"}
	args = append(args, t.commonOptions(opts)...)

	cmd := newCmd(ctx, args...)

	// stderr, err := cmd.StderrPipe()
	// if err != nil {
	// 	return err
	// }
	cmd.Stdout = opts.CmdOut
	cmd.Stderr = opts.CmdOut

	//log.Println("executing command: ", cmd.String())
	err := cmd.Start()
	if err != nil {
		return err
	}

	t.currentWatch = cmd

	// go func() {
	// 	defer stderr.Close()
	// 	// read error output
	// 	buffer := make([]byte, 16)
	// 	io.CopyBuffer(log.Writer(), stderr, buffer)
	// }()

	log.Printf("Waiting for command to finish...")
	return t.currentWatch.Wait()
}

func (t *Compiler) mkTempDir() {
	if t.tempDir != "" {
		return
	}

	tempDir, err := os.MkdirTemp("", "typstify-"+filepath.Base(t.workDir))
	if err != nil {
		panic(err)
	}

	t.tempDir = tempDir
}

func (t *Compiler) Close() error {
	t.Unwatch()

	return os.Remove(t.tempDir)
}
