package pkg

import (
	"fmt"
	"sync/atomic"

	tpix "github.com/typstify/tpix-cli"
)

type remoteRepo struct {
	loading   atomic.Bool
	data      atomic.Pointer[[]TypstPkg]
	searchKey string
}

// ListPkgs queries packages/templates from TPIX package index, this includes
// public and private namespaces the user have access permissions.
func (r *remoteRepo) SearchPkgs(namespace string, kind string, category string, query string) ([]TypstPkg, error) {
	if r.searchKey != "" && r.searchKey == fmt.Sprintf("%s:%s:%s", namespace, kind, query) {
		cachedData := r.data.Load()
		if cachedData != nil {
			return *cachedData, nil
		}
	}

	if r.loading.Load() {
		return nil, nil
	}

	pkgs, err := r.search(namespace, kind, category, query)
	if err != nil {
		return nil, err
	}

	r.data.Store(&pkgs)
	return *r.data.Load(), nil
}

func (r *remoteRepo) search(namespace string, kind string, category string, query string) ([]TypstPkg, error) {
	if !r.loading.CompareAndSwap(false, true) {
		return nil, nil
	}

	defer func() {
		r.loading.CompareAndSwap(true, false)
	}()

	results, err := tpix.SearchPackages(namespace, query, kind, category, "", 100)
	if err != nil {
		return nil, err
	}

	pkgs := make([]TypstPkg, len(results.Results))
	for i, result := range results.Results {
		pkgs[i] = TypstPkg{
			SearchResult: result,
		}
	}

	return pkgs, nil
}
