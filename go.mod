module github.com/solarlune/masterplan

go 1.13

require (
	github.com/adrg/xdg v0.2.3
	github.com/blang/semver v3.5.1+incompatible
	github.com/cavaliercoder/grab v2.0.0+incompatible
	github.com/dsnet/compress v0.0.1 // indirect
	github.com/faiface/beep v1.0.2
	github.com/gabriel-vasile/mimetype v1.3.0
	github.com/golang/snappy v0.0.1 // indirect
	github.com/mholt/archiver v3.1.1+incompatible
	github.com/ncruces/zenity v0.7.4
	github.com/nwaples/rardecode v1.0.0 // indirect
	github.com/otiai10/copy v1.2.0
	github.com/pierrec/lz4 v2.0.5+incompatible // indirect
	github.com/tanema/gween v0.0.0-20200427131925-c89ae23cc63c
	github.com/tidwall/gjson v1.6.0
	github.com/tidwall/sjson v1.1.1
	github.com/veandco/go-sdl2 v0.4.5
	github.com/xi2/xz v0.0.0-20171230120015-48954b6210f8 // indirect
	golang.design/x/clipboard v0.5.1
)

// The below line replaces the normal raylib-go dependency with my branch that has the config.h tweaked to
// remove screenshot-taking because we're do it manually in MasterPlan.
replace github.com/gen2brain/raylib-go => github.com/solarlune/raylib-go v0.0.0-20210122080031-04529085ce96
