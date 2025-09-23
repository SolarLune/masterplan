package main

import (
	"github.com/Zyko0/go-sdl3/sdl"
)

func RectIntersecting(a, b *sdl.FRect) bool {

	return !(a.X > b.X+b.W || a.Y > b.Y+b.H || a.X+a.W < b.X || a.Y+a.H < b.Y)

	// if ra.X < rb.X {
	// 	ra.X = rb.X
	// }
	// if ra.Y < rb.Y {
	// 	ra.Y = rb.Y
	// }
	// if ra.X+ra.W > rb.X+rb.W {
	// 	ra.W = rb.X + rb.W - ra.X
	// }
	// if ra.Y+ra.H > rb.Y+rb.H {
	// 	ra.H = rb.Y + rb.H - ra.Y
	// }
	// // Letting r0 and s0 be the values of r and s at the time that the method
	// // is called, this next line is equivalent to:
	// //
	// // if max(r0.Min.X, s0.Min.X) >= min(r0.Max.X, s0.Max.X) || likewiseForY { etc }
	// return ra.W > 0 && ra.H > 0

}
