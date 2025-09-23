package main

import (
	"bufio"
	"image/gif"
	"log"
	"math"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Zyko0/go-sdl3/img"
	"github.com/Zyko0/go-sdl3/sdl"
	"github.com/cavaliergopher/grab/v3"
	"github.com/faiface/beep"
	"github.com/faiface/beep/flac"
	"github.com/faiface/beep/mp3"
	"github.com/faiface/beep/vorbis"
	"github.com/faiface/beep/wav"
	"github.com/gabriel-vasile/mimetype"
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

	for resourceName, resource := range resourceBank {
		if resource.Destructible {
			resource.Destroy()
			delete(resourceBank, resourceName)
		}
	}

}

type Resource struct {
	Name          string // The ID / name identifying the Resource; for offline files, this is the same as LocalFilepath
	LocalFilepath string // The actual path to the file on-disk
	Extension     string
	Data          interface{} // The data the resource represents; this might be an image, a sound stream, etc.
	MimeType      string
	Response      *grab.Response
	TempFile      bool // Whether the file is temporary (and so should be deleted after MasterPlan closes) - downloaded images, for example, are temporary
	SaveFile      bool // Whether the file should be saved along with the project - pasted screenshots, as an example, are saved
	Parsed        bool
	Destructible  bool // System resources aren't able to be deleted
}

func NewResource(resourcePath string) (*Resource, error) {

	resource := &Resource{
		Name:         resourcePath,
		Destructible: true,
	}

	if _, err := os.ReadFile(resourcePath); err == nil {

		resource.LocalFilepath = resourcePath
		resource.Extension = filepath.Ext(resourcePath)
		resource.Parse()

	} else {

		log.Println("possible online resource: ", resourcePath)

		project := globals.Project

		if globals.NextProject != nil {
			project = globals.NextProject
		}

		destDir := project.Properties.Get(ProjectCacheDirectory).AsString()

		if destDir == "" || !FolderExists(destDir) {
			destDir = filepath.Join(os.TempDir(), "masterplan")
			resource.TempFile = true
			// It's online, so we don't need to save it
		}

		if req, err := grab.NewRequest("", resourcePath); err != nil {
			return nil, err
		} else {
			unescapedPath, _ := url.QueryUnescape(req.URL().Path)
			req.Filename = filepath.Join(destDir, filepath.FromSlash(req.URL().Hostname()+"/"+unescapedPath))

			// It's already been downloaded
			if FileExists(req.Filename) {
				resource.LocalFilepath = req.Filename
				resource.Extension = filepath.Ext(resource.LocalFilepath)
				resource.Parse()
				return resource, nil
			}

			resource.LocalFilepath = req.Filename
			resource.Extension = filepath.Ext(resourcePath)
			resource.Response = globals.GrabClient.Do(req) // This can take time, up to and including seconds to execute

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

			internalSizeSetting := globals.Settings.Get(SettingsMaxInternalImageSize).AsString()

			internalSizeMax := int32(256)
			switch internalSizeSetting {
			case ImageBufferSize512:
				internalSizeMax = 512
			case ImageBufferSize1024:
				internalSizeMax = 1024
			case ImageBufferSize2048:
				internalSizeMax = 2048
			case ImageBufferSize4096:
				internalSizeMax = 4096
			case ImageBufferSize8192:
				internalSizeMax = 8192
			case ImageBufferSizeMax:
				internalSizeMax = math.MaxInt32
			}

			if maxSize := SmallestRendererMaxTextureSize(); internalSizeMax > maxSize {
				internalSizeMax = maxSize
			}

			w := surface.W
			h := surface.H

			// Image is too big to be displayed on our graphics card, we have to resize it
			if w > internalSizeMax || h > internalSizeMax {

				asr := float64(h) / float64(w)
				if w > h {
					w = internalSizeMax
					h = int32(math.Ceil(float64(w) * asr))
				} else if h > w {
					h = internalSizeMax
					w = int32(math.Ceil(float64(h) / asr))
				} else {
					w = internalSizeMax
					h = internalSizeMax
				}

				newSurf, _ := sdl.CreateSurface(int(w), int(h), surface.Format)

				surface.BlitScaled(nil, newSurf, &sdl.Rect{0, 0, w, h}, sdl.SCALEMODE_NEAREST)

				ogSurf := surface
				surface = newSurf
				ogSurf.Destroy()

			}

			if err != nil {
				panic(err)
			}
			// defer surface.Free()

			texture, err := globals.Renderer.CreateTextureFromSurface(surface)
			if err != nil {
				panic(err)
			}
			texture.SetBlendMode(sdl.BLENDMODE_BLEND)
			texture.SetScaleMode(sdl.SCALEMODE_NEAREST)

			resource.Data = Image{
				Size:    Vector{float32(surface.W), float32(surface.H)},
				Texture: texture,
				// Surface: surface,
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

var playingSounds = map[string]time.Time{}

func (resource *Resource) AsNewSound(limitPlayback bool, channel int) (*Sound, error) {

	if limitPlayback {
		if s, ok := playingSounds[resource.LocalFilepath]; ok && time.Since(s) < time.Millisecond*100 {
			return nil, nil
		}
	}

	originalFile, err := os.Open(resource.LocalFilepath)
	if err != nil {
		return nil, err
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
		return nil, err
	}

	s := NewSound(originalStream, format, channel)
	s.filepath = resource.LocalFilepath
	s.limitPlayback = limitPlayback
	return s, nil
}

func (resource *Resource) Destroy() {

	if resource.TempFile {
		os.Remove(resource.LocalFilepath)
	}

	if resource.IsTexture() {
		resource.AsImage().Texture.Destroy()
	}

	if resource.IsGIF() {
		resource.AsGIF().Destroy()
	}

	resource.Data = nil

}
