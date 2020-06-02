module github.com/solarlune/masterplan

go 1.13

require (
	github.com/atotto/clipboard v0.1.2
	github.com/blang/semver v3.5.1+incompatible
	github.com/chonla/roman-number-go v0.0.0-20181101035413-6768129de021
	github.com/faiface/beep v1.0.2
	github.com/gabriel-vasile/mimetype v1.1.0
	github.com/gen2brain/dlgs v0.0.0-20200211102745-b9c2664df42f
	github.com/gen2brain/raylib-go v0.0.0-20200528082952-e0f56b22753f
	github.com/hako/durafmt v0.0.0-20191009132224-3f39dc1ed9f4
	github.com/otiai10/copy v1.2.0
	github.com/pkg/browser v0.0.0-20180916011732-0a3d74bf9ce4
	github.com/tanema/gween v0.0.0-20200427131925-c89ae23cc63c
)

// The below line replaces the normal raylib-go dependency with a local one that has the config.h tweaked to allow for more image formats.
replace github.com/gen2brain/raylib-go => ../raylib-go-solarlune
