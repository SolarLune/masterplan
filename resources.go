package main

import (
	"errors"
	"image/gif"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/cavaliercoder/grab"
	"github.com/faiface/beep"
	"github.com/faiface/beep/flac"
	"github.com/faiface/beep/mp3"
	"github.com/faiface/beep/vorbis"
	"github.com/faiface/beep/wav"
	"github.com/gabriel-vasile/mimetype"
	rl "github.com/gen2brain/raylib-go/raylib"
)

const (
	RESOURCE_STATE_DOWNLOADING = iota
	RESOURCE_STATE_LOADING
	RESOURCE_STATE_READY
)

type Resource struct {
	ModTime time.Time
	// Path facing the object requesting the resouce (e.g. "~/home/pictures/test.png" or "https://solarlune.com/media/bartender.png")
	ResourcePath string

	// Where the resource is located on disk (e.g. "~/home/pictures/test.png" or "/tmp/masterplan_resource076755900.png", for downloaded resources)
	LocalFilepath string

	// Pointer to the data the resource stands for (e.g. a rl.Texture2D for an image)
	Data interface{}

	// If the resource was downloaded via a URL, this points to the *grab.Response used to load the data.
	DownloadResponse *grab.Response

	// MIME data for the Resource.
	MimeData *mimetype.MIME

	Exists bool

	Project *Project
}

func (project *Project) RegisterResource(resourcePath, localFilepath string, response *grab.Response) *Resource {

	modTime := time.Time{}

	if localFile, err := os.Open(localFilepath); err == nil {
		if stats, err := localFile.Stat(); err == nil {
			modTime = stats.ModTime()
		}
	}

	res := &Resource{
		ResourcePath:     resourcePath,
		LocalFilepath:    localFilepath,
		ModTime:          modTime,
		DownloadResponse: response,
		Project:          project,
	}

	project.Resources[resourcePath] = res

	if response != nil {
		project.DownloadingResources[resourcePath] = res
	}

	return res
}

func (res *Resource) Filename() string {
	_, fname := filepath.Split(res.LocalFilepath)
	return fname
}

func (res *Resource) ParseData() error {

	var err error = nil

	if !FileExists(res.LocalFilepath) {
		err = errors.New("file doesn't exist")
		res.Exists = false
	} else {

		// ParseData() is automatically called when the resource is (or, at least, should be) fully downloaded, so the Mime data should be complete and usable
		mime, _ := mimetype.DetectFile(res.LocalFilepath)

		res.MimeData = mime

		// We have to do this because sometimes the suggested filepath is simply not enough to go off of (images off of Twitter, for example, don't have extensions).
		// Without an extension, raylib can't identify the file to load it.
		if filepath.Ext(res.LocalFilepath) == "" {
			os.Rename(res.LocalFilepath, res.LocalFilepath+mime.Extension())
			res.LocalFilepath += mime.Extension()
		}

		res.Exists = true

		if strings.Contains(res.MimeData.String(), "image") {

			if strings.Contains(res.MimeData.String(), "gif") {

				file, newError := os.Open(res.LocalFilepath)
				if newError != nil {
					err = newError
				}

				defer file.Close()

				gifFile, newError := gif.DecodeAll(file)

				if newError != nil {
					err = newError
				}

				gif := NewGifAnimation(gifFile)
				res.Data = gif

			} else { // Ordinary image
				res.Data = rl.LoadTexture(res.LocalFilepath)
			}

		} else if strings.Contains(res.MimeData.String(), "audio") {
			res.Data = res.MimeData.String() // We don't actually have any data to store for audio, as Sound Tasks simply create their own streams
		} else {
			err = errors.New("unrecognized resource type")
			res.Exists = false
		}

	}

	if err != nil {
		res.Project.Log("ERROR : "+err.Error()+" : %s", res.ResourcePath)
		delete(res.Project.DownloadingResources, res.ResourcePath)
	}

	return err

}

func (res *Resource) State() int {

	if res.DownloadResponse != nil && !res.DownloadResponse.IsComplete() {
		return RESOURCE_STATE_DOWNLOADING
	}

	if res.IsGif() {
		if res.Gif().LoadingProgress() < 1 {
			return RESOURCE_STATE_LOADING
		}

	}

	if res.Data != nil {
		return RESOURCE_STATE_READY
	}
	return RESOURCE_STATE_LOADING

}

func (res *Resource) IsTexture() bool {
	_, isTexture := res.Data.(rl.Texture2D)
	return isTexture
}

func (res *Resource) Texture() rl.Texture2D {
	return res.Data.(rl.Texture2D)
}

func (res *Resource) IsGif() bool {
	_, isGIF := res.Data.(*GifAnimation)
	return isGIF
}

func (res *Resource) Gif() *GifAnimation {
	return res.Data.(*GifAnimation)
}

func (res *Resource) IsAudio() bool {
	return strings.Contains(res.MimeData.String(), "audio")
}

// Progress returns the progress of downloading or loading the resource, as an integer ranging from 0 to 100. If the returned value is less than 0, the progress cannot be determined.
func (res *Resource) Progress() int {
	if res.DownloadResponse != nil && !res.DownloadResponse.IsComplete() {
		if res.DownloadResponse.Size < 0 {
			return -1 // We have to return some kind of number
		}
		return int(res.DownloadResponse.Progress() * 100)
	} else if res.IsGif() {
		return int(res.Gif().LoadingProgress() * 100)
	}
	return 0
}

// Audio is special in that there is no resource to be shared between Tasks like with Images, as each Task
// should have its own stream it manages to play back audio. So instead, the resource's Audio() function returns
// a brand new stream pointing to the audio file. The Task (or whatever uses the Stream) has to handle closing
// the Stream when it's deleted (which it does in the ReceiveMessage() function when it is informed that it is
// going to be deleted).
func (res *Resource) Audio() (beep.StreamSeekCloser, beep.Format, error) {

	var stream beep.StreamSeekCloser
	var format beep.Format
	var err error

	if res.IsAudio() {

		file, err := os.Open(res.LocalFilepath)
		if err != nil {
			currentProject.Log("Could not open audio file: %s", err.Error())
		} else {

			switch ext := res.MimeData.Extension(); ext {
			case ".wav":
				stream, format, err = wav.Decode(file)
			case ".flac":
				stream, format, err = flac.Decode(file)
			case ".ogg":
			case ".oga":
				stream, format, err = vorbis.Decode(file)
			case ".mp3":
				stream, format, err = mp3.Decode(file)
			}

			if err != nil {
				currentProject.Log("Error decoding audio file: %s", err.Error())
			}

		}

	}

	return stream, format, err

}

func (res *Resource) Destroy() {

	if res.IsTexture() {
		rl.UnloadTexture(res.Texture())
	} else if res.IsGif() {
		res.Gif().Destroy()
	}
	// GIFs don't need to be disposed of directly here; the file handle was already Closed.
	// Audio streams are closed by the Task, as each Sound Task has its own stream.

	// We no longer delete temporary files here, as the project deletes the entire temporary directory in Project.Destroy().
	// if res.DownloadResponse != nil {
	// 	os.Remove(res.LocalFilepath)
	// }

}
