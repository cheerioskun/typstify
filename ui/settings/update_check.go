package settings

import (
	"image/color"
	"log"

	"gioui.org/layout"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"github.com/dustin/go-humanize"
	"github.com/inkeliz/giohyperlink"
	"github.com/oligo/gioview/page"
	"github.com/oligo/gioview/theme"
	"looz.ws/typstify/i18n"
	"looz.ws/typstify/service"
	"looz.ws/typstify/service/net"
	"looz.ws/typstify/version"
)

type UpdateCheck struct {
	srv         *service.ServiceFacade
	newRelease  *net.ReleaseInfo
	checkBtn    widget.Clickable
	downloadBtn widget.Clickable
	err         error
}

func (u *UpdateCheck) Layout(gtx C, th *theme.Theme) D {
	u.Update(gtx)
	return layout.Flex{
		Axis: layout.Vertical,
	}.Layout(gtx,
		layout.Rigid(func(gtx C) D {
			return layout.Center.Layout(gtx, func(gtx C) D {
				gtx.Constraints.Min.X = 0
				btn := material.Button(th.Theme, &u.checkBtn, i18n.Translate("Check new version"))
				btn.Inset = layout.UniformInset(unit.Dp(6))
				return btn.Layout(gtx)
			})
		}),

		layout.Rigid(layout.Spacer{Height: unit.Dp(20)}.Layout),

		layout.Rigid(func(gtx C) D {
			return layoutErrorLabel(gtx, th, u.err)
		}),

		layout.Rigid(func(gtx C) D {
			return u.layoutReleaseInfo(gtx, th)
		}),
	)
}

func (u *UpdateCheck) layoutReleaseInfo(gtx C, th *theme.Theme) D {
	if u.newRelease == nil {
		return layout.Dimensions{}
	}

	return layout.Flex{
		Axis:      layout.Vertical,
		Alignment: layout.Middle,
	}.Layout(gtx,
		layout.Rigid(func(gtx C) D {
			return page.RowItem{Alignment: layout.Middle}.Layout(gtx, th, i18n.Translate("Version"), func(gtx C) D {
				label := material.Label(th.Theme, th.TextSize*0.9, u.newRelease.AppVersion)
				return label.Layout(gtx)
			})
		}),

		layout.Rigid(func(gtx C) D {
			return page.RowItem{Alignment: layout.Start}.Layout(gtx, th, i18n.Translate("Changelog"), func(gtx C) D {
				label := material.Label(th.Theme, th.TextSize*0.8, u.newRelease.Changelog)
				return label.Layout(gtx)
			})
		}),

		layout.Rigid(func(gtx C) D {
			return page.RowItem{Alignment: layout.Middle}.Layout(gtx, th, i18n.Translate("Release Time"), func(gtx C) D {
				label := material.Label(th.Theme, th.TextSize*0.9, humanize.Time(u.newRelease.CreatedAt))
				return label.Layout(gtx)
			})
		}),

		layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout),

		layout.Rigid(func(gtx C) D {
			return layout.Center.Layout(gtx, func(gtx C) D {
				gtx.Constraints.Min.X = 0
				return material.Button(th.Theme, &u.downloadBtn, i18n.Translate("Go to download")).Layout(gtx)
			})
		}),
	)
}

func (u *UpdateCheck) Update(gtx C) {
	if u.checkBtn.Clicked(gtx) {
		req := net.UpdateCheckReq{
			DeviceID:       u.srv.Settings().General().DeviceID,
			CurrentVersion: version.BinVersion,
			UseBeta:        false,
		}

		api := net.NewRemote()
		re, err := api.CheckUpdate(&req)
		if err != nil {
			u.err = err
			u.newRelease = nil
		} else {
			u.newRelease = re
			u.err = nil
		}
	}

	if u.downloadBtn.Clicked(gtx) && u.newRelease != nil {
		err := giohyperlink.Open("https://typstify.com/download")
		if err != nil {
			log.Printf("error: opening hyperlink: %v", err)
		}
		u.err = err
	}
}

func layoutErrorLabel(gtx C, th *theme.Theme, err error) D {
	if err != nil {
		return layout.Inset{
			Top:    unit.Dp(10),
			Bottom: unit.Dp(10),
			Left:   unit.Dp(15),
			Right:  unit.Dp(15),
		}.Layout(gtx, func(gtx C) D {
			label := material.Label(th.Theme, th.TextSize*0.8, err.Error())
			label.Color = color.NRGBA{R: 255, A: 255}
			label.Alignment = text.Middle
			return label.Layout(gtx)
		})
	} else {
		return layout.Dimensions{}
	}
}
