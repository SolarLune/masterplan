package main

import (
	"image"
	"image/color"
	"image/gif"

	rl "github.com/gen2brain/raylib-go/raylib"
)

type GifAnimation struct {
	Data         *gif.GIF
	Frames       []*rl.Image
	Delays       []float32 // 100ths of a second?
	CurrentFrame int
	Timer        float32
	frameImg     *image.RGBA
	DrawTexture  *rl.Texture2D
}

func NewGifAnimation(data *gif.GIF) *GifAnimation {
	tex := rl.LoadTextureFromImage(rl.NewImageFromImage(data.Image[0]))
	anim := &GifAnimation{Data: data, frameImg: image.NewRGBA(data.Image[0].Rect), DrawTexture: &tex}
	return anim
}

func (gifAnim *GifAnimation) IsEmpty() bool {
	// return true
	return gifAnim.Data == nil || len(gifAnim.Data.Image) == 0
}

func (gifAnim *GifAnimation) Update(dt float32) {

	gifAnim.Timer += dt
	if gifAnim.Timer >= gifAnim.Delays[gifAnim.CurrentFrame] {
		gifAnim.Timer -= gifAnim.Delays[gifAnim.CurrentFrame]
		gifAnim.CurrentFrame++
	}
	if gifAnim.CurrentFrame >= len(gifAnim.Data.Image) {
		gifAnim.CurrentFrame = 0
	}
}

func (gifAnim *GifAnimation) GetTexture() rl.Texture2D {

	if gifAnim.CurrentFrame == len(gifAnim.Frames) && len(gifAnim.Frames) < len(gifAnim.Data.Image) {

		// After decoding, we have to manually create a new image and plot each frame of the GIF because transparent GIFs
		// can only have frames that account for changed pixels (i.e. if you have a 320x240 GIF, but on frame
		// 17 only one pixel changes, the image generated for frame 17 will be 1x1 for Bounds.Size()).

		img := gifAnim.Data.Image[gifAnim.CurrentFrame]

		disposalMode := gifAnim.Data.Disposal[gifAnim.CurrentFrame]

		for y := 0; y < gifAnim.frameImg.Bounds().Size().Y; y++ {
			for x := 0; x < gifAnim.frameImg.Bounds().Size().X; x++ {
				if x >= img.Bounds().Min.X && x < img.Bounds().Max.X && y >= img.Bounds().Min.Y && y < img.Bounds().Max.Y {
					color := img.At(x, y)
					_, _, _, a := color.RGBA()
					if disposalMode != gif.DisposalNone || a >= 255 {
						gifAnim.frameImg.Set(x, y, color)
					}
				} else {
					if disposalMode == gif.DisposalBackground {
						gifAnim.frameImg.Set(x, y, color.RGBA{0, 0, 0, 0})
					} else if disposalMode == gif.DisposalPrevious && gifAnim.CurrentFrame > 0 {
						gifAnim.frameImg.Set(x, y, gifAnim.Data.Image[gifAnim.CurrentFrame-1].At(x, y))
					}
					// For gif.DisposalNone, it doesn't matter, I think?
					// For clarification on disposal method specs, see: https://www.w3.org/Graphics/GIF/spec-gif89a.txt
				}
			}

		}

		gifAnim.Frames = append(gifAnim.Frames, rl.NewImageFromImage(gifAnim.frameImg))
		gifAnim.Delays = append(gifAnim.Delays, float32(gifAnim.Data.Delay[gifAnim.CurrentFrame])/100)

	}

	if gifAnim.DrawTexture != nil {
		rl.UnloadTexture(*gifAnim.DrawTexture)
	}
	tex := rl.LoadTextureFromImage(gifAnim.Frames[gifAnim.CurrentFrame])
	gifAnim.DrawTexture = &tex
	return *gifAnim.DrawTexture

}

func (gifAnimation *GifAnimation) Destroy() {
	for _, frame := range gifAnimation.Frames {
		rl.UnloadImage(frame)
	}
	rl.UnloadTexture(*gifAnimation.DrawTexture)
}
