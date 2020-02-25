module github.com/solarlune/masterplan

go 1.13

require (
	github.com/atotto/clipboard v0.1.2
	github.com/blang/semver v3.5.1+incompatible
	github.com/chonla/roman-number-go v0.0.0-20181101035413-6768129de021
	github.com/faiface/beep v1.0.2
	github.com/gabriel-vasile/mimetype v1.0.2
	github.com/gen2brain/dlgs v0.0.0-20191023125645-8832682fb7f8
	github.com/gen2brain/raylib-go v0.0.0-20191004100518-02424e2e10ea
	github.com/gopherjs/gopherjs v0.0.0-20180825215210-0210a2f0f73c
	github.com/hako/durafmt v0.0.0-20191009132224-3f39dc1ed9f4
	github.com/otiai10/copy v1.0.2
	github.com/pkg/browser v0.0.0-20180916011732-0a3d74bf9ce4
	golang.org/x/exp v0.0.0-20180710024300-14dda7b62fcd
)

// The below line replaces the normal raylib-go dependency with a local one that has the config.h tweaked to allow for more image formats.
replace github.com/gen2brain/raylib-go => ../raylib-go-solarlune
