package pkgmgmt

import (
	"errors"
	"slices"
	"strings"
	"time"

	"gioui.org/font"
	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"github.com/oligo/gioview/image"
	"github.com/oligo/gioview/misc"
	"github.com/oligo/gioview/tabview"
	"github.com/oligo/gioview/theme"
	"github.com/oligo/gioview/view"
	gvwidget "github.com/oligo/gioview/widget"

	// gvwiget "github.com/oligo/gioview/widget"
	"golang.org/x/exp/shiny/materialdesign/icons"
	"looz.ws/typstify/i18n"
	"looz.ws/typstify/service"
	"looz.ws/typstify/service/bus"
	"looz.ws/typstify/typst/pkg"
	"looz.ws/typstify/ui/dialog"
	"looz.ws/typstify/ui/statusbar"
	"looz.ws/typstify/ui/viewer"
	"looz.ws/typstify/widgets"
)

type (
	C = layout.Context
	D = layout.Dimensions
)

const unknownCategory = "Other"

const (
	PublicSource   = "Public"
	LocalSource    = "Local"
	TypstifySource = "Typstify"
)

var (
	PkgListViewID = view.NewViewID("PkgListView")
	searchIcon, _ = widget.NewIcon(icons.ActionSearch)
)

var sources = []any{PublicSource, LocalSource, TypstifySource}

type PkgListView struct {
	*view.BaseView
	srv       *service.ServiceFacade
	vm        view.ViewManager
	cards     []*PkgCard
	refreshed bool

	sourceSelect *widgets.Dropdown

	tabs         *tabview.TabView
	packageList  *PkgList
	templateList *PkgList
	searchInput  gvwidget.TextField
	categoryList *CategoryList
}

func (vw *PkgListView) ID() view.ViewID {
	return PkgListViewID
}

func (vw *PkgListView) Title() string {
	return "Typst Packages"
}

func (vw *PkgListView) OnNavTo(intent view.Intent) error {
	vw.BaseView.OnNavTo(intent)
	vw.loadData(sources[0].(string))
	vw.tabs = tabview.NewTabView(layout.Horizontal, vw.buildTabItems()...)
	return nil
}

func (vw *PkgListView) Layout(gtx C, th *theme.Theme) D {
	vw.update(gtx)

	return layout.Inset{
		Top:    unit.Dp(36),
		Bottom: unit.Dp(36),
		Left:   unit.Dp(80),
	}.Layout(gtx, func(gtx C) D {
		return layout.Flex{
			Axis:      layout.Horizontal,
			Alignment: layout.Start,
		}.Layout(gtx,
			layout.Rigid(func(gtx C) D {
				gtx.Constraints.Max.X = gtx.Dp(unit.Dp(220))
				gtx.Constraints.Min.X = gtx.Constraints.Max.X

				return layout.Flex{
					Axis: layout.Vertical,
				}.Layout(gtx,
					layout.Rigid(func(gtx C) D {
						return layout.Inset{
							Right: unit.Dp(8),
						}.Layout(gtx, func(gtx C) D {
							vw.searchInput.SingleLine = true
							vw.searchInput.LabelOption = gvwidget.LabelOption{Alignment: gvwidget.Hidden}
							// vw.searchInput.MaxChars = 64
							vw.searchInput.Padding = unit.Dp(8)
							vw.searchInput.Leading = func(gtx C) D {
								return misc.Icon{Icon: searchIcon, Size: unit.Dp(18), Color: misc.WithAlpha(th.Fg, 0xb0)}.Layout(gtx, th)
							}
							return vw.searchInput.Layout(gtx, th, i18n.Translate("Search by name..."))
						})
					}),
					layout.Rigid(layout.Spacer{Height: unit.Dp(24)}.Layout),
					layout.Rigid(func(gtx C) D {
						return layout.Inset{
							Right: unit.Dp(8),
						}.Layout(gtx, func(gtx C) D {
							return layout.Flex{
								Axis: layout.Vertical,
							}.Layout(gtx,
								layout.Rigid(func(gtx C) D {
									lb := material.Subtitle2(th.Theme, i18n.Translate("Repository"))
									lb.Font.Weight = font.SemiBold
									return lb.Layout(gtx)
								}),
								layout.Rigid(layout.Spacer{Height: unit.Dp(10)}.Layout),

								layout.Rigid(func(gtx C) D {
									return vw.sourceSelect.Layout(gtx, th)
								}),
							)
						})
					}),

					layout.Rigid(layout.Spacer{Height: unit.Dp(24)}.Layout),
					layout.Rigid(func(gtx C) D {
						return vw.categoryList.Layout(gtx, th)
					}),
				)
			}),
			layout.Rigid(layout.Spacer{Width: unit.Dp(24)}.Layout),
			layout.Flexed(1, func(gtx C) D {
				return vw.tabs.Layout(gtx, th)
			}),
		)
	})

}

func (vw *PkgListView) update(gtx C) {
	if vw.sourceSelect == nil {
		vw.sourceSelect = widgets.NewDropDown(sources)
	}

	if vw.sourceSelect.Update(gtx) {
		vw.loadData(vw.sourceSelect.Value().(string))
	}

	if vw.refreshed {
		vw.categoryList.setCategories(extractCategories(vw.cards))
	}

	// packages and templates cards

	if vw.refreshed || vw.packageList == nil {
		cards1 := filterPkgs(vw.cards, func(card *PkgCard) bool {
			return card.pkgInfo.Template == nil
		})

		vw.packageList = newPkgList(cards1, vw.filter)
	}

	if vw.refreshed || vw.templateList == nil {
		cards2 := filterPkgs(vw.cards, func(card *PkgCard) bool {
			return card.pkgInfo.Template != nil
		})
		vw.templateList = newPkgList(cards2, vw.filter)
	}

	if vw.refreshed {
		vw.refreshed = false
	}

}

