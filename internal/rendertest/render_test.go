package rendertest

import (
	"image"
	"image/color"
	"math"
	"testing"

	"golang.org/x/image/colornames"

	"gioui.org/f32"
	"gioui.org/internal/f32color"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
)

func TestTransformMacro(t *testing.T) {
	// testcase resulting from original bug when rendering layout.Stacked

	// Build clip-path.
	c := constSqPath()

	run(t, func(o *op.Ops) {

		// render the first Stacked item
		m1 := op.Record(o)
		dr := image.Rect(0, 0, 128, 50)
		paint.FillShape(o, black, clip.Rect(dr).Op())
		c1 := m1.Stop()

		// Render the second stacked item
		m2 := op.Record(o)
		paint.ColorOp{Color: red}.Add(o)
		// Simulate a draw text call
		stack := op.Save(o)
		op.Offset(f32.Pt(0, 10)).Add(o)

		// Apply the clip-path.
		c.Add(o)

		paint.PaintOp{}.Add(o)
		stack.Load()

		c2 := m2.Stop()

		// Call each of them in a transform
		s1 := op.Save(o)
		op.Offset(f32.Pt(0, 0)).Add(o)
		c1.Add(o)
		s1.Load()
		s2 := op.Save(o)
		op.Offset(f32.Pt(0, 0)).Add(o)
		c2.Add(o)
		s2.Load()
	}, func(r result) {
		r.expect(5, 15, colornames.Red)
		r.expect(15, 15, colornames.Black)
		r.expect(11, 51, colornames.White)
	})
}

func TestRepeatedPaintsZ(t *testing.T) {
	run(t, func(o *op.Ops) {
		// Draw a rectangle
		paint.FillShape(o, black, clip.Rect(image.Rect(0, 0, 128, 50)).Op())

		builder := clip.Path{}
		builder.Begin(o)
		builder.Move(f32.Pt(0, 0))
		builder.Line(f32.Pt(10, 0))
		builder.Line(f32.Pt(0, 10))
		builder.Line(f32.Pt(-10, 0))
		builder.Line(f32.Pt(0, -10))
		p := builder.End()
		clip.Outline{
			Path: p,
		}.Op().Add(o)
		paint.Fill(o, red)
	}, func(r result) {
		r.expect(5, 5, colornames.Red)
		r.expect(11, 15, colornames.Black)
		r.expect(11, 51, colornames.White)
	})
}

func TestNoClipFromPaint(t *testing.T) {
	// ensure that a paint operation does not pollute the state
	// by leaving any clip paths in place.
	run(t, func(o *op.Ops) {
		a := f32.Affine2D{}.Rotate(f32.Pt(20, 20), math.Pi/4)
		op.Affine(a).Add(o)
		paint.FillShape(o, red, clip.Rect(image.Rect(10, 10, 30, 30)).Op())
		a = f32.Affine2D{}.Rotate(f32.Pt(20, 20), -math.Pi/4)
		op.Affine(a).Add(o)

		paint.FillShape(o, black, clip.Rect(image.Rect(0, 0, 50, 50)).Op())
	}, func(r result) {
		r.expect(1, 1, colornames.Black)
		r.expect(20, 20, colornames.Black)
		r.expect(49, 49, colornames.Black)
		r.expect(51, 51, colornames.White)
	})
}

func TestDeferredPaint(t *testing.T) {
	run(t, func(o *op.Ops) {
		state := op.Save(o)
		clip.Rect(image.Rect(0, 0, 80, 80)).Op().Add(o)
		paint.ColorOp{Color: color.NRGBA{A: 0xff, G: 0xff}}.Add(o)
		paint.PaintOp{}.Add(o)

		op.Affine(f32.Affine2D{}.Offset(f32.Pt(20, 20))).Add(o)
		m := op.Record(o)
		clip.Rect(image.Rect(0, 0, 80, 80)).Op().Add(o)
		paint.ColorOp{Color: color.NRGBA{A: 0xff, R: 0xff, G: 0xff}}.Add(o)
		paint.PaintOp{}.Add(o)
		paintMacro := m.Stop()
		op.Defer(o, paintMacro)

		state.Load()
		op.Affine(f32.Affine2D{}.Offset(f32.Pt(10, 10))).Add(o)
		clip.Rect(image.Rect(0, 0, 80, 80)).Op().Add(o)
		paint.ColorOp{Color: color.NRGBA{A: 0xff, B: 0xff}}.Add(o)
		paint.PaintOp{}.Add(o)
	}, func(r result) {
	})
}

func constSqPath() op.CallOp {
	innerOps := new(op.Ops)
	m := op.Record(innerOps)
	builder := clip.Path{}
	builder.Begin(innerOps)
	builder.Move(f32.Pt(0, 0))
	builder.Line(f32.Pt(10, 0))
	builder.Line(f32.Pt(0, 10))
	builder.Line(f32.Pt(-10, 0))
	builder.Line(f32.Pt(0, -10))
	p := builder.End()
	clip.Outline{Path: p}.Op().Add(innerOps)
	return m.Stop()
}

