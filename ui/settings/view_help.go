package settings

import (
	"fmt"
	"image"
	"log"
	"time"

	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"github.com/inkeliz/giohyperlink"
	"looz.ws/typstify/i18n"
	"looz.ws/typstify/version"

	gvimg "github.com/oligo/gioview/image"
	"github.com/oligo/gioview/theme"
)

type AppMeta struct {
	desc        string
	version     string
	copyright   string
	contactMail string
	website     string
}

type HelpView struct {
	updateCheck *UpdateCheck
	websiteLink widget.Clickable
}

var (
	iconImg = gvimg.ImageFromBuf(version.AppIcon)
	appMeta = AppMeta{
		website:     "https://typstify.com",
		desc:        "An elegant and intuitive editor designed for Typst.",
		version:     version.VersionStr(),
		copyright:   fmt.Sprintf("Copyright © %d Typstify. All rights reserved.", time.Now().Year()),
		contactMail: "zhangzj33@gmail.com",
	}
)

func (va *HelpView) Title() string {
	return i18n.Translate("About")
}

func (va *HelpView) Layout(gtx layout.Context, th *theme.Theme) layout.Dimensions {
	va.update(gtx)

	gtx.Constraints.Min.X = gtx.Constraints.Max.X
	return layout.Center.Layout(gtx, func(gtx C) D {
		return layout.Flex{
			Axis:      layout.Vertical,
			Alignment: layout.Middle,
		}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				sz := 160
				gtx.Constraints = layout.Exact(image.Pt(sz, sz))
				return gvimg.ImageStyle{Src: iconImg}.Layout(gtx)
			}),

			layout.Rigid(layout.Spacer{Height: unit.Dp(20)}.Layout),
			layout.Rigid(func(gtx C) D {
				gtx.Constraints.Min.X = 0

				return layout.Flex{
					Axis:      layout.Vertical,
					Alignment: layout.Middle,
				}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						label := material.Label(th.Theme, th.TextSize, "Typstify")
						// label.Color = misc.WithAlpha(th.Palette.Fg, 0xf6)
						return label.Layout(gtx)
					}),
					layout.Rigid(layout.Spacer{Height: unit.Dp(10)}.Layout),

					layout.Rigid(func(gtx C) D {
						label := material.Label(th.Theme, th.TextSize*0.9, appMeta.desc)
						// label.Color = misc.WithAlpha(th.Palette.Fg, 0xbb)
						return label.Layout(gtx)
					}),
					layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout),

					layout.Rigid(func(gtx C) D {
						label := material.Label(th.Theme, th.TextSize*0.9, appMeta.version)
						// label.Color = misc.WithAlpha(th.Palette.Fg, 0xbb)
						return label.Layout(gtx)
					}),
					layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout),
					layout.Rigid(func(gtx C) D {
						label := material.Label(th.Theme, th.TextSize*0.9, appMeta.copyright)
						// label.Color = misc.WithAlpha(th.Palette.Fg, 0xbb)
						return label.Layout(gtx)
					}),

					layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout),
					layout.Rigid(func(gtx C) D {
						return material.Clickable(gtx, &va.websiteLink, func(gtx C) D {
							return layout.UniformInset(unit.Dp(8)).Layout(gtx, func(gtx C) D {
								label := material.Label(th.Theme, th.TextSize*0.9, appMeta.website)
								label.Color = th.ContrastBg
								return label.Layout(gtx)
							})
						})
					}),
				)
			}),
			// layout.Rigid(layout.Spacer{Height: unit.Dp(25)}.Layout),

			// layout.Rigid(func(gtx C) D {
			// 	return settingItem{}.Layout(gtx, th,
			// 		i18n.Translate("Check Update"), "",
			// 		func(gtx C) D {
			// 			return va.updateCheck.Layout(gtx, th)

			// 		})
			// }),
		)
	})

}

func (va *HelpView) update(gtx layout.Context) {
	if va.websiteLink.Clicked(gtx) {
		if err := giohyperlink.Open(appMeta.website); err != nil {
			log.Printf("error: opening hyperlink: %v", err)
		}
	}
}
