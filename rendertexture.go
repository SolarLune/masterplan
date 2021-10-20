package main

import "github.com/veandco/go-sdl2/sdl"

var renderTextures = []*RenderTexture{}

type RenderTexture struct {
	Texture    *sdl.Texture
	Size       Point
	RenderFunc func(*RenderTexture)
}

// func (rt *RenderTexture) Destroy() {
// 	for i, t := range renderTextures {
// 		if t == rt {
// 			renderTextures[i] = nil
// 			renderTextures = append(renderTextures[:i], renderTextures[i+1:]...)
// 			break
// 		}
// 	}
// 	rt.Texture.Destroy()
// }

func (rt *RenderTexture) Rerender(newW, newH int32) {

	rt.Size.X = float32(newW)
	rt.Size.Y = float32(newH)

	if rt.Texture != nil {
		rt.Texture.Destroy()
	}

	newTex, err := globals.Renderer.CreateTexture(sdl.PIXELFORMAT_RGBA8888, sdl.TEXTUREACCESS_TARGET, int32(rt.Size.X), int32(rt.Size.Y))
	if err != nil {
		panic(err)
	}

	rt.Texture = newTex

	rt.RenderFunc(rt)

}

// NewRenderTexture creates a new *RenderTexture. However, it does NOT return the RenderTexture created; instead, it allows you to specify
// the width and height of the Texture, as well as a function to be called when the Texture needs to be rendered (i.e. directly after calling
// NewRenderTexture(), as well as whenever SDL loses context and render textures need to be rebuilt (see: https://wiki.libsdl.org/SDL_EventType, SDL_RENDER_TARGETS_RESET)).
// This is a bit of a doozy.
func NewRenderTexture(w, h int32, renderFunc func(rt *RenderTexture)) {

	rt := &RenderTexture{
		Size:       Point{float32(w), float32(h)},
		RenderFunc: renderFunc,
	}

	renderTextures = append(renderTextures, rt)

	rt.Rerender(w, h)

}
