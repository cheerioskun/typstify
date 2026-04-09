package pkg

import (
	_ "embed"
)


type Package struct {
	Name        string        `json:"name" toml:"name"`
	Version     string        `json:"version" toml:"version"`
	Entrypoint  string        `json:"entrypoint" toml:"entrypoint"`
	Authors     []string      `json:"authors" toml:"authors"`
	License     string        `json:"license" toml:"license"`
	Description string        `json:"description" toml:"description"`
	Homepage    string        `json:"homepage" toml:"homepage"`
	Repository  string        `json:"repository" toml:"repository"`
	Keywords    []string      `json:"keywords" toml:"keywords"`
	Categories  []string      `json:"categories" toml:"categories"`
	Disciplines []string      `json:"disciplines" toml:"disciplines"`
	Compiler    string        `json:"compiler" toml:"compiler"`
	Exclude     []string      `json:"exclude" toml:"exclude"`
	Template    *TemplateInfo `json:"template" toml:"template"` // if this package contains template.
	UpdatedAt   int64         `json:"updatedAt"`
}

type TemplateInfo struct {
	Path       string `json:"path" toml:"path"`
	Entrypoint string `json:"entrypoint" toml:"entrypoint"`
	Thumbnail  string `json:"thumbnail" toml:"thumbnail"`
}
