package menu

import (
	"github.com/oligo/gioview/theme"

	"gioui.org/layout"
)

type ContextMenu struct {
	Menu
	contextArea ContextArea
	// position hint
	PositionHint layout.Direction
}

func NewContextMenu() *ContextMenu {
	m := &ContextMenu{
		Menu: newMenu(),
	}

	return m
}

func (m *ContextMenu) Layout(gtx C, th *theme.Theme) D {
	m.Update(gtx)

	return m.layout(gtx, th, m.contextArea.Layout)
}

// Update the state and reports if the menu is active.
func (m *ContextMenu) Update(gtx C) bool {
	m.contextArea.PositionHint = layout.E
	if m.contextArea.Activated() {
		m.onActivated(gtx)
	}

	if m.requestDismiss {
		m.contextArea.Dismiss()
		m.requestDismiss = false
	}

	if m.contextArea.Active() {
		m.update(gtx)
		return true
	}

	return false
}