func constSqCirc() op.CallOp {
	innerOps := new(op.Ops)
	m := op.Record(innerOps)
	clip.RRect{Rect: f32.Rect(0, 0, 40, 40),
		NW: 20, NE: 20, SW: 20, SE: 20}.Add(innerOps)
	return m.Stop()
}

func drawChild(ops *op.Ops, text op.CallOp) op.CallOp {
	r1 := op.Record(ops)
	text.Add(ops)
	paint.PaintOp{}.Add(ops)
	return r1.Stop()
}

func TestReuseStencil(t *testing.T) {
	txt := constSqPath()
	run(t, func(ops *op.Ops) {
		c1 := drawChild(ops, txt)
		c2 := drawChild(ops, txt)

		// lay out the children
		stack1 := op.Save(ops)
		c1.Add(ops)
		stack1.Load()

		stack2 := op.Save(ops)
		op.Offset(f32.Pt(0, 50)).Add(ops)
		c2.Add(ops)
		stack2.Load()
	}, func(r result) {
		r.expect(5, 5, colornames.Black)
		r.expect(5, 55, colornames.Black)
	})
}

func TestBuildOffscreen(t *testing.T) {
	// Check that something we in one frame build outside the screen
	// still is rendered correctly if moved into the screen in a later
	// frame.

	txt := constSqCirc()
	draw := func(off float32, o *op.Ops) {
		s := op.Save(o)
		op.Offset(f32.Pt(0, off)).Add(o)
		txt.Add(o)
		paint.PaintOp{}.Add(o)
		s.Load()
	}

	multiRun(t,
		frame(
			func(ops *op.Ops) {
				draw(-100, ops)
			}, func(r result) {
				r.expect(5, 5, colornames.White)
				r.expect(20, 20, colornames.White)
			}),
		frame(
			func(ops *op.Ops) {
				draw(0, ops)
			}, func(r result) {
				r.expect(2, 2, colornames.White)
				r.expect(20, 20, colornames.Black)
				r.expect(38, 38, colornames.White)
			}))
}

func TestNegativeOverlaps(t *testing.T) {
	run(t, func(ops *op.Ops) {
		clip.RRect{Rect: f32.Rect(50, 50, 100, 100)}.Add(ops)
		clip.Rect(image.Rect(0, 120, 100, 122)).Add(ops)
		paint.PaintOp{}.Add(ops)
	}, func(r result) {
		r.expect(60, 60, colornames.White)
		r.expect(60, 110, colornames.White)
		r.expect(60, 120, colornames.White)
		r.expect(60, 122, colornames.White)
	})
}

type Gradient struct {
	From, To color.NRGBA
}

var gradients = []Gradient{
	{From: color.NRGBA{R: 0x00, G: 0x00, B: 0x00, A: 0xFF}, To: color.NRGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xFF}},
	{From: color.NRGBA{R: 0x19, G: 0xFF, B: 0x19, A: 0xFF}, To: color.NRGBA{R: 0xFF, G: 0x19, B: 0x19, A: 0xFF}},
	{From: color.NRGBA{R: 0xFF, G: 0x19, B: 0x19, A: 0xFF}, To: color.NRGBA{R: 0x19, G: 0x19, B: 0xFF, A: 0xFF}},
	{From: color.NRGBA{R: 0x19, G: 0x19, B: 0xFF, A: 0xFF}, To: color.NRGBA{R: 0x19, G: 0xFF, B: 0x19, A: 0xFF}},
	{From: color.NRGBA{R: 0x19, G: 0xFF, B: 0xFF, A: 0xFF}, To: color.NRGBA{R: 0xFF, G: 0x19, B: 0x19, A: 0xFF}},
	{From: color.NRGBA{R: 0xFF, G: 0xFF, B: 0x19, A: 0xFF}, To: color.NRGBA{R: 0x19, G: 0x19, B: 0xFF, A: 0xFF}},
}

