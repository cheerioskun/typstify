// pkg handles typst package/template, either remote or local.
package pkg

import (
	"fmt"
	"os"
	"path/filepath"

	tpix "github.com/typstify/tpix-cli"
	"github.com/typstify/tpix-cli/api"
	"looz.ws/typstify/service/settings"
)

type TypstPkg struct {
	api.SearchResult
	// If the package is a remote package, it may have beed cached.
	IsCached bool
	Versions []api.PackageVersionInfo
}

func (p *TypstPkg) ImportPath() string {
	return fmt.Sprintf("@%s/%s:%s", p.Namespace, p.Name, p.LatestVersion)
}

type TypstPkgService struct {
	cacheDir string
	remoteRepo
}

func (p *TypstPkg) ThumbUrl(size string) string {
	if !p.IsTemplate {
		return ""
	}

	if size == "" {
		return fmt.Sprintf("https://packages.typst.org/preview/thumbnails/%s-%s.webp", p.Name, p.Versions[0].Version)
	}
	return fmt.Sprintf("https://packages.typst.org/preview/thumbnails/%s-%s-%s.webp", p.Name, p.Versions[0].Version, size)
}

func ImportPath(namespace string, name string, version string) string {
	return fmt.Sprintf("@%s/%s:%s", namespace, name, version)
}

func DefaultCacheDir() string {
	dir, err := os.UserCacheDir()
	if err != nil {
		return ""
	}

	return filepath.Join(dir, "typst", "packages")
}

func NewTypstPkgService(config *settings.TypstSettings) *TypstPkgService {
	cacheDir := config.PackageCacheDir

	if cacheDir == "" {
		cacheDir = DefaultCacheDir()
	}

	return &TypstPkgService{
		cacheDir: cacheDir,
	}
}

// Create a empty package using builtin template manifest. Returning the dir of
// of package, and a optional error.
func (s *TypstPkgService) CreatePkg(pkgDir string, name string, isTemplate bool) (string, error) {
	return CreatePkg(pkgDir, name, isTemplate)
}

func (s *TypstPkgService) CreateSampleDocument(projectDir string, name string) (string, error) {
	return createTemplateDocument(projectDir, name)
}

func (s *TypstPkgService) CachedPkgs() ([]TypstPkg, error) {
	pkgMap, err := scanPackages(s.cacheDir)
	if err != nil {
		return nil, err
	}

	list := make([]TypstPkg, 0)
	for _, p := range pkgMap {
		for _, v := range p {
			list = append(list, v)
		}
	}
	return list, nil
}

func (s *TypstPkgService) Download(namespace string, name string, version string) (int, error) {
	spec := fmt.Sprintf("@%s/%s", namespace, name)
	if version != "" {
		spec += ":" + version
	}

	return tpix.DownloadPackage(spec, s.cacheDir, false, nil)
}
