package navpanel

import (
	"image"
	"image/color"
	"io"
	"log"
	"path/filepath"

	"gioui.org/io/clipboard"
	"gioui.org/io/event"
	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/io/transfer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"github.com/fsnotify/fsnotify"
	"github.com/oligo/gioview/explorer"
	"github.com/oligo/gioview/menu"
	"github.com/oligo/gioview/misc"
	"github.com/oligo/gioview/theme"
	"github.com/oligo/gioview/view"
	"looz.ws/typstify/ui/dialog"
	"looz.ws/typstify/utils"
	"looz.ws/typstify/widgets"
)

// For read from clipboard use.
const mimeText = "application/text"

type visibleItem struct {
	Item      *explorer.EntryNavItem
	Depth     int
	OnClicked func(item *visibleItem)

	label widgets.InteractiveLabel
	menu  *menu.ContextMenu
}

func (v *visibleItem) IsSelected() bool {
	return v.label.IsSelected()
}

func (v *visibleItem) Unselect() {
	v.label.Unselect()
}

func (v *visibleItem) Update(gtx C) bool {
	// if v.label == nil {
	// 	v.label = &list.InteractiveLabel{}
	// }

	if v.menu == nil {
		menuOpts, fixPos := v.Item.ContextMenuOptions(gtx)
		if len(menuOpts) > 0 {
			v.menu = menu.NewContextMenu(menuOpts, fixPos)
			v.menu.PositionHint = layout.S
		}
	}

	if v.menu != nil {
		if v.menu.Update(gtx) {
			v.label.SetActivated(true)
		} else {
			v.label.SetActivated(false)
		}
	}

	// handle naviitem events
	if v.label.Update(gtx) && v.OnClicked != nil {
		v.OnClicked(v)
		if v.Item.IsDir() {
			return true
		}
	}

	return false
}

func (v *visibleItem) Layout(gtx layout.Context, th *theme.Theme, inset layout.Inset) layout.Dimensions {
	v.Update(gtx)

	macro := op.Record(gtx.Ops)
	dims := layout.Inset{Bottom: unit.Dp(1)}.Layout(gtx, func(gtx C) D {
		return v.label.Layout(gtx, th, func(gtx C, color color.NRGBA) D {
			return inset.Layout(gtx, func(gtx C) D {
				return layout.W.Layout(gtx, func(gtx C) D {
					return v.Item.Layout(gtx, th, color)
				})
			})
		})
	})
	c := macro.Stop()
	defer clip.Rect(image.Rectangle{Max: dims.Size}).Push(gtx.Ops).Pop()
	c.Add(gtx.Ops)

	// if menu is not fixed position, let it follow the pointer.
	if v.menu != nil {
		v.menu.Layout(gtx, th)
	}

	return dims
}

type NavTree struct {
	item        *explorer.EntryNavItem
	menu        *menu.ContextMenu
	outerList   widget.List
	children    []visibleItem
	needRefresh bool

	// extra fields to  handle DnD event
	entered   bool
	dndInited bool

	fsWatcher *fsnotify.Watcher

	OnClicked         func(item *visibleItem)
	OnDropConfirmFunc func(srcPath string, dest *explorer.EntryNavItem, onConfirm func())
	Indention         unit.Dp
	VerticalPadding   unit.Dp
}