func (vw *PkgListView) loadData(source string) {
	go func() {
		vw.srv.EventBus().Emit(bus.TopicStatusbarNotifyEvent, statusbar.Notification{Content: i18n.Translate("Query packages...")})
		defer vw.srv.EventBus().Emit(bus.TopicStatusbarNotifyEvent, statusbar.Notification{Content: i18n.Translate("Typst packages info loaded."), Duration: 3 * time.Second})

		var pkgs []*pkg.TypstPkg
		var err error

		switch source {
		case PublicSource:
			pkgs, err = vw.srv.PkgService().RemotePublicPkgs()
		case LocalSource:
			pkgs, err = vw.srv.PkgService().LocalPkgs()
		case TypstifySource:
			err = errors.New("no data found")
		}

		if err != nil {
			vw.srv.EventBus().Emit(bus.TopicStatusbarNotifyEvent, statusbar.Notification{Content: i18n.Translate("Query packages failed: ") + err.Error()})
		}

		vw.cards = vw.cards[:0]
		for _, p := range pkgs {
			latestVer := p.Versions[0]
			var thumb *image.ImageSource
			if latestVer.Template != nil {
				if latestVer.ThumbUrl("small") != "" {
					thumb = image.ImageFromFile(latestVer.ThumbUrl("small"))
				}
			}

			card := newPkgCard(latestVer, thumb,
				func(imgPath string) {
					if imgPath != "" {
						intent := view.Intent{
							Target:      viewer.ImgViewerViewID,
							ShowAsModal: true,
							Params: map[string]interface{}{
								"path": imgPath,
							},
						}
						vw.vm.RequestSwitch(intent)
					}
				},
				func(pkgInfo *pkg.PackageInfo) {
					vw.initProject(pkgInfo)
				})

			vw.cards = append(vw.cards, card)
		}

		vw.refreshed = true
	}()
}

func (vw *PkgListView) initProject(pkgInfo *pkg.PackageInfo) {
	vw.vm.RequestSwitch(view.Intent{
		Target:      dialog.CreateProjectDialogViewID,
		ShowAsModal: true,
		Params: map[string]interface{}{
			"template": pkgInfo.ImportPath(),
		},
	})
}

func (vw *PkgListView) buildTabItems() []*tabview.TabItem {
	inset := layout.Inset{
		Left:   unit.Dp(12),
		Right:  unit.Dp(12),
		Top:    unit.Dp(8),
		Bottom: unit.Dp(8),
	}

	var tabItems []*tabview.TabItem
	tabItems = append(tabItems, tabview.SimpleTabItem(inset, i18n.Translate("Packages"), func(gtx C, th *theme.Theme) D {
		gtx.Constraints.Min.X = gtx.Constraints.Max.X
		return vw.packageList.Layout(gtx, th)
	}))

	tabItems = append(tabItems, tabview.SimpleTabItem(inset, i18n.Translate("Templates"), func(gtx C, th *theme.Theme) D {
		gtx.Constraints.Min.X = gtx.Constraints.Max.X
		return vw.templateList.Layout(gtx, th)
	}))

	return tabItems
}

func (vw *PkgListView) filter(card *PkgCard) bool {
	categories := vw.categoryList.getChecked()
	query := strings.TrimSpace(vw.searchInput.Text())

	if categories != nil {
		if len(card.pkgInfo.Categories) <= 0 && !slices.Contains(categories, unknownCategory) {
			return false
		}

		if len(card.pkgInfo.Categories) > 0 && !slices.ContainsFunc(categories, func(category string) bool {
			return slices.Contains(card.pkgInfo.Categories, category)
		}) {
			return false
		}
	}

	if query != "" && !strings.Contains(card.pkgInfo.Name, query) {
		return false
	}

	return true
}

func (vw *PkgListView) LayoutStatus(gtx C, th *theme.Theme) D {
	currentTab := vw.tabs.CurrentTab()
	var pkgStatus = ""
	if currentTab == 0 && vw.packageList != nil {
		pkgStatus = i18n.Translate("Found %d packages.", len(vw.packageList.cards))
	} else if currentTab == 1 && vw.templateList != nil {
		pkgStatus = i18n.Translate("Found %d templates.", len(vw.templateList.cards))
	}

	return material.Label(th.Theme, th.TextSize*0.9, pkgStatus).Layout(gtx)
}

func NewPkgListView(srv *service.ServiceFacade, vm view.ViewManager) view.View {
	return &PkgListView{
		BaseView:     &view.BaseView{},
		srv:          srv,
		vm:           vm,
		categoryList: newCategoryList(),
	}
}

func filterPkgs(cards []*PkgCard, filterFunc func(card *PkgCard) bool) []*PkgCard {
	filtered := make([]*PkgCard, 0)
	for _, c := range cards {
		if !filterFunc(c) {
			continue
		}

		filtered = append(filtered, c)
	}

	return filtered
}

func extractCategories(cards []*PkgCard) []string {
	categories := make([]string, 0)
	for _, c := range cards {
		categories = append(categories, c.pkgInfo.Categories...)
		if len(c.pkgInfo.Categories) <= 0 {
			categories = append(categories, unknownCategory)
		}
	}

	slices.Sort(categories)
	categories = slices.Compact(categories)
	return categories
}
