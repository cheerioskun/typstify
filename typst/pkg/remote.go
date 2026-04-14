package pkg

import (
	"fmt"
	"sync/atomic"

	tpix "github.com/typstify/tpix-cli"
	"github.com/typstify/tpix-cli/api"
)

type remoteRepo struct {
	loading   atomic.Bool
	data      atomic.Pointer[api.SearchResponse]
	searchKey string
}

// ListPkgs queries packages/templates from TPIX package index, this includes
// public and private namespaces the user have access permissions.
func (r *remoteRepo) SearchPkgs(namespace string, kind string, category string, query string) ([]TypstPkg, int, error) {
	if r.searchKey != "" && r.searchKey == fmt.Sprintf("%s:%s:%s", namespace, kind, query) {
		cachedData := r.data.Load()
		if cachedData != nil {
			pkgs := r.convertToPkg(cachedData)
			return pkgs, cachedData.Total, nil
		}
	}

	if r.loading.Load() {
		return nil, 0, nil
	}

	resp, err := r.search(namespace, kind, category, query)
	if err != nil {
		return nil, 0, err
	}

	pkgs := r.convertToPkg(resp)
	r.data.Store(resp)

	return pkgs, resp.Total, nil
}

func (r *remoteRepo) search(namespace string, kind string, category string, query string) (*api.SearchResponse, error) {
	if !r.loading.CompareAndSwap(false, true) {
		return nil, nil
	}

	defer func() {
		r.loading.CompareAndSwap(true, false)
	}()

	return tpix.SearchPackages(namespace, query, kind, category, "", 100)

}

func (r *remoteRepo) convertToPkg(resp *api.SearchResponse) []TypstPkg {
	pkgs := make([]TypstPkg, len(resp.Results))
	for i, result := range resp.Results {
		pkgs[i] = TypstPkg{
			SearchResult: result,
		}
	}

	return pkgs
}
