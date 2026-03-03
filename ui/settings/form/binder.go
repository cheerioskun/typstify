package form

import (
	"math"

	"gioui.org/layout"
	"gioui.org/widget"
)

type FloatBinder struct {
	w widget.Float
	// value in range [0, 1]
	value      float32
	valueRange [2]float32
}

func (b *FloatBinder) Update(gtx layout.Context) (float32, bool) {
	var updated bool
	if b.w.Update(gtx) {
		b.value = b.w.Value
		updated = true
	}

	return b.Value(), updated
}

// FIXME: may not be reasonable to round the value to a integer?
func (b *FloatBinder) Value() float32 {
	val := float32(math.Round(float64(b.value*(b.valueRange[1]-b.valueRange[0]) + b.valueRange[0])))
	return val
}

func (b *FloatBinder) GetWidget(gtx layout.Context) *widget.Float {
	// sync from widget to value
	b.Update(gtx)
	// sync from value to widget
	if b.value != b.w.Value {
		b.w.Value = b.value
	}

	return &b.w
}

func NewFloatBinder(initVal float32, valRange []float32) *FloatBinder {
	if len(valRange) < 2 || valRange[0] >= valRange[1] {
		return nil
	}

	if initVal < valRange[0] {
		initVal = valRange[0]
	}
	if initVal > valRange[1] {
		initVal = valRange[1]
	}

	return &FloatBinder{
		w:          widget.Float{},
		value:      (initVal - valRange[0]) / (valRange[1] - valRange[0]),
		valueRange: [2]float32(valRange),
	}
}