func TestLinearGradient(t *testing.T) {
	const halfpx = 0.5
	const gradienth = 8
	// 0.5 offset from ends to ensure that the center of the pixel
	// aligns with gradient from and to colors.

	stop1, stop2 := f32.Pt(32+halfpx, 4), f32.Pt(96-halfpx, 4)
	clipRange := f32.Rect(16, 0, 112, gradienth)

	type Sample struct {
		x int
		p float32
	}

	samples := []Sample{
		{x: 24, p: 0},
		{x: 32, p: 0},
		{x: 48, p: 0.25},
		{x: 64, p: 0.5},
		{x: 80, p: 0.75},
		{x: 96, p: 1},
		{x: 104, p: 1},
	}

	run(t, func(ops *op.Ops) {
		for i, g := range gradients {
			st := op.Push(ops)
			op.Affine(f32.Affine2D{}.Offset(f32.Pt(0, float32(i)*gradienth))).Add(ops)
			paint.LinearGradientOp{
				Stop1:  stop1,
				Color1: g.From,
				Stop2:  stop2,
				Color2: g.To,
			}.Add(ops)
<<<<<<< HEAD
			st := op.Save(ops)
			clip.RRect{Rect: gr}.Add(ops)
			op.Affine(f32.Affine2D{}.Offset(pixelAligned.Min)).Add(ops)
			scale(pixelAligned.Dx()/128, 1).Add(ops)
			paint.PaintOp{}.Add(ops)
			st.Load()
			gr = gr.Add(f32.Pt(0, gradienth))
		}
	}, func(r result) {
		gr := pixelAligned
		for _, g := range gradients {
			from := f32color.LinearFromSRGB(g.From)
			to := f32color.LinearFromSRGB(g.To)
			for _, p := range samples {
				exp := lerp(from, to, float32(p)/float32(r.img.Bounds().Dx()-1))
				r.expect(p, int(gr.Min.Y+gradienth/2), f32color.NRGBAToRGBA(exp.SRGB()))
=======
			clip.RRect{Rect: clipRange}.Add(ops)
			paint.PaintOp{}.Add(ops)
			st.Pop()
		}
	}, func(r result) {
		for i, g := range gradients {
			from := f32color.RGBAFromSRGB(g.From)
			to := f32color.RGBAFromSRGB(g.To)
			for _, s := range samples {
				exp := lerp(from, to, s.p)
				r.expect(s.x, int(i*gradienth+gradienth/2), exp.SRGB())
>>>>>>> 7184f65 (gpu: fix linear gradient)
			}
		}
	})
}

func TestLinearGradientAngled(t *testing.T) {
	run(t, func(ops *op.Ops) {
		paint.LinearGradientOp{
			Stop1:  f32.Pt(64, 64),
			Color1: black,
			Stop2:  f32.Pt(0, 0),
			Color2: red,
		}.Add(ops)
		st := op.Save(ops)
		clip.Rect(image.Rect(0, 0, 64, 64)).Add(ops)
		paint.PaintOp{}.Add(ops)
		st.Load()

		paint.LinearGradientOp{
			Stop1:  f32.Pt(64, 64),
			Color1: white,
			Stop2:  f32.Pt(128, 0),
			Color2: green,
		}.Add(ops)
		st = op.Save(ops)
		clip.Rect(image.Rect(64, 0, 128, 64)).Add(ops)
		paint.PaintOp{}.Add(ops)
		st.Load()

		paint.LinearGradientOp{
			Stop1:  f32.Pt(64, 64),
			Color1: black,
			Stop2:  f32.Pt(128, 128),
			Color2: blue,
		}.Add(ops)
		st = op.Save(ops)
		clip.Rect(image.Rect(64, 64, 128, 128)).Add(ops)
		paint.PaintOp{}.Add(ops)
		st.Load()

		paint.LinearGradientOp{
			Stop1:  f32.Pt(64, 64),
			Color1: white,
			Stop2:  f32.Pt(0, 128),
			Color2: magenta,
		}.Add(ops)
		st = op.Save(ops)
		clip.Rect(image.Rect(0, 64, 64, 128)).Add(ops)
		paint.PaintOp{}.Add(ops)
		st.Load()
	}, func(r result) {})
}

func TestLinearGradientStar(t *testing.T) {
	run(t, func(ops *op.Ops) {
		center := f32.Pt(64, 64)
		const N = 32
		for i := 0; i < N; i++ {
			stack := op.Push(ops)
			paint.LinearGradientOp{
				Stop1:  f32.Pt(72, 0),
				Color1: colornames.Red,
				Stop2:  f32.Pt(128, 0),
				Color2: colornames.Blue,
			}.Add(ops)
			clip.Rect(image.Rect(72, 62, 128, 66)).Add(ops)
			paint.PaintOp{}.Add(ops)
			stack.Pop()

			op.Affine(f32.Affine2D{}.Rotate(center, 2*math.Pi/N)).Add(ops)
		}
	}, func(r result) {
	})
}

func TestLinearGradientTransformedRotated(t *testing.T) {
	run(t, func(ops *op.Ops) {
		op.Affine(f32.Affine2D{}.Rotate(f32.Pt(64, 64), math.Pi/2)).Add(ops)

		paint.LinearGradientOp{
			Stop1:  f32.Pt(0, 0),
			Color1: colornames.Black,
			Stop2:  f32.Pt(128, 128),
			Color2: colornames.White,
		}.Add(ops)
		paint.PaintOp{}.Add(ops)

		stack := op.Push(ops)
		paint.ColorOp{Color: colornames.Red}.Add(ops)
		clip.Rect(image.Rect(0, 0, 8, 8)).Add(ops)
		paint.PaintOp{}.Add(ops)
		stack.Pop()

		stack = op.Push(ops)
		paint.ColorOp{Color: colornames.Green}.Add(ops)
		clip.Rect(image.Rect(120, 120, 128, 128)).Add(ops)
		paint.PaintOp{}.Add(ops)
		stack.Pop()

	}, func(r result) {
	})
}

// lerp calculates linear interpolation with color b and p.
func lerp(a, b f32color.RGBA, p float32) f32color.RGBA {
	return f32color.RGBA{
		R: a.R*(1-p) + b.R*p,
		G: a.G*(1-p) + b.G*p,
		B: a.B*(1-p) + b.B*p,
		A: a.A*(1-p) + b.A*p,
	}
}
