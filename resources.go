package main

import (
	"bufio"
	"image/gif"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/cavaliergopher/grab/v3"
	"github.com/faiface/beep"
	"github.com/faiface/beep/flac"
	"github.com/faiface/beep/mp3"
	"github.com/faiface/beep/vorbis"
	"github.com/faiface/beep/wav"
	"github.com/gabriel-vasile/mimetype"
	"github.com/veandco/go-sdl2/img"
	"github.com/veandco/go-sdl2/sdl"
)

type ResourceBank map[string]*Resource

func NewResourceBank() ResourceBank {
	return ResourceBank{}
}

func (resourceBank ResourceBank) Get(resourcePath string) *Resource {

	resource, exists := resourceBank[resourcePath]

	if !exists {
		res, err := NewResource(resourcePath)

		if err != nil {
			return nil
		}

		resource = res

		resourceBank[resourcePath] = res

	}

	return resource

}

func (resourceBank ResourceBank) Destroy() {

	for _, resource := range resourceBank {
		resource.Destroy()
	}

}

type Resource struct {
	Name          string // The ID / name identifying the Resource; for offline files, this is the same as LocalFilepath
	LocalFilepath string // The actual path to the file on-disk
	Extension     string
	Data          interface{} // The data the resource represents; this might be an image, a sound stream, etc.
	MimeType      string
	Response      *grab.Response
	TempFile      bool
	Parsed        bool
}

func NewResource(resourcePath string) (*Resource, error) {

	resource := &Resource{
		Name: resourcePath,
	}

	if _, err := os.ReadFile(resourcePath); err == nil {

		resource.LocalFilepath = resourcePath
		resource.Extension = filepath.Ext(resourcePath)
		resource.Parse()

	} else {

		log.Println("possible online resource")

		destDir := globals.Settings.Get(SettingsDownloadDirectory).AsString()
		if destDir == "" {
			destDir = filepath.Join(os.TempDir(), "masterplan")
			resource.TempFile = true
		}

		if req, err := grab.NewRequest("", resourcePath); err != nil {
			return nil, err
		} else {
			unescapedPath, _ := url.QueryUnescape(req.URL().Path)
			req.Filename = filepath.Join(destDir, filepath.FromSlash(req.URL().Hostname()+"/"+unescapedPath))
			resource.LocalFilepath = req.Filename
			resource.Extension = filepath.Ext(resourcePath)
			resource.Response = globals.GrabClient.Do(req)

			if resource.Response.IsComplete() && resource.Response.Err() != nil {
				return nil, resource.Response.Err()
			} else if site, err := http.Get(resourcePath); err == nil {

				// We're going to get what the Mime data is by downloading the first 5kb of a link, and then passing
				// it into mimetype.Detect().

				r := bufio.NewScanner(site.Body)
				r.Buffer([]byte{}, 5000)
				r.Scan()

				resource.MimeType = mimetype.Detect(r.Bytes()).String()

			}

		}

	}

	return resource, nil

}

func (resource *Resource) Parse() {

	// If the resource has already been parsed, then we can just skip it
	if resource.Parsed {
		return
	}

	mime, _ := mimetype.DetectFile(resource.LocalFilepath)
	resource.MimeType = mime.String()

	// if data, err := os.ReadFile(resource.LocalFilepath); err == nil {
	// 	// We use mimetype because http.DetectContentType doesn't detect mp3 as being an audio file somehow
	// 	resource.MimeType = mimetype.Detect(data).String()
	// }

	isTGA := resource.Extension == ".tga"

	if isTGA || strings.Contains(resource.MimeType, "image") {

		if strings.Contains(resource.MimeType, "gif") {

			data, err := os.Open(resource.LocalFilepath)

			if err != nil {
				panic(err)
			}

			gifAnim, err := gif.DecodeAll(data)

			if err != nil {
				panic(err)
			}

			resource.Data = NewGifAnimation(gifAnim)

		} else {

			surface, err := img.Load(resource.LocalFilepath)
			if err != nil {
				panic(err)
			}
			defer surface.Free()

			texture, err := globals.Renderer.CreateTextureFromSurface(surface)
			if err != nil {
				panic(err)
			}
			texture.SetBlendMode(sdl.BLENDMODE_BLEND)

			resource.Data = Image{
				Size:    Point{float32(surface.W), float32(surface.H)},
				Texture: texture,
			}

		}

	} else if strings.Contains(resource.MimeType, "audio") {

		// Sounds aren't shared, actually, so Resource.Data is nil for audio files.

	} else {
		globals.EventLog.Log("Warning: could not parse resource: %s", true, resource.Name)
	}

	resource.Parsed = true

}

// DownloadPercentage returns 0-1 as the Resource downloads, until it's finished downloading. Sometimes the download percentage is -1
// for some things (gifer does this, for example).
func (resource *Resource) DownloadPercentage() float64 {
	if resource.Response == nil {
		return 1
	} else if resource.Response.Size() > 0 {
		return resource.Response.Progress()
	} else if resource.Response.IsComplete() {
		return 1
	}

	return -1

}

func (resource *Resource) FinishedDownloading() bool {
	return resource.Response == nil || resource.DownloadPercentage() >= 1
}

func (resource *Resource) IsTexture() bool {
	if resource.FinishedDownloading() {
		resource.Parse()
		return resource.Extension == ".tga" || resource.MimeType != "image/gif" && strings.Contains(resource.MimeType, "image")
	}
	return false
}

func (resource *Resource) AsImage() Image {
	resource.Parse()
	return resource.Data.(Image)
}

func (resource *Resource) IsGIF() bool {
	if resource.FinishedDownloading() {
		resource.Parse()
		return strings.Contains(resource.MimeType, "gif")
	}
	return false
}

func (resource *Resource) AsGIF() *GifAnimation {
	resource.Parse()
	return resource.Data.(*GifAnimation)
}

func (resource *Resource) IsSound() bool {
	if resource.FinishedDownloading() {
		resource.Parse()
		return strings.Contains(resource.MimeType, "audio")
	}
	return false
}

func (resource *Resource) AsNewSound() *Sound {

	originalFile, err := os.Open(resource.LocalFilepath)
	if err != nil {
		panic(err)
	}

	var originalStream beep.StreamSeekCloser
	var format beep.Format

	if resource.MimeType == "audio/mpeg" {
		originalStream, format, err = mp3.Decode(originalFile)
	} else if resource.MimeType == "audio/wav" {
		originalStream, format, err = wav.Decode(originalFile)
	} else if resource.MimeType == "audio/flac" {
		originalStream, format, err = flac.Decode(originalFile)
	} else if strings.Contains(resource.MimeType, "ogg") {
		originalStream, format, err = vorbis.Decode(originalFile)
	}

	if err != nil {
		panic(err)
	}

	return NewSound(originalStream, format)
}

func (resource *Resource) Destroy() {

	if resource.TempFile {
		os.Remove(resource.LocalFilepath)
	}

	if resource.IsTexture() {
		resource.AsImage().Texture.Destroy()
	}
}