func (n *NavTree) Update(gtx C) {
	if len(n.children) == 0 || n.needRefresh {
		n.children = n.children[:0]
		n.collectChildren(&n.children, 0, n.item)
		n.needRefresh = false
	}

	if n.menu == nil {
		menuOpts, fixPos := n.item.ContextMenuOptions(gtx)
		if len(menuOpts) > 0 {
			n.menu = menu.NewContextMenu(menuOpts, fixPos)
			n.menu.PositionHint = layout.S
		}
	}

	// execute root Update here as we do not render it.
	n.item.Update(gtx)

	//  handle key & pointer events to let the navtree handle paste and drag&drop of root item.
	filters := []event.Filter{
		key.Filter{Focus: n, Name: "V", Required: key.ModShortcut},
		transfer.TargetFilter{Target: n.item, Type: mimeText}, //for copy, cut and paste
		// For DnD. This ensures only dir can be dragged and dropped to.
		transfer.TargetFilter{Target: n, Type: explorer.EntryMIME},
		// Detect if pointer is inside of the dir item, so we can highlight it when dropping items to it.
		pointer.Filter{Target: n, Kinds: pointer.Enter | pointer.Leave},
	}

	for {
		ke, ok := gtx.Event(filters...)
		if !ok {
			break
		}

		switch event := ke.(type) {
		case key.Event:
			if !event.Modifiers.Contain(key.ModShortcut) {
				break
			}

			switch event.Name {
			// Initiate a paste operation, by requesting the clipboard contents; other
			// half is in DataEvent.
			case "V":
				gtx.Execute(clipboard.ReadCmd{Tag: n})
			}

		case pointer.Event:
			if event.Kind == pointer.Enter {
				n.entered = true
			} else if event.Kind == pointer.Leave {
				n.entered = false
			}

		case transfer.InitiateEvent:
			n.dndInited = true
		case transfer.CancelEvent:
			n.dndInited = false
			n.entered = false
		case transfer.DataEvent:
			// read the clipboard content:
			reader := event.Open()
			defer reader.Close()
			content, err := io.ReadAll(reader)
			if err != nil {
				break
			}

			defer gtx.Execute(op.InvalidateCmd{})

			switch event.Type {
			case mimeText:
				//FIXME: clipboard data might be invalid file path.
				p, err := explorer.ParseClipboardData(content)
				if err == nil {
					if err := n.item.OnPaste(p.Data, p.IsCut, p.GetSrc()); err != nil {
						log.Println("paste failed: ", err)
						return
					}
				} else {
					if err := n.item.OnPaste(string(content), false, nil); err != nil {
						log.Println("paste failed: ", err)
						return
					}
				}
			case explorer.EntryMIME:
				// Origin of transfer.OfferCmd is kept by gio
				source, isFromEntryItem := reader.(*explorer.EntryNavItem)
				if !isFromEntryItem {
					break
				}

				if source == n.item || source.Parent() == n.item {
					break
				}

				if n.OnDropConfirmFunc != nil {
					n.OnDropConfirmFunc(string(content), n.item, func() {
						n.item.OnPaste(string(content), true, source)
					})
				} else {
					n.item.OnPaste(string(content), true, source)
				}

			}

		}
	}

}

func (n *NavTree) collectChildren(dst *[]visibleItem, depth int, node *explorer.EntryNavItem) {
	if !node.IsDir() {
		return
	}

	if n.fsWatcher != nil {
		n.fsWatcher.Add(node.Path())
	}

	itemChildren, _ := node.Children()

	for _, child := range itemChildren {
		child := child.(*explorer.EntryNavItem)
		*dst = append(*dst, visibleItem{Item: child, Depth: depth + 1, OnClicked: n.OnClicked})
		// if child is expanded, recurse to find all nodes in DFS way.
		if child.Expanded() {
			n.collectChildren(dst, depth+1, child)
		}

	}

}

func onDropConfirmFunc2(vm view.ViewManager, root *explorer.EntryNavItem) func(srcPath string, dest *explorer.EntryNavItem, onConfirm func()) {
	rootDir := filepath.Clean(root.Path())

	return func(srcPath string, dest *explorer.EntryNavItem, onConfirm func()) {
		go func() {
			caller := dialog.NewDialogChooser[bool](vm)
			srcPath = filepath.Clean(srcPath)
			relPath, err := filepath.Rel(rootDir, srcPath)
			if err != nil {
				log.Printf("Error calculating relative path: %v\n", err)
			} else {
				srcPath = relPath
			}

			result, err := caller.Call(dialog.DndDropFileDialogViewID, map[string]any{"source": srcPath, "destination": dest.Name()})
			if err != nil {
				log.Println("DnD dialog error: ", err)
				return
			}

			if result.Params {
				onConfirm()
			}
		}()
	}
}

// func (n *NavTree) layoutRoot(gtx layout.Context, th *theme.Theme, inset layout.Inset) layout.Dimensions {
// 	maxViewportY := gtx.Constraints.Max.Y

// 	var fakeOps op.Ops
// 	original := gtx.Ops
// 	gtx.Ops = &fakeOps
// 	mainDims := n.layout(gtx, th, inset, 0)

// 	gtx.Ops = original
// 	list := material.List(th.Theme, &n.outerList)
// 	list.AnchorStrategy = material.Overlay

