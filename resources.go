package main

import (
	"image/gif"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/cavaliercoder/grab"
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

func (resourceBank ResourceBank) Destroy(resource *Resource) {
	resource.Destroy()
	delete(resourceBank, resource.Name)
}

type Resource struct {
	Name          string      // The ID / name identifying the Resource; for offline files, this is the same as LocalFilepath
	LocalFilepath string      // The actual path to the file on-disk; for
	Data          interface{} // The data the resource represents; this might be an image, a sound stream, etc.
	MimeType      string
	Response      *grab.Response
	Parsed        bool
}

func NewResource(resourcePath string) (*Resource, error) {

	resource := &Resource{
		Name: resourcePath,
	}

	if _, err := os.ReadFile(resourcePath); err == nil {

		resource.LocalFilepath = resourcePath
		resource.Parse()

	} else {

		log.Println("possible online resource")

		destDir := globals.OldProgramSettings.DownloadDirectory
		if destDir == "" {
			destDir = filepath.Join(os.TempDir(), "masterplan")
		}

		if req, err := grab.NewRequest("", resourcePath); err != nil {
			return nil, err
		} else {
			unescapedPath, _ := url.PathUnescape(req.URL().Path)
			req.Filename = filepath.Join(destDir, filepath.FromSlash(req.URL().Hostname()+"/"+unescapedPath))
			resource.LocalFilepath = req.Filename
			resource.Response = globals.GrabClient.Do(req)
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

	if strings.Contains(resource.MimeType, "image") {

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
		globals.EventLog.Log("Warning: could not parse resource: %s", resource.Name)
	}

	resource.Parsed = true

}

// LoadingPercentage returns 0-1 as the Resource loads, until it's finished loading.
func (resource *Resource) LoadingPercentage() float64 {
	if resource.Response == nil {
		return 1
	} else if resource.Response.Size > 0 {
		return resource.Response.Progress()
	} else {
		return -1
	}
}

func (resource *Resource) FinishedLoading() bool {
	return resource.Response == nil || resource.LoadingPercentage() >= 1
}

func (resource *Resource) IsTexture() bool {
	if resource.FinishedLoading() {
		resource.Parse()
		return resource.MimeType != "image/gif" && strings.Contains(resource.MimeType, "image")
	}
	return false
}

func (resource *Resource) AsImage() Image {
	resource.Parse()
	return resource.Data.(Image)
}

func (resource *Resource) IsGIF() bool {
	if resource.FinishedLoading() {
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
	if resource.FinishedLoading() {
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
	if resource.IsTexture() {
		resource.AsImage().Texture.Destroy()
	}
}

// import (
// 	"errors"
// 	"image/gif"
// 	"os"
// 	"path/filepath"
// 	"strings"
// 	"time"

// 	"github.com/cavaliercoder/grab"
// 	"github.com/faiface/beep"
// 	"github.com/faiface/beep/flac"
// 	"github.com/faiface/beep/mp3"
// 	"github.com/faiface/beep/vorbis"
// 	"github.com/faiface/beep/wav"
// 	"github.com/gabriel-vasile/mimetype"
// 	rl "github.com/gen2brain/raylib-go/raylib"
// )

// const (
// 	RESOURCE_STATE_DOWNLOADING = iota
// 	RESOURCE_STATE_LOADING
// 	RESOURCE_STATE_READY
// 	RESOURCE_STATE_DELETED
// )

// type Resource struct {
// 	// Modified time of the local filepath resource
// 	ModTime time.Time

// 	// Size of the resource on disk
// 	Size int64
// 	// Path facing the object requesting the resouce (e.g. "~/home/pictures/test.png" or "https://solarlune.com/media/bartender.png")
// 	ResourcePath string

// 	// Where the resource is located on disk (e.g. "~/home/pictures/test.png" or "/tmp/masterplan_resource076755900.png", for downloaded resources)
// 	LocalFilepath string

// 	// Pointer to the data the resource stands for (e.g. a rl.Texture2D for an image)
// 	Data interface{}

// 	// If the resource was downloaded via a URL, this points to the *grab.Response used to load the data.
// 	DownloadResponse *grab.Response

// 	// MIME data for the Resource.
// 	MimeData *mimetype.MIME

// 	// DataParsed bool
// 	DataParsed chan bool

// 	Project *Project

// 	valid bool
// }

// func (project *Project) RegisterResource(resourcePath, localFilepath string, response *grab.Response) *Resource {

// 	modTime := time.Time{}
// 	size := int64(0)

// 	if localFile, err := os.Open(localFilepath); err == nil {
// 		if stats, err := localFile.Stat(); err == nil {
// 			modTime = stats.ModTime()
// 			size = stats.Size()
// 		}
// 	}

// 	res := &Resource{
// 		ModTime:          modTime,
// 		Size:             size,
// 		ResourcePath:     resourcePath,
// 		LocalFilepath:    localFilepath,
// 		DownloadResponse: response,
// 		Project:          project,
// 		DataParsed:       make(chan bool, 1),
// 		valid:            true,
// 	}

// 	project.Resources[resourcePath] = res

// 	if response != nil {

// 		project.DownloadingResources[resourcePath] = res

// 		// The first few bytes of a file indicates the kind of file it is; according to mimetype's internals, it's 3072 (at max, probably).
// 		// We wait for a fe4w seconds to at least let that download before attempting to detect the mime type below
// 		for !response.IsComplete() && response.BytesComplete() < 3072 {
// 			time.Sleep(time.Millisecond * 100)
// 		}

// 	}

// 	res.MimeData, _ = mimetype.DetectFile(res.LocalFilepath)

// 	// We have to do this because sometimes the suggested filepath is simply not enough to go off of (images off of Twitter, for example, don't have extensions).
// 	// Without an extension, raylib can't identify the file to load it.
// 	if filepath.Ext(res.LocalFilepath) == "" {
// 		os.Rename(res.LocalFilepath, res.LocalFilepath+res.MimeData.Extension())
// 		res.LocalFilepath += res.MimeData.Extension()
// 	}

// 	return res
// }

// func (res *Resource) Filename() string {
// 	_, fname := filepath.Split(res.LocalFilepath)
// 	return fname
// }

// func (res *Resource) ParseData() error {

// 	// If we've already parsed the data once before, remove the indicator before parsing it again.
// 	if len(res.DataParsed) > 0 {
// 		<-res.DataParsed
// 	}

// 	var err error = nil

// 	// If the mime data is just a generic sequence of data, then try to parse it again
// 	if res.MimeData.Is("application/octet-stream") {
// 		res.MimeData, _ = mimetype.DetectFile(res.LocalFilepath)
// 	}

// 	if !FileExists(res.LocalFilepath) {
// 		err = errors.New("file doesn't exist")
// 	} else {

// 		if strings.Contains(res.MimeData.String(), "image") {

// 			if strings.Contains(res.MimeData.String(), "gif") {

// 				file, newError := os.Open(res.LocalFilepath)
// 				if newError != nil {
// 					err = newError
// 				}

// 				defer file.Close()

// 				gifFile, newError := gif.DecodeAll(file)

// 				if newError != nil {
// 					err = newError
// 				}

// 				gif := NewGifAnimation(gifFile)
// 				res.Data = gif

// 			} else { // Ordinary image
// 				res.Data = rl.LoadTexture(res.LocalFilepath)
// 			}

// 		} else if strings.Contains(res.MimeData.String(), "audio") {
// 			res.Data = res.MimeData.String() // We don't actually have any data to store for audio, as Sound Tasks simply create their own streams
// 		} else {
// 			err = errors.New("unrecognized resource type")
// 			delete(res.Project.Resources, res.ResourcePath) // Delete the resource if it isn't recognized, as it shouldn't be used anyway (hopefully this is OK?)
// 		}

// 	}

// 	if err != nil {
// 		res.Project.Log("ERROR : "+err.Error()+" : %s", res.ResourcePath)
// 	} else {
// 		res.DataParsed <- true
// 	}

// 	return err

// }

// func (res *Resource) MimeIsImage() bool {
// 	return res.MimeData != nil && strings.Contains(res.MimeData.String(), "image")
// }

// func (res *Resource) MimeIsAudio() bool {
// 	return res.MimeData != nil && strings.Contains(res.MimeData.String(), "audio")
// }

// func (res *Resource) State() int {

// 	if !res.valid {
// 		return RESOURCE_STATE_DELETED
// 	}

// 	if res.DownloadResponse != nil && !res.DownloadResponse.IsComplete() {
// 		return RESOURCE_STATE_DOWNLOADING
// 	}

// 	if res.IsGif() {
// 		if res.Gif().LoadingProgress() < 1 {
// 			return RESOURCE_STATE_LOADING
// 		}

// 	}

// 	if len(res.DataParsed) > 0 && res.Data != nil {
// 		return RESOURCE_STATE_READY
// 	}
// 	return RESOURCE_STATE_LOADING

// }

// func (res *Resource) IsTexture() bool {
// 	_, isTexture := res.Data.(rl.Texture2D)
// 	return isTexture
// }

// func (res *Resource) Texture() rl.Texture2D {
// 	return res.Data.(rl.Texture2D)
// }

// func (res *Resource) IsGif() bool {
// 	_, isGIF := res.Data.(*GifAnimation)
// 	return isGIF
// }

// func (res *Resource) Gif() *GifAnimation {
// 	return res.Data.(*GifAnimation)
// }

// func (res *Resource) IsAudio() bool {
// 	return strings.Contains(res.MimeData.String(), "audio")
// }

// // Progress returns the progress of downloading or loading the resource, as an integer ranging from 0 to 100. If the returned value is less than 0, the progress cannot be determined.
// func (res *Resource) Progress() int {
// 	if res.DownloadResponse != nil && !res.DownloadResponse.IsComplete() {
// 		if res.DownloadResponse.Size < 0 {
// 			return -1 // We have to return some kind of number
// 		}
// 		return int(res.DownloadResponse.Progress() * 100)
// 	} else if res.IsGif() {
// 		return int(res.Gif().LoadingProgress() * 100)
// 	}
// 	return 0
// }

// // Audio is special in that there is no resource to be shared between Tasks like with Images, as each Task
// // should have its own stream it manages to play back audio. So instead, the resource's Audio() function returns
// // a brand new stream pointing to the audio file. The Task (or whatever uses the Stream) has to handle closing
// // the Stream when it's deleted (which it does in the ReceiveMessage() function when it is informed that it is
// // going to be deleted).
// func (res *Resource) Audio() (beep.StreamSeekCloser, beep.Format, error) {

// 	var stream beep.StreamSeekCloser
// 	var format beep.Format
// 	var err error

// 	if res.IsAudio() {

// 		file, err := os.Open(res.LocalFilepath)
// 		if err != nil {
// 			currentProject.Log("Could not open audio file: %s", err.Error())
// 		} else {

// 			switch ext := res.MimeData.Extension(); ext {
// 			case ".wav":
// 				stream, format, err = wav.Decode(file)
// 			case ".flac":
// 				stream, format, err = flac.Decode(file)
// 			case ".ogg":
// 			case ".oga":
// 				stream, format, err = vorbis.Decode(file)
// 			case ".mp3":
// 				stream, format, err = mp3.Decode(file)
// 			}

// 			if err != nil {
// 				currentProject.Log("Error decoding audio file: %s", err.Error())
// 			}

// 		}

// 	}

// 	return stream, format, err

// }

// func (res *Resource) Destroy() {

// 	if res.IsTexture() {
// 		rl.UnloadTexture(res.Texture())
// 	} else if res.IsGif() {
// 		res.Gif().Destroy()
// 	}
// 	// GIFs don't need to be disposed of directly here; the file handle was already Closed.
// 	// Audio streams are closed by the Task, as each Sound Task has its own stream.

// 	// We no longer delete temporary files here, as the project deletes the entire temporary directory in Project.Destroy().
// 	// if res.DownloadResponse != nil {
// 	// 	os.Remove(res.LocalFilepath)
// 	// }

// 	delete(res.Project.Resources, res.ResourcePath)

// 	res.valid = false

// }
