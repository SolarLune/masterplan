package main

import (
	"image/gif"

	"github.com/Zyko0/go-sdl3/sdl"
)

type GifAnimation struct {
	Data            *gif.GIF
	Frames          []*sdl.Surface
	Delays          []float32 // 100ths of a second?
	frameImg        *sdl.Surface
	Width           float32
	Height          float32
	progressChannel chan float32
	progress        float32
}

func NewGifAnimation(data *gif.GIF) *GifAnimation {

	surf, _ := sdl.CreateSurface(data.Image[0].Rect.Dx(), data.Image[0].Rect.Dy(), sdl.PIXELFORMAT_ARGB8888)

	anim := &GifAnimation{
		Data:            data,
		frameImg:        surf,
		Width:           float32(data.Image[0].Rect.Dx()),
		Height:          float32(data.Image[0].Rect.Dy()),
		progressChannel: make(chan float32, 1),
	}
	go anim.Load() // Load the frames in the background
	return anim
}

// Load loads the frames of the GIF animation.
func (gifAnim *GifAnimation) Load() {

	for index, img := range gifAnim.Data.Image {

		// After decoding, we have to manually create a new image and plot each frame of the GIF because transparent GIFs
		// can only have frames that account for changed pixels (i.e. if you have a 320x240 GIF, but on frame
		// 17 only one pixel changes, the image generated for frame 17 will be 1x1 for Bounds.Size()).

		prev := 0

		disposalMode := byte(gif.DisposalNone)
		if index > 0 {
			disposalMode = gifAnim.Data.Disposal[index-1]
		}

		// empty := sdl.RGBA8888{0, 0, 0, 0}

		w := gifAnim.frameImg.W
		h := gifAnim.frameImg.H

		for y := int32(0); y < h; y++ {

			for x := int32(0); x < w; x++ {

				// We clear each pixel of each frame, but only the pixels within the rectangle specified by the frame is plotted below, as
				// some frames of GIFs can have a "changed rectangle", indicating which pixels in which rectangle need to actually change.

				if disposalMode == gif.DisposalBackground || disposalMode == gif.DisposalPrevious {
					a, r, g, b := gifAnim.Data.Image[prev].At(int(x), int(y)).RGBA()
					gifAnim.frameImg.WritePixel(x, y, uint8(r), uint8(g), uint8(b), uint8(a))
				} else if disposalMode == gif.DisposalNone {
					r, g, b, a := gifAnim.Data.Image[0].At(int(x), int(y)).RGBA()
					gifAnim.frameImg.WritePixel(x, y, uint8(r), uint8(g), uint8(b), uint8(a))
				}

				if disposalMode != gif.DisposalPrevious {
					prev = index
				}

				color := img.RGBA64At(int(x), int(y))

				if x >= int32(img.Rect.Min.X) && x < int32(img.Rect.Max.X) && y >= int32(img.Rect.Min.Y) && y < int32(img.Rect.Max.Y) {

					// alpha is 65535 max
					if _, _, _, alpha := color.RGBA(); alpha > 128 {
						r, g, b, a := color.RGBA()
						gifAnim.frameImg.WritePixel(x, y, uint8(r), uint8(g), uint8(b), uint8(a))
					}

				}

			}

		}

		newSurf, err := gifAnim.frameImg.Duplicate()
		if err != nil {
			panic(err)
		}
		gifAnim.Frames = append(gifAnim.Frames, newSurf)

		delay := float32(gifAnim.Data.Delay[index]) / 100

		if delay <= 0 {
			delay = 0.1
		}

		gifAnim.Delays = append(gifAnim.Delays, delay)

		// If there's something in the progress channel, it's an old value indicating the progress of the
		// loading process, so we take it out.
		if len(gifAnim.progressChannel) > 0 {
			<-gifAnim.progressChannel
		}

		gifAnim.progressChannel <- float32(len(gifAnim.Frames)) / float32(len(gifAnim.Data.Image))

	}

}

func (gifAnim *GifAnimation) Destroy() {
	for _, frame := range gifAnim.Frames {
		frame.Destroy()
	}
}

func (gifAnim *GifAnimation) IsReady() bool {
	return gifAnim.LoadingProgress() >= 1
}

// LoadingProgress returns the progress at loading the GIF into memory as a fraction spanning 0 to 1.
func (gifAnim *GifAnimation) LoadingProgress() float32 {

	for len(gifAnim.progressChannel) > 0 {
		gifAnim.progress = <-gifAnim.progressChannel
	}
	return gifAnim.progress

}

type GifPlayer struct {
	Animation    *GifAnimation
	CurrentFrame int
	Timer        float32
	// FrameTex     *sdl.Texture
	Frames []*sdl.Texture
}

func NewGifPlayer(gifAnim *GifAnimation) *GifPlayer {

	return &GifPlayer{
		Animation: gifAnim,
	}

}

func (gifPlayer *GifPlayer) Update(dt float32) {

	gifPlayer.Timer += dt

	for gifPlayer.Timer >= gifPlayer.Animation.Delays[gifPlayer.CurrentFrame] {
		gifPlayer.Timer -= gifPlayer.Animation.Delays[gifPlayer.CurrentFrame]
		gifPlayer.CurrentFrame++
		if gifPlayer.CurrentFrame >= len(gifPlayer.Animation.Frames) {
			gifPlayer.CurrentFrame = 0
		}
	}

}

func (gifPlayer *GifPlayer) Destroy() {
	for _, frame := range gifPlayer.Frames {
		frame.Destroy()
	}
}

func (gifPlayer *GifPlayer) Texture() *sdl.Texture {
	for len(gifPlayer.Frames) <= gifPlayer.CurrentFrame {
		frame, _ := globals.Renderer.CreateTextureFromSurface(gifPlayer.Animation.Frames[gifPlayer.CurrentFrame])
		frame.SetScaleMode(sdl.SCALEMODE_NEAREST)
		gifPlayer.Frames = append(gifPlayer.Frames, frame)
	}
	return gifPlayer.Frames[gifPlayer.CurrentFrame]
}
