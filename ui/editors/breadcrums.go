package editors

import (
	"cmp"
	"image"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"slices"

	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"github.com/oligo/gioview/menu"
	"github.com/oligo/gioview/misc"
	"github.com/oligo/gioview/theme"
	"golang.org/x/exp/shiny/materialdesign/icons"
)

var (
	includeStmtPattern = regexp.MustCompile(`(import|include)\s+"(?P<path>.+\.typ)"\s+`)
	arrowRightIcon, _  = widget.NewIcon(icons.NavigationChevronRight)
	subDirArrowIcon, _ = widget.NewIcon(icons.NavigationSubdirectoryArrowRight)
	expandMoreIcon, _  = widget.NewIcon(icons.NavigationExpandMore)
	expandLessIcon, _  = widget.NewIcon(icons.NavigationExpandLess)
)

type fileBreadcrums struct {
	onSelect        func(path string)
	rootDir         string
	rootNode        *fileNode
	pathes          []*fileNode
	refreshPath     bool
	pathList        layout.List
	currentPathNode *fileNode
}

type fileNode struct {
	path string
	// included or imported files.
	children []*fileNode
	// A marker used to build a navigation path.
	// Only one node should be marked as selected in a tree.
	selected bool
	expanded bool
	click    widget.Clickable
	dropdown *menu.DropdownMenu
}

func newBreadcrums(rootDir string, rootFile string, onSelect func(path string)) *fileBreadcrums {
	bc := &fileBreadcrums{
		rootDir:  rootDir,
		onSelect: onSelect,
		rootNode: &fileNode{
			path:     rootFile,
			selected: true,
		},
	}
	bc.pathes = bc.rootNode.Pathes()

	return bc
}

func (fb *fileBreadcrums) Layout(gtx C, th *theme.Theme) D {
	fb.update(gtx)
	fb.pathList.Axis = layout.Horizontal

	return layout.Inset{
		Left:   unit.Dp(8),
		Top:    unit.Dp(2),
		Bottom: unit.Dp(2),
	}.Layout(gtx, func(gtx C) D {
		return fb.pathList.Layout(gtx, len(fb.pathes), func(gtx C, index int) D {
			child := fb.pathes[index]
			expandIcon := expandLessIcon
			if child.click.Clicked(gtx) {
				fb.onPathClicked(gtx, index, child)
			}
			if child.expanded {
				expandIcon = expandMoreIcon
			}

			return layout.Flex{
				Axis:      layout.Horizontal,
				Alignment: layout.Middle,
			}.Layout(gtx,
				layout.Rigid(func(gtx C) D {
					dims := child.click.Layout(gtx, func(gtx C) D {
						return layout.Flex{
							Axis:      layout.Horizontal,
							Alignment: layout.Middle,
						}.Layout(gtx,
							layout.Rigid(func(gtx C) D {
								lb := material.Label(th.Theme, th.TextSize, child.relPath(fb.rootDir))
								if fb.currentPathNode != child {
									lb.Color = misc.WithAlpha(th.Fg, 0xb6)
								}
								return lb.Layout(gtx)

							}),
							layout.Rigid(layout.Spacer{Width: unit.Dp(4)}.Layout),
							layout.Rigid(func(gtx C) D {
								return misc.Icon{Icon: expandIcon, Size: unit.Dp(th.TextSize) * 1.2}.Layout(gtx, th)

							}),
						)
					})

					if len(child.children) > 0 && child.dropdown != nil {
						op.Offset(image.Point{Y: dims.Size.Y}).Add(gtx.Ops)
						child.dropdown.Background = misc.WithAlpha(th.Fg, th.HoverAlpha)
						child.dropdown.MaxWidth = unit.Dp(250)
						child.dropdown.Layout(gtx, th)
					}

					return dims
				}),

				layout.Rigid(layout.Spacer{Width: unit.Dp(2)}.Layout),
				layout.Rigid(func(gtx C) D {
					if index == len(fb.pathes)-1 {
						return D{}
					}
					return misc.Icon{Icon: arrowRightIcon, Size: unit.Dp(th.TextSize) * 1.2, Color: misc.WithAlpha(th.Fg, 0xb6)}.Layout(gtx, th)
				}),
				layout.Rigid(func(gtx C) D {
					if index != len(fb.pathes)-1 {
						return D{}
					}

					return layout.Spacer{Width: unit.Dp(2)}.Layout(gtx)
				}),
			)

		})

	})

}

func (fb *fileBreadcrums) update(gtx C) {
	if fb.currentPathNode == nil {
		fb.currentPathNode = fb.rootNode
	}

	if fb.refreshPath {
		fb.pathes = fb.rootNode.Pathes()
		fb.onSelect(fb.currentPathNode.path)
		fb.refreshPath = false
	}

	for _, path := range fb.pathes {
		if path.dropdown == nil {
			continue
		}

		dismissed := path.dropdown.Update(gtx)
		if dismissed {
			path.expanded = false
		}
	}

}