// 	// draw a layer to indicate a DnD dropping.
// 	macro := op.Record(gtx.Ops)
// 	dims := list.Layout(gtx, 1, func(gtx C, index int) D {
// 		return n.layout(gtx, th, inset, gtx.Metric.PxToDp(maxViewportY-mainDims.Size.Y))
// 	})
// 	callOp := macro.Stop()

// 	defer clip.Rect(image.Rectangle{Max: dims.Size}).Push(gtx.Ops).Pop()
// 	// draw a highlighted background for the potential drop target.
// 	if n.droppable() {
// 		paint.ColorOp{Color: misc.WithAlpha(th.ContrastBg, th.HoverAlpha)}.Add(gtx.Ops)
// 		paint.PaintOp{}.Add(gtx.Ops)
// 	}
// 	callOp.Add(gtx.Ops)

// 	return dims
// }

func (n *NavTree) layout(gtx C, th *theme.Theme) D {
	n.outerList.Axis = layout.Vertical
	list := material.List(th.Theme, &n.outerList)
	list.AnchorStrategy = material.Overlay
	list.ScrollbarStyle = utils.MakeScrollbar(th.Theme, list.Scrollbar, misc.WithAlpha(th.Fg, 0x30))

	return list.Layout(gtx, len(n.children), func(gtx layout.Context, index int) layout.Dimensions {
		inset := layout.Inset{
			Top:    n.VerticalPadding,
			Bottom: n.VerticalPadding,
			Left:   unit.Dp(8) + unit.Dp(n.children[index].Depth*int(n.Indention)),
			Right:  unit.Dp(10),
		}

		if n.children[index].Update(gtx) {
			n.needRefresh = true
			gtx.Execute(op.InvalidateCmd{})
		}

		return n.children[index].Layout(gtx, th, inset)
	})

}

func (n *NavTree) Layout(gtx C, th *theme.Theme) D {
	n.Update(gtx)

	macro := op.Record(gtx.Ops)
	dims := layout.Flex{
		Axis:      layout.Vertical,
		Alignment: layout.Middle,
	}.Layout(gtx,
		layout.Rigid(func(gtx C) D {
			return n.layout(gtx, th)
		}),
		layout.Flexed(1, func(gtx C) D {
			// setup an clip area for context menu and DnD, key, pointer events.
			defer clip.Rect(image.Rectangle{Max: gtx.Constraints.Max}).Push(gtx.Ops).Pop()
			event.Op(gtx.Ops, n)

			// top level menu.
			if n.menu != nil {
				n.menu.Layout(gtx, th)
			}

			return layout.Dimensions{Size: gtx.Constraints.Max}
		}),
	)

	callOp := macro.Stop()

	defer clip.Rect(image.Rectangle{Max: dims.Size}).Push(gtx.Ops).Pop()
	// draw a highlighted background for the potential drop target.
	if n.droppable() {
		paint.ColorOp{Color: misc.WithAlpha(th.ContrastBg, th.HoverAlpha)}.Add(gtx.Ops)
		paint.PaintOp{}.Add(gtx.Ops)
	}
	event.Op(gtx.Ops, n)
	callOp.Add(gtx.Ops)

	return dims
}

func (n *NavTree) droppable() bool {
	return n.entered && n.dndInited
}

func (n *NavTree) Close() error {
	if n != nil && n.fsWatcher != nil {
		if n.item != nil {
			n.fsWatcher.Remove(n.item.Path())
		}
		return n.fsWatcher.Close()
	}

	return nil
}

func NewNavTree(item *explorer.EntryNavItem, vm view.ViewManager, onClicked func(item *visibleItem)) *NavTree {
	watcher, err := fsnotify.NewWatcher()

	nt := &NavTree{
		item:              item,
		fsWatcher:         watcher,
		OnClicked:         onClicked,
		Indention:         unit.Dp(16),
		VerticalPadding:   unit.Dp(3),
		OnDropConfirmFunc: onDropConfirmFunc2(vm, item),
	}

	if err == nil {
		// Start listening for events.
		go func() {
			for {
				select {
				case event, ok := <-nt.fsWatcher.Events:
					if !ok {
						return
					}
					if event.Has(fsnotify.Create) || event.Has(fsnotify.Remove) || event.Has(fsnotify.Rename) {
						nt.needRefresh = true
					}
				case err, ok := <-nt.fsWatcher.Errors:
					log.Println("fs watch error:", err)
					if !ok {
						return
					}
				}
			}
		}()

	}

	return nt
}
