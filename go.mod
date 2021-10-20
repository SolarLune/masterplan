module github.com/solarlune/masterplan

go 1.13

require (
	github.com/adrg/xdg v0.3.4
	github.com/blang/semver v3.5.1+incompatible
	github.com/cavaliercoder/grab v2.0.0+incompatible
	github.com/dsnet/compress v0.0.1 // indirect
	github.com/faiface/beep v1.1.0
	github.com/frankban/quicktest v1.13.1 // indirect
	github.com/gabriel-vasile/mimetype v1.4.0
	github.com/gen2brain/beeep v0.0.0-20210529141713-5586760f0cc1 // indirect
	github.com/godbus/dbus/v5 v5.0.5 // indirect
	github.com/golang/snappy v0.0.4 // indirect
	github.com/gopherjs/gopherjs v0.0.0-20211014092231-cac8ae8ec3c8 // indirect
	github.com/hajimehoshi/go-mp3 v0.3.2 // indirect
	github.com/hajimehoshi/oto v1.0.1 // indirect
	github.com/hako/durafmt v0.0.0-20210608085754-5c1018a4e16b
	github.com/jfreymuth/oggvorbis v1.0.3 // indirect
	github.com/josephspurrier/goversioninfo v1.3.0 // indirect
	github.com/mewkiz/pkg v0.0.0-20210827150434-97fe13d38bc8 // indirect
	github.com/mholt/archiver v3.1.1+incompatible
	github.com/ncruces/zenity v0.7.12
	github.com/nwaples/rardecode v1.1.2 // indirect
	github.com/otiai10/copy v1.6.0
	github.com/pierrec/lz4 v2.6.1+incompatible // indirect
	github.com/tanema/gween v0.0.0-20200427131925-c89ae23cc63c
	github.com/tidwall/gjson v1.9.4
	github.com/tidwall/sjson v1.2.2
	github.com/ulikunitz/xz v0.5.10 // indirect
	github.com/veandco/go-sdl2 v0.4.10
	github.com/xi2/xz v0.0.0-20171230120015-48954b6210f8 // indirect
	golang.design/x/clipboard v0.5.3
	golang.org/x/exp v0.0.0-20211012155715-ffe10e552389 // indirect
	golang.org/x/mobile v0.0.0-20210924032853-1c027f395ef7 // indirect
	golang.org/x/net v0.0.0-20211013171255-e13a2654a71e // indirect
	golang.org/x/sys v0.0.0-20211015200801-69063c4bb744 // indirect
)

// The below line replaces the normal raylib-go dependency with my branch that has the config.h tweaked to
// remove screenshot-taking because we're do it manually in MasterPlan.
replace github.com/gen2brain/raylib-go => github.com/solarlune/raylib-go v0.0.0-20210122080031-04529085ce96
