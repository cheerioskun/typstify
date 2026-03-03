package pkg

import (
	"cmp"
	"encoding/json"
	"log"
	"net/http"
	"slices"
	"sync/atomic"
	"time"
)

const (
	// The non-public official template and package index API for Typst.
	officialPkgIndexUrl = "https://packages.typst.org/preview/index.json"
	defaultNamespace    = "preview"
)

type officialPkgRepo struct {
	cacheDir      string
	loading       atomic.Bool
	lastFetchTime time.Time
	lastFetchErr  error
	data          []Package
}

func (op *officialPkgRepo) CachedPkgs() ([]*TypstPkg, error) {
	pkgMap, err := scanPackages(op.cacheDir, true)
	if err != nil {
		return nil, err
	}

	list := make([]*TypstPkg, 0)
	for _, p := range pkgMap {
		for _, v := range p {
			list = append(list, v)
		}
	}
	return list, nil
}

// ListPkgs queries remote package index and merges the found packages with
// local cached package meta data to give the caller a unified view.
func (op *officialPkgRepo) ListPkgs() ([]*TypstPkg, error) {
	if !op.loading.CompareAndSwap(false, true) {
		return nil, nil
	}

	defer func() {
		op.loading.CompareAndSwap(true, false)
	}()

	pkgs, err := op.fetchIndex()
	if err != nil {
		log.Println("get remote package index failed: ", err)
		return op.CachedPkgs()
	}

	cached, err := scanPackages(op.cacheDir, true)
	if err != nil {
		log.Println("scan cache package failed: ", err)
	}

	pkgMap := make(map[string]*TypstPkg)
	for _, pkg := range pkgs {
		p, ok := pkgMap[pkg.Name]
		if !ok {
			p = &TypstPkg{
				Namespace: defaultNamespace,
				Name:      pkg.Name,
			}
			pkgMap[pkg.Name] = p
		}

		ver := &PackageInfo{Package: pkg, Namespace: defaultNamespace, IsCached: false, IsLocal: false}
		cachedPkg, ok := cached[defaultNamespace][pkg.Name]
		if ok && slices.ContainsFunc(cachedPkg.Versions, func(v *PackageInfo) bool { return v.Version == ver.Version }) {
			ver.IsCached = true
		}

		p.Versions = append(p.Versions, ver)
		slices.SortFunc(p.Versions, func(a, b *PackageInfo) int {
			return -cmp.Compare(a.Version, b.Version)
		})
	}

	list := make([]*TypstPkg, 0)
	for _, p := range pkgMap {
		list = append(list, p)
	}
	slices.SortStableFunc(list, func(a, b *TypstPkg) int {
		return cmp.Compare(a.Name, b.Name)
	})

	return list, nil
}

func (op *officialPkgRepo) fetchIndex() ([]Package, error) {
	now := time.Now()
	if op.lastFetchTime.IsZero() {
		op.lastFetchTime = now
	}

	if op.data == nil || op.lastFetchErr != nil || now.Sub(op.lastFetchTime) > 10*time.Minute {
		pkgs, err := op.getRemoteIndex()
		op.lastFetchErr = err
		if err == nil {
			op.data = pkgs
		}
		op.lastFetchTime = now
	} else {
		log.Println("returning cached package index")
	}

	return op.data, op.lastFetchErr
}

func (op *officialPkgRepo) getRemoteIndex() ([]Package, error) {
	request, err := http.NewRequest("GET", officialPkgIndexUrl, nil)
	if err != nil {
		return nil, err
	}

	client := &http.Client{
		// wait for some time to download the file
		Timeout: 15 * time.Second,
	}

	resp, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var pkgs []Package
	decoder := json.NewDecoder(resp.Body)
	err = decoder.Decode(&pkgs)
	if err != nil {
		return nil, err
	}

	return pkgs, nil
}
