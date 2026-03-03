package typst

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"

	"archive/tar"
	"archive/zip"

	"github.com/xi2/xz"
)

const (
	// use version v0.12.0
	targetReleaseUrl = "https://api.github.com/repos/typst/typst/releases/180764750"
	latestReleaseUrl = "https://api.github.com/repos/typst/typst/releases/latest"
)

type githubRelease struct {
	ID          int64     `json:"id"`
	URL         string    `json:"url"`
	TagName     string    `json:"tag_name"`
	Name        string    `json:"name"`
	PublishedAt time.Time `json:"published_at"`
	Assets      []*Asset  `json:"assets"`
}

type Asset struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Size        int    `json:"size"`
	DownloadURL string `json:"browser_download_url"`
}

type Release struct {
	Asset
	Version     string // may not the same as version get from typst command.
	PublishedAt time.Time
}

// DownloadCounter counts the number of bytes written to it. It implements to the io.Writer interface
// and we can pass this into io.TeeReader() which will report progress on each write cycle.
type DownloadProgress struct {
	finished uint64
	total    uint64
	Err      error
}

func (dp *DownloadProgress) Write(p []byte) (int, error) {
	n := len(p)
	dp.finished += uint64(n)
	return n, nil
}

func (dp *DownloadProgress) Progress() float32 {
	return float32(dp.finished) / float32(dp.total)
}

// Downloader check and download the latest version of Typst compiler.
// Typst releases executable binary in Github, so we have to download
// using Github api.
type Downloader struct {
	targetRelease string
	destDir       string
	client        *http.Client
	onFinished    func(dlErr error)
}

func newDownloader(targetRelease string, destDir string, onFinished func(dlErr error)) *Downloader {
	c := &http.Client{
		// wait for 10 minutes to download the file
		Timeout: 10 * time.Minute,
	}

	if targetRelease == "" {
		targetRelease = latestReleaseUrl
	}

	d := &Downloader{
		client:        c,
		targetRelease: targetRelease,
		destDir:       destDir,
		onFinished:    onFinished,
	}

	return d
}

func checkExists() bool {
	if _, err := exec.LookPath(executableName); err != nil {
		return false
	}

	return true
}

func (d *Downloader) get(url string) (*http.Response, error) {
	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	return d.client.Do(request)
}

func (d *Downloader) getRelease() (*Release, error) {
	// Get release meta from Github API
	resp, err := d.get(d.targetRelease)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	decoder := json.NewDecoder(resp.Body)
	var release = githubRelease{}
	err = decoder.Decode(&release)
	if err != nil {
		return nil, err
	}

	// Check if the latest release is newer that the installed one.
	// Compare version using string compare. We have to check if there exists a compiler already
	// in order to check that.
	if checkExists() && strings.Compare(CurrentVersion(), strings.TrimPrefix(release.TagName, "v")) >= 0 {
		// no need to download
		log.Println("Your typst compiler is already the required version.")
		return nil, errors.New("You are already using the required version of Typst.")
	}

	arch := runtime.GOARCH
	if arch == "arm64" {
		arch = "aarch64"
	} else if arch == "amd64" {
		arch = "x86_64"
	}

	assetNamePat := fmt.Sprintf(`^typst-%s-\w+-%s-?\w*?\.(tar\.xz|zip)$`, arch, runtime.GOOS)
	//log.Println("re pattern: ", assetNamePat)
	re := regexp.MustCompile(assetNamePat)
	var target *Asset
	for _, asset := range release.Assets {
		if re.Match([]byte(asset.Name)) {
			target = asset
			break
		}
	}

	if target == nil {
		log.Println("No matched typst release for your system")
		return nil, fmt.Errorf("No matched typst release for your system")
	}

	return &Release{
		Asset:       *target,
		Version:     release.Name,
		PublishedAt: release.PublishedAt}, nil
}

