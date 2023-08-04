package main

import (
	"github.com/veandco/go-sdl2/sdl"
)

var renderTextures = []*RenderTexture{}

type RenderTexture struct {
	Image
	RenderFunc func()
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

func (rt *RenderTexture) Destroy() {

	if rt.Texture != nil {
		rt.Texture.Destroy()
	}
	rt.Texture = nil

}

// StopTracking removes the RenderTexture from the global renderTextures slice, which means that it won't be refreshed when a windowing event requires it (which
// is specifically applicable on Windows).
func (rt *RenderTexture) StopTracking() {

	for i, slot := range renderTextures {
		if rt == slot {
			renderTextures[i] = nil
			renderTextures = append(renderTextures[:i], renderTextures[i+1:]...)
			break
		}
	}

}

func (rt *RenderTexture) Recreate(newW, newH int32) {

	if newW > globals.RendererInfo.MaxTextureWidth {
		newW = globals.RendererInfo.MaxTextureWidth
	}

	if newH > globals.RendererInfo.MaxTextureHeight {
		newH = globals.RendererInfo.MaxTextureHeight
	}

	// SDL Texture size can't be 0x0 (this should never happen, but it seems to be happening on Windows periodically for certain users?)
	if newW < 2 {
		newW = 2
	}
	if newH < 2 {
		newH = 2
	}

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

}

// NewRenderTexture creates a new *RenderTexture. However, it does NOT return the RenderTexture created; instead, it allows you to specify
// a function to be called when the Texture needs to be rendered (i.e. directly after calling ewRenderTexture(), as well as whenever SDL loses context
// and render textures need to be rebuilt (see: https://wiki.libsdl.org/SDL_EventType, SDL_RENDER_TARGETS_RESET)). This is a bit of a doozy.
func NewRenderTexture() *RenderTexture {

	rt := &RenderTexture{}

	renderTextures = append(renderTextures, rt)

	return rt

}
