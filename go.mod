module github.com/solarlune/masterplan

go 1.13

require (
	github.com/atotto/clipboard v0.1.2
	github.com/blang/semver v3.5.1+incompatible
	github.com/chonla/roman-number-go v0.0.0-20181101035413-6768129de021
	github.com/dsnet/compress v0.0.1 // indirect
	github.com/faiface/beep v1.0.2
	github.com/gabriel-vasile/mimetype v1.1.0
	github.com/gen2brain/raylib-go v0.0.0-20200528082952-e0f56b22753f
	github.com/golang/snappy v0.0.1 // indirect
	github.com/goware/urlx v0.3.1
	github.com/hako/durafmt v0.0.0-20191009132224-3f39dc1ed9f4
	github.com/mholt/archiver v3.1.1+incompatible
	github.com/ncruces/zenity v0.4.2
	github.com/nwaples/rardecode v1.0.0 // indirect
	github.com/otiai10/copy v1.2.0
	github.com/pierrec/lz4 v2.0.5+incompatible // indirect
	github.com/pkg/browser v0.0.0-20180916011732-0a3d74bf9ce4
	github.com/tanema/gween v0.0.0-20200427131925-c89ae23cc63c
	github.com/tidwall/gjson v1.6.0
	github.com/tidwall/sjson v1.1.1
	github.com/xi2/xz v0.0.0-20171230120015-48954b6210f8 // indirect
)

// The below line replaces the normal raylib-go dependency with my branch that has the config.h tweaked to
// remove screenshot-taking because we're do it manually in MasterPlan.
replace github.com/gen2brain/raylib-go => github.com/solarlune/raylib-go v0.0.0-20200921060307-652b10c4b3d8