// Download downloads the release file in async manner, and reports its progress.
func (d *Downloader) Download() *DownloadProgress {
	progress := &DownloadProgress{}

	go func() {
		target, err := d.getRelease()
		if err != nil {
			progress.Err = err
			return
		}

		progress.total = uint64(target.Size)

		// download the asset
		log.Println("Downloading the lastest release of Typst...")
		resp, err := d.get(target.DownloadURL)
		if err != nil {
			progress.Err = err
			return
		}

		var targetFile *os.File
		targetFile, err = os.OpenFile(filepath.Join(d.destDir, target.Name), os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0755)
		if err != nil {
			progress.Err = err
			return
		}

		defer targetFile.Close()
		defer os.Remove(targetFile.Name())

		if n, err := io.Copy(targetFile, io.TeeReader(resp.Body, progress)); err != nil || n != int64(target.Size) {
			progress.Err = errors.New("Download typst error")
			d.onFinished(progress.Err)
			return
		} else if d.onFinished != nil {
			d.onFinished(nil)
		}

		//uncompress, do not return progress until it finishes.
		err = d.uncompressToDir(targetFile)
		if err != nil {
			progress.Err = err
			return
		}
	}()

	return progress
}

func (d *Downloader) uncompressToDir(targetFile *os.File) error {
	isZip := strings.HasSuffix(targetFile.Name(), ".zip")
	isXz := strings.HasSuffix(targetFile.Name(), ".tar.xz")
	targetFile.Seek(0, io.SeekStart)

	if isXz {
		err := d.uncompressXZFile(targetFile)
		if err != nil {
			return err
		}
	} else if isZip {
		err := d.unzipFile(targetFile)
		if err != nil {
			return err
		}
	} else {
		return errors.New("Unknown typst release format: " + targetFile.Name())
	}

	// move uncompressed files to destDir:
	var suffix string
	if isXz {
		suffix = ".tar.xz"
	} else {
		suffix = ".zip"
	}

	dir := strings.TrimSuffix(targetFile.Name(), suffix)

	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		os.Rename(filepath.Join(dir, entry.Name()), filepath.Join(d.destDir, entry.Name()))

		// os.Rename does not copy file attributes such as file permission.
		// We have to do a file copy here.
		// src, err := os.Open(filepath.Join(dir, entry.Name()))
		// if err != nil {
		// 	return err
		// }
		// defer src.Close()

		// dest, err := os.Create(filepath.Join(d.destDir, entry.Name()))
		// if err != nil {
		// 	return err
		// }
		// defer dest.Close()

		// _, err = io.Copy(dest, src)
		// if err != nil {
		// 	return err
		// }

		// // Get the original file's permissions
		// info, err := src.Stat()
		// if err != nil {
		// 	return err
		// }

		// // Set the permissions to the copied file
		// err = os.Chmod(dest.Name(), info.Mode())
		// if err != nil {
		// 	return err
		// }

		// // Delete the original file
		// os.Remove(src.Name())
	}

	return nil
}

func (d *Downloader) uncompressXZFile(targetFile *os.File) error {
	r, err := xz.NewReader(targetFile, 0)
	if err != nil {
		return err
	}

	// Create a tar Reader
	tr := tar.NewReader(r)
	// Iterate through the files in the archive.
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		switch header.Typeflag {
		case tar.TypeDir:
			// create a directory
			err = os.MkdirAll(filepath.Join(d.destDir, header.Name), 0755)
			if err != nil {
				return err
			}
		case tar.TypeReg:
			// write a file
			fmt.Println("extracting: " + header.Name)
			w, err := os.Create(filepath.Join(d.destDir, header.Name))
			if err != nil {
				return err
			}
			_, err = io.Copy(w, tr)
			if err != nil {
				return err
			}
			w.Close()
		}
	}

	return nil
}

func (d *Downloader) unzipFile(targetFile *os.File) error {
	stat, err := targetFile.Stat()
	if err != nil {
		return err
	}
	var r *zip.Reader
	r, err = zip.NewReader(targetFile, stat.Size())
	if err != nil {
		return err
	}

	for _, f := range r.File {
		if f.FileInfo().IsDir() {
			// create a directory
			err = os.MkdirAll(filepath.Join(d.destDir, f.Name), 0755)
			if err != nil {
				return err
			}
			continue
		}

		// normal file, write to destDir directly.
		dest, err := os.Create(filepath.Join(d.destDir, f.Name))
		if err != nil {
			return err
		}
		defer dest.Close()

		rc, err := f.Open()
		if err != nil {
			log.Printf("impossible to open file %s in archine: %s", f.Name, err)
		}
		defer rc.Close()

		_, err = io.Copy(dest, rc)
		if err != nil {
			return err
		}
	}

	return nil

}
