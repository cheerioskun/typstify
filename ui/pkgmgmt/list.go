package pkgmgmt

import (
	"gioui.org/font"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"github.com/oligo/gioview/misc"
	"github.com/oligo/gioview/theme"
	"looz.ws/typstify/i18n"
)

type PkgList struct {
	cards      []*PkgCard
	list       *widget.List
	filtered   []*PkgCard
	filterFunc func(c *PkgCard) bool
}

type CategoryList struct {
	categoryList *widget.List
	categories   []string
	checkboxes   []*widget.Bool
	checked      []string
	selectAll    widget.Bool
}

func newPkgList(cards []*PkgCard, filter func(c *PkgCard) bool) *PkgList {
	return &PkgList{
		cards:      cards,
		filterFunc: filter,
		list: &widget.List{
			List: layout.List{
				Axis: layout.Vertical,
			},
		},
	}
}

func newCategoryList() *CategoryList {
	return &CategoryList{
		categoryList: &widget.List{
			List: layout.List{Axis: layout.Vertical},
		},
		// checkboxes: make([]*widget.Bool, 0),
	}
}

func (p *PkgList) doFilter() {
	p.filtered = p.filtered[:0]
	for _, c := range p.cards {
		if p.filterFunc != nil && p.filterFunc(c) {
			p.filtered = append(p.filtered, c)
		}
	}
}

func (p *PkgList) Layout(gtx C, th *theme.Theme) D {
	p.doFilter()

	if len(p.filtered) <= 0 {
		return layout.Center.Layout(gtx, func(gtx C) D {
			lb := material.Label(th.Theme, th.TextSize, i18n.Translate("No packages/templates found"))
			lb.Color = misc.WithAlpha(th.Fg, 0xb6)
			return lb.Layout(gtx)
		})
	}
	return material.List(th.Theme, p.list).Layout(gtx, len(p.filtered), func(gtx C, index int) D {
		card := p.filtered[index]

		return layout.Inset{Top: unit.Dp(6), Right: unit.Dp(80)}.Layout(gtx, func(gtx C) D {
			return card.Layout(gtx, th)
		})
	})
}

func (c *CategoryList) Layout(gtx C, th *theme.Theme) D {
	c.update(gtx)

	return layout.Flex{
		Axis: layout.Vertical,
	}.Layout(gtx,
		layout.Rigid(func(gtx C) D {
			return layout.Flex{
				Axis:      layout.Horizontal,
				Alignment: layout.Middle,
				Spacing:   layout.SpaceBetween,
			}.Layout(gtx,
				layout.Rigid(func(gtx C) D {
					lb := material.Subtitle2(th.Theme, i18n.Translate("Category"))
					lb.Font.Weight = font.SemiBold
					return lb.Layout(gtx)

				}),

				layout.Rigid(func(gtx C) D {
					// set the right margin size to the width of the scrollbar to align with the search box.
					return layout.Inset{Right: unit.Dp(10)}.Layout(gtx, func(gtx C) D {
						checkbox := material.CheckBox(th.Theme, &c.selectAll, i18n.Translate("Select All"))
						checkbox.Size = unit.Dp(th.TextSize)
						checkbox.Font.Style = font.Italic
						return checkbox.Layout(gtx)
					})
				}),
			)
		}),
		layout.Rigid(layout.Spacer{Height: unit.Dp(12)}.Layout),
		layout.Rigid(func(gtx C) D {
			gtx.Constraints.Min.X = gtx.Constraints.Max.X
			if len(c.categories) <= 0 {
				lb := material.Label(th.Theme, th.TextSize, i18n.Translate("No categories."))
				lb.Color = misc.WithAlpha(th.Fg, 0xb6)
				return lb.Layout(gtx)
			}

			return material.List(th.Theme, c.categoryList).Layout(gtx, len(c.categories),
				func(gtx C, index int) D {
					return layout.Inset{
						Bottom: unit.Dp(6),
					}.Layout(gtx, func(gtx C) D {
						if len(c.checkboxes) < index+1 {
							c.checkboxes = append(c.checkboxes, &widget.Bool{Value: true})
						}

						checkbox := material.CheckBox(th.Theme, c.checkboxes[index], c.categories[index])
						checkbox.Size = unit.Dp(th.TextSize * 1.2)
						return checkbox.Layout(gtx)
					})
				})
		}),
	)
}

func (c *CategoryList) setCategories(categories []string) {
	c.categories = categories
	c.checkboxes = c.checkboxes[:0]
	// a nil categories means all.
	c.checked = nil
}

func (c *CategoryList) update(gtx C) {
	refresh := false
	if c.selectAll.Update(gtx) {
		c.toggleSelection()
		refresh = true
	}

	c.checked = c.checked[:0]
	for idx, checkVal := range c.checkboxes {
		if checkVal.Update(gtx) {
			refresh = true
		}
		if checkVal.Value {
			c.checked = append(c.checked, c.categories[idx])
		}
	}

	if len(c.checked) != len(c.categories) {
		c.selectAll.Value = false
	} else {
		c.selectAll.Value = true
	}

	if refresh {
		gtx.Execute(op.InvalidateCmd{})
	}
}

func (c *CategoryList) getChecked() []string {
	return c.checked
}

func (c *CategoryList) toggleSelection() {
	for _, checkVal := range c.checkboxes {
		checkVal.Value = c.selectAll.Value
	}
}
