module github.com/solarlune/masterplan

go 1.13

require (
	github.com/adrg/xdg v0.2.3
	github.com/atotto/clipboard v0.1.4 // indirect
	github.com/blang/semver v3.5.1+incompatible
	github.com/chonla/roman-number-go v0.0.0-20181101035413-6768129de021 // indirect
	github.com/dsnet/compress v0.0.1 // indirect
	github.com/gen2brain/raylib-go v0.0.0-20200528082952-e0f56b22753f
	github.com/golang/snappy v0.0.1 // indirect
	github.com/kr/pretty v0.1.0 // indirect
	github.com/kvartborg/vector v0.0.0-20210122071920-91df40ba4054 // indirect
	github.com/mholt/archiver v3.1.1+incompatible
	github.com/ncruces/zenity v0.7.4 // indirect
	github.com/nwaples/rardecode v1.0.0 // indirect
	github.com/otiai10/copy v1.2.0
	github.com/pierrec/lz4 v2.0.5+incompatible // indirect
	github.com/tanema/gween v0.0.0-20200427131925-c89ae23cc63c
	github.com/tidwall/gjson v1.6.0
	github.com/tidwall/sjson v1.1.1
	github.com/veandco/go-sdl2 v0.4.5
	github.com/xi2/xz v0.0.0-20171230120015-48954b6210f8 // indirect
	gopkg.in/check.v1 v1.0.0-20180628173108-788fd7840127 // indirect
)

// The below line replaces the normal raylib-go dependency with my branch that has the config.h tweaked to
// remove screenshot-taking because we're do it manually in MasterPlan.
replace github.com/gen2brain/raylib-go => github.com/solarlune/raylib-go v0.0.0-20210122080031-04529085ce96