func (fb *fileBreadcrums) onPathClicked(gtx C, index int, file *fileNode) {

	// read from saved file.
	options := []menu.MenuOption{}
	options = append(options, newMenuOption(file, true, fb))
	for _, child := range file.GetChildren() {
		child := child
		options = append(options, newMenuOption(child, false, fb))
	}

	file.dropdown = menu.NewDropdownMenu([][]menu.MenuOption{options})
	file.dropdown.OptionInset = layout.Inset{
		Left:   unit.Dp(2),
		Right:  unit.Dp(2),
		Top:    unit.Dp(2),
		Bottom: unit.Dp(2),
	}

	file.expanded = !file.expanded
	if len(file.children) > 0 {
		file.dropdown.ToggleVisibility(gtx)
	}

}

func newMenuOption(node *fileNode, isParent bool, fb *fileBreadcrums) menu.MenuOption {
	return menu.MenuOption{
		Layout: func(gtx C, th *theme.Theme) D {
			return layout.Flex{
				Axis:      layout.Horizontal,
				Alignment: layout.Middle,
			}.Layout(gtx,
				layout.Rigid(func(gtx C) D {
					if isParent {
						return D{}
					}
					return layout.Inset{Right: unit.Dp(4)}.Layout(gtx, func(gtx C) D {
						return misc.Icon{Icon: subDirArrowIcon, Size: unit.Dp(th.TextSize) * 0.8}.Layout(gtx, th)
					})
				}),
				layout.Rigid(func(gtx C) D {
					return material.Label(th.Theme, th.TextSize, node.relPath(fb.rootDir)).Layout(gtx)
				}),
			)
		},
		OnClicked: func() error {
			if fb.currentPathNode != nil {
				// unselect the old node
				fb.currentPathNode.selected = false
				log.Println("old path is unselected: ", fb.currentPathNode.path)
			}
			node.selected = true
			fb.currentPathNode = node
			fb.refreshPath = true
			return nil
		},
	}
}

func (fl *fileNode) relPath(rootDir string) string {
	rel, err := filepath.Rel(rootDir, fl.path)
	if err != nil {
		return fl.path
	}

	return rel
}

func (fl *fileNode) Pathes() []*fileNode {
	pathes := []*fileNode{}

	pathes, found := findMarkedPath(fl, pathes)
	if !found {
		pathes = append(pathes, fl)
	}

	return pathes
}

func normalizePath(fl *fileNode, path string) string {
	return filepath.Clean(filepath.Join(filepath.Dir(fl.path), path))
}

func (fl *fileNode) GetChildren() []*fileNode {
	pathes := extractImportedFiles(fl.path)

	// relative path to absolute path
	for idx := range pathes {
		pathes[idx] = normalizePath(fl, pathes[idx])
	}

	fl.children = slices.DeleteFunc(fl.children, func(child *fileNode) bool {
		return !slices.ContainsFunc(pathes, func(path string) bool {
			return child.path == path
		})
	})

	for _, p := range pathes {
		exists := slices.ContainsFunc(fl.children, func(child *fileNode) bool { return child.path == p })
		if !exists {
			fl.children = append(fl.children, &fileNode{path: p})
		}
	}

	// sort the children to preserve order.
	slices.SortFunc(fl.children, func(a, b *fileNode) int {
		aIdx := slices.IndexFunc(pathes, func(path string) bool { return path == a.path })
		bIdx := slices.IndexFunc(pathes, func(path string) bool { return path == b.path })

		return cmp.Compare[int](aIdx, bIdx)
	})

	return fl.children
}

// findMarkedPath performs a DFS to find the path from the root to the marked node
func findMarkedPath(node *fileNode, path []*fileNode) ([]*fileNode, bool) {
	if node == nil {
		return nil, false
	}

	// Add the current node to the path
	path = append(path, node)

	// Check if this node is marked
	if node.selected {
		return path, true
	}

	// Recursively search the children
	for _, child := range node.children {
		if p, found := findMarkedPath(child, path); found {
			return p, true
		}
	}

	// If not found in this branch, remove the current node from the path
	return path[:len(path)-1], false
}

func extractImportedFiles(file string) []string {
	content, err := os.ReadFile(file)
	if err != nil {
		log.Println("Failed to read file: ", file, err)
		return nil
	}

	matches := includeStmtPattern.FindAllStringSubmatch(string(content), -1)

	var pathes []string

	for _, match := range matches {
		// path is the second submatch:
		pathes = append(pathes, filepath.Clean(match[2]))
	}

	return pathes
}
