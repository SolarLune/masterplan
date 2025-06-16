package main

import (
	"bytes"
	"context"
	"fmt"
	"image/png"
	"log"
	"runtime"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/cdproto/target"
	"github.com/chromedp/chromedp"
	"github.com/chromedp/chromedp/device"
	"github.com/goware/urlx"
	"github.com/veandco/go-sdl2/sdl"
)

func NewBrowser(w, h int, contents *WebContents) (*BrowserTab, error) {

	opts := append(
		[]func(*chromedp.ExecAllocator){},
		chromedp.Flag("hide-scrollbars", true), // Not sure if we want this or not
		chromedp.Flag("headless", true),
		chromedp.Flag("no-first-run", true),
		chromedp.Flag("no-default-browser-check", true),
		chromedp.Flag("mute-audio", false),
		// chromedp.Flag("disable-background-networking", true),
		// chromedp.Flag("enable-features", "NetworkService,NetworkServiceInProcess"),
		chromedp.Flag("disable-background-timer-throttling", true),
		chromedp.Flag("disable-backgrounding-occluded-windows", true),
		chromedp.Flag("disable-renderer-backgrounding", true),
		// chromedp.Flag("disable-breakpad", true),
		// chromedp.Flag("disable-client-side-phishing-detection", true),
		// chromedp.Flag("disable-default-apps", true),
		// chromedp.Flag("disable-dev-shm-usage", true),
		chromedp.Flag("disable-extensions", false),
		// // chromedp.Flag("disable-features", "site-per-process,Translate,BlinkGenPropertyTrees"),
		// chromedp.Flag("disable-hang-monitor", true),
		// chromedp.Flag("disable-ipc-flooding-protection", true),
		// chromedp.Flag("disable-popup-blocking", true),
		// chromedp.Flag("block-new-web-contents", true),
		// chromedp.Flag("disable-prompt-on-repost", true),
		// chromedp.Flag("disable-renderer-backgrounding", true),
		// chromedp.Flag("disable-sync", true),
		// chromedp.Flag("force-color-profile", "srgb"),
		// chromedp.Flag("metrics-recording-only", true),
		// chromedp.Flag("safebrowsing-disable-auto-update", true),
		// chromedp.Flag("enable-automation", true),
		// chromedp.Flag("password-store", "basic"),
		// chromedp.Flag("use-mock-keychain", true),
	)

	if browserPath := globals.Settings.Get(SettingsBrowserPath).AsString(); browserPath != "" {
		opts = append(opts, chromedp.ExecPath(browserPath))
	}

	if userDataPath := globals.Settings.Get(SettingsBrowserUserDataPath).AsString(); userDataPath != "" {
		opts = append(opts, chromedp.UserDataDir(userDataPath))
	}

	alloc, _ := chromedp.NewExecAllocator(context.Background(), opts...)

	// create context
	browserContext, cancel := chromedp.NewContext(alloc)

	// Try context to confirm it exists and is good
	if err := chromedp.Run(browserContext, chromedp.Reload()); err != nil {
		globals.EventLog.Log("Error creating web card: %s", true, err.Error())
		cancel() // Cancel the broken browser context
		return nil, err
	}

	globals.EventLog.Log("Created web context.", false)

	browser := newBrowserTab(contents)

	// Attempt to run something; this should create a new tab
	if err := chromedp.Run(browserContext, chromedp.Reload()); err != nil {
		globals.EventLog.Log("Error creating web card: %s", true, err.Error())
		return nil, err
	}

	chromedp.ListenTarget(browserContext, func(ev interface{}) {
		switch e := ev.(type) {

		case *page.EventLifecycleEvent:
			if e.Name == "DOMContentLoaded" {
				browser.UpdateZoom()
				browser.LoadingWebpage.Store(true)
			}
			if e.Name == "networkIdle" {
				browser.LoadingWebpage.Store(false)
			}
		}
	})

	browser.CancelFunc = cancel

	browser.Context = browserContext

	browser.UpdateBufferSize(w, h)

	browser.Initialized.Store(true)

	browser.NewTabChannel = chromedp.WaitNewTarget(browser.Context, func(i *target.Info) bool {

		id := i.TargetID

		ctx, _ := chromedp.NewContext(browser.Context, chromedp.WithTargetID(id))

		intendedLocation := ""

		if err := chromedp.Run(ctx, chromedp.Location(&intendedLocation), page.Close()); err != nil {
			log.Println(err)
		}

		fmt.Println(intendedLocation)

		if intendedLocation != "" {
			browser.Navigate(intendedLocation)
			// card := browser.Contents.Card.Page.CreateNewCard(ContentTypeCheckbox)
			// card.Contents.(*WebContents).BrowserTab.Navigate(intendedLocation)
		}

		return true
	})

	// Browser Rendering Loop

	go func(tab *BrowserTab) {

		for {

			// Maybe don't do this?
			time.Sleep(time.Millisecond)

			deadlinedTabContext, _ := context.WithTimeout(browser.Context, time.Second)

			if !tab.Initialized.Load() {
				continue
			}

			// log.Println("loop start")

			// Force refreshing for the first few seconds to hopefully ensure cards display when loading a project
			if time.Since(tab.CreationTime) < time.Second*5 || tab.ForceRefresh.Load() {
				tab.ForceRefresh.Store(false) // If we're forcing refresh, we refresh this frame, but unset it for the future
			} else {

				updateOnlyWhen := tab.Contents.Card.Properties.Get("update only when").AsString()

				switch updateOnlyWhen {
				case WebCardUpdateOptionWhenRecordingInputs:
					if !tab.Contents.RecordInput {
						continue
					}
				case WebCardUpdateOptionWhenSelected:
					if !tab.Contents.Card.selected {
						continue
					}
					// case WebCardUpdateOptionAlways:
				}

				updateFPS := tab.Contents.Card.Properties.Get("update framerate").AsString()
				if updateFPS != WebCardFPSAsOftenAsPossible {

					switch updateFPS {
					case WebCardFPS1FPS:
						if time.Since(tab.UpdateFrametime) < time.Second {
							continue
						}
					case WebCardFPS10FPS:
						if time.Since(tab.UpdateFrametime) < time.Second/10 {
							continue
						}
					case WebCardFPS20FPS:
						if time.Since(tab.UpdateFrametime) < time.Second/20 {
							continue
						}
					}

				}

			}

			// log.Println("action start")

			// c := chromedp.FromContext(tab.Context)
			// chromedp.Run(deadlinedTabContext, target.ActivateTarget(c.Target.TargetID))

			// for len(tab.Actions) > 0 { // This is laggier when it comes to selecting text for some reason...?

			for len(tab.Actions) > 0 {

				action := <-tab.Actions

				err := chromedp.Run(browser.Context, action)

				if err == context.Canceled {
					break
				} else if err != nil && err != context.DeadlineExceeded {
					globals.EventLog.Log("error: %s", false, err.Error())
				}

			}

			tab.UpdateFrametime = time.Now()

			// log.Println("take picture")

			err := chromedp.Run(deadlinedTabContext, chromedp.CaptureScreenshot(&tab.ImageBuffer))

			if err == context.Canceled {
				break
			} else if err != nil && err != context.DeadlineExceeded {
				globals.EventLog.Log("error: %s", false, err.Error())
			} else if err == nil {

				// fmt.Println("capture screenshot success")

				decoded, err := png.Decode(bytes.NewReader(tab.ImageBuffer))

				if err != nil {
					globals.EventLog.Log(err.Error(), true)
					break
				}

				tab.Pause.Lock()

				imgWidth := decoded.Bounds().Dx()
				imgHeight := decoded.Bounds().Dy()

				waitGroup := &sync.WaitGroup{}

				decode := func(wg *sync.WaitGroup, yStart, yEnd int) {

					defer wg.Done()

					i := yStart * imgWidth * 4

					for y := yStart; y < yEnd; y++ {
						for x := 0; x < decoded.Bounds().Dx(); x++ {
							r, g, b, a := decoded.At(x, y).RGBA()
							if i >= len(tab.RawImage) {
								return
							}
							tab.RawImage[i] = byte(a)
							tab.RawImage[i+1] = byte(b)
							tab.RawImage[i+2] = byte(g)
							tab.RawImage[i+3] = byte(r)
							i += 4
						}
					}

				}

				chunkSize := runtime.NumCPU()
				if chunkSize < 1 {
					chunkSize = 1
				}

				imgChunkHeight := float32(imgHeight) / float32(chunkSize)

				for i := float32(0); i < float32(chunkSize); i++ {
					waitGroup.Add(1)
					go decode(waitGroup, int(i*imgChunkHeight), int(imgChunkHeight*(i+1)))
				}

				waitGroup.Wait()

				tab.Pause.Unlock()

				tab.ImageChanged.Store(true)

				if time.Now().After(tab.urlTime) {
					currentLocation := ""

					// log.Println("get url time")

					// If this errors out, it's nbd; just try again later
					err := chromedp.Run(deadlinedTabContext, chromedp.Location(&currentLocation))

					if err == context.Canceled {
						break
					} else if err != nil && err != context.DeadlineExceeded {
						globals.EventLog.Log("error: %s", false, err.Error())
					}

					if currentLocation != "" {
						tab.CurrentURL = currentLocation
						tab.urlTime = time.Now().Add(time.Second / 4)
					}
				}

			}

		}

		log.Println("exiting web goroutine")

	}(browser)

	return browser, nil

}

// type ChromeBrowser struct {
// 	Context context.Context
// 	Tabs    []*BrowserTab
// }

// func (b *ChromeBrowser) Init() error {

// 	if b.Context == nil {

// 		// opts := append(chromedp.DefaultExecAllocatorOptions[:], chromedp.Flag("headless", true))
// 		opts := append([]func(*chromedp.ExecAllocator){},
// 			chromedp.Flag("hide-scrollbars", true), // Not sure if we want this or not
// 			chromedp.Flag("headless", true),
// 			chromedp.Flag("no-first-run", true),
// 			chromedp.Flag("no-default-browser-check", true),
// 			chromedp.Flag("mute-audio", false),
// 			// chromedp.Flag("disable-background-networking", true),
// 			// chromedp.Flag("enable-features", "NetworkService,NetworkServiceInProcess"),
// 			chromedp.Flag("disable-background-timer-throttling", true),
// 			chromedp.Flag("disable-backgrounding-occluded-windows", true),
// 			chromedp.Flag("disable-renderer-backgrounding", true),
// 			// chromedp.Flag("disable-breakpad", true),
// 			// chromedp.Flag("disable-client-side-phishing-detection", true),
// 			// chromedp.Flag("disable-default-apps", true),
// 			// chromedp.Flag("disable-dev-shm-usage", true),
// 			chromedp.Flag("disable-extensions", false),
// 			// // chromedp.Flag("disable-features", "site-per-process,Translate,BlinkGenPropertyTrees"),
// 			// chromedp.Flag("disable-hang-monitor", true),
// 			// chromedp.Flag("disable-ipc-flooding-protection", true),
// 			chromedp.Flag("disable-popup-blocking", true),
// 			// chromedp.Flag("disable-prompt-on-repost", true),
// 			// chromedp.Flag("disable-renderer-backgrounding", true),
// 			// chromedp.Flag("disable-sync", true),
// 			// chromedp.Flag("force-color-profile", "srgb"),
// 			// chromedp.Flag("metrics-recording-only", true),
// 			// chromedp.Flag("safebrowsing-disable-auto-update", true),
// 			// chromedp.Flag("enable-automation", true),
// 			// chromedp.Flag("password-store", "basic"),
// 			// chromedp.Flag("use-mock-keychain", true),
// 		)

// 		if browserPath := globals.Settings.Get(SettingsBrowserPath).AsString(); browserPath != "" {
// 			opts = append(opts, chromedp.ExecPath(browserPath))
// 		}

// 		if userDataPath := globals.Settings.Get(SettingsBrowserUserDataPath).AsString(); userDataPath != "" {
// 			opts = append(opts, chromedp.UserDataDir(userDataPath))
// 		}

// 		alloc, _ := chromedp.NewExecAllocator(context.Background(), opts...)
// 		browserContext, cancel := chromedp.NewContext(alloc)

// 		// Try context to confirm it exists and is good
// 		if err := chromedp.Run(browserContext, chromedp.Reload()); err != nil {
// 			globals.EventLog.Log("Error creating web card: %s", true, err.Error())
// 			cancel() // Cancel the broken browser context
// 			return err
// 		}

// 		b.Context = browserContext

// 		globals.EventLog.Log("Created web context.", false)

// 	}

// 	return nil

// }

// func (b *ChromeBrowser) CreateTab(w, h int, contents *WebContents) (*BrowserTab, error) {
// 	tab := newBrowserTab(contents)

// 	// create context
// 	ctx, tabCancel := chromedp.NewContext(
// 		globals.ChromeBrowser.Context,
// 	)

// 	// Attempt to run something; this should create a new tab
// 	if err := chromedp.Run(ctx, chromedp.Reload()); err != nil {
// 		globals.EventLog.Log("Error creating web card: %s", true, err.Error())
// 		return nil, err
// 	}

// 	chromedp.ListenTarget(ctx, func(ev interface{}) {
// 		switch e := ev.(type) {
// 		case *page.EventLifecycleEvent:
// 			if e.Name == "DOMContentLoaded" {
// 				tab.LoadingWebpage.Store(true)
// 			}
// 			if e.Name == "networkIdle" {
// 				tab.LoadingWebpage.Store(false)
// 			}
// 		}
// 	})

// 	tab.CancelFunc = tabCancel

// 	tab.Context = ctx

// 	// go func() {
// 	// 	// New tab created
// 	// 	for {
// 	// 		exists := chromedp.WaitNewTarget(browserContext, func(i *target.Info) bool {
// 	// 			return true
// 	// 		})

// 	// 		if len(exists) > 0 {
// 	// 			target := <-exists
// 	// 			fmt.Println("NEW TARGET CREATED:", target)
// 	// 		}
// 	// 		time.Sleep(time.Second / 2)
// 	// 	}
// 	// }()

// 	tab.UpdateBufferSize(w, h)

// 	tab.Initialized.Store(true)

// 	b.Tabs = append(b.Tabs, tab)

// 	// Browser Rendering Loop

// 	go func(tab *BrowserTab) {

// 		for {

// 			deadlinedTabContext, cancel := context.WithTimeout(tab.Context, time.Second)
// 			defer cancel()

// 			if !tab.Initialized.Load() {
// 				continue
// 			}

// 			if time.Since(tab.CreationTime) < time.Second*5 || tab.ForceRefresh.Load() {
// 				tab.ForceRefresh.Store(false) // If we're forcing refresh, we refresh this frame, but unset it for the future
// 			} else {

// 				updateOnlyWhen := tab.Contents.Card.Properties.Get("update only when").AsString()

// 				switch updateOnlyWhen {
// 				case WebCardUpdateOptionWhenRecordingInputs:
// 					if !tab.Contents.RecordInput {
// 						continue
// 					}
// 				case WebCardUpdateOptionWhenSelected:
// 					if !tab.Contents.Card.selected {
// 						continue
// 					}
// 					// case WebCardUpdateOptionAlways:
// 				}

// 				updateFPS := tab.Contents.Card.Properties.Get("update framerate").AsString()
// 				if updateFPS != WebCardFPSAsOftenAsPossible {

// 					switch updateFPS {
// 					case WebCardFPS1FPS:
// 						if time.Since(tab.UpdateFrametime) < time.Second {
// 							continue
// 						}
// 					case WebCardFPS10FPS:
// 						if time.Since(tab.UpdateFrametime) < time.Second/10 {
// 							continue
// 						}
// 					case WebCardFPS20FPS:
// 						if time.Since(tab.UpdateFrametime) < time.Second/20 {
// 							continue
// 						}
// 					}

// 				}

// 			}

// 			c := chromedp.FromContext(tab.Context)
// 			chromedp.Run(deadlinedTabContext, target.ActivateTarget(c.Target.TargetID))

// 			// for len(tab.Actions) > 0 { // This is laggier when it comes to selecting text for some reason...?
// 			if len(tab.Actions) > 0 {

// 				action := <-tab.Actions

// 				err := chromedp.Run(deadlinedTabContext, action)

// 				if err == context.Canceled {
// 					return
// 				} else if err != nil && err != context.DeadlineExceeded {
// 					globals.EventLog.Log("error: %s", false, err.Error())
// 				}

// 			}

// 			tab.UpdateFrametime = time.Now()

// 			err := chromedp.Run(deadlinedTabContext, chromedp.CaptureScreenshot(&tab.ImageBuffer))

// 			if err == context.Canceled {
// 				return
// 			} else if err != nil && err != context.DeadlineExceeded {
// 				globals.EventLog.Log("error: %s", false, err.Error())
// 			} else if err == nil {

// 				// fmt.Println("capture screenshot success")

// 				decoded, err := png.Decode(bytes.NewReader(tab.ImageBuffer))

// 				if err != nil {
// 					globals.EventLog.Log(err.Error(), true)
// 					return
// 				}

// 				i := 0

// 			out:

// 				for y := 0; y < decoded.Bounds().Dy(); y++ {
// 					for x := 0; x < decoded.Bounds().Dx(); x++ {
// 						r, g, b, a := decoded.At(x, y).RGBA()
// 						if i >= len(tab.RawImage) {
// 							break out
// 						}
// 						tab.RawImage[i] = byte(a)
// 						tab.RawImage[i+1] = byte(b)
// 						tab.RawImage[i+2] = byte(g)
// 						tab.RawImage[i+3] = byte(r)
// 						i += 4
// 					}
// 				}

// 				tab.ImageChanged.Store(true)

// 				if time.Now().After(tab.urlTime) {
// 					currentLocation := ""

// 					// If this errors out, it's nbd; just try again later
// 					err := chromedp.Run(deadlinedTabContext, chromedp.Location(&currentLocation))

// 					if err == context.Canceled {
// 						return
// 					} else if err != nil && err != context.DeadlineExceeded {
// 						globals.EventLog.Log("error: %s", false, err.Error())
// 					}

// 					if currentLocation != "" {
// 						tab.CurrentURL = currentLocation
// 						tab.urlTime = time.Now().Add(time.Second / 4)
// 					}
// 				}

// 			}

// 		}

// 	}(tab)

// 	return tab, nil
// }

type BrowserTab struct {
	Contents        *WebContents
	CancelFunc      context.CancelFunc
	Context         context.Context
	Actions         chan chromedp.Action
	Initialized     atomic.Bool
	LoadingWebpage  atomic.Bool
	ForceRefresh    atomic.Bool
	CreationTime    time.Time
	UpdateFrametime time.Time
	NewTabChannel   <-chan target.ID

	urlTime    time.Time
	PastURL    string
	CurrentURL string

	Pause sync.Mutex

	ImageChanged atomic.Bool
	RawImage     []byte
	ImageBuffer  []byte
	ImageTexture *sdl.Texture
	DeviceInfo   chromedp.Device

	BufferWidth  int
	BufferHeight int
	ToRemove     bool
}

func newBrowserTab(contents *WebContents) *BrowserTab {
	bt := &BrowserTab{
		Pause:        sync.Mutex{},
		Actions:      make(chan chromedp.Action, 9999),
		Contents:     contents,
		CreationTime: time.Now(),
	}
	return bt
}

func (b *BrowserTab) Destroy() {
	err := chromedp.Cancel(b.Context)
	if err != nil {
		log.Println(err)
	}
	b.ImageTexture.Destroy()
	b.ToRemove = true

	// for i, t := range globals.ChromeBrowser.Tabs {
	// 	if t == b {
	// 		globals.ChromeBrowser.Tabs[i] = nil
	// 		globals.ChromeBrowser.Tabs = append(globals.ChromeBrowser.Tabs[:i], globals.ChromeBrowser.Tabs[i+1:]...)
	// 		break
	// 	}
	// }

}

func (b *BrowserTab) Do(action chromedp.Action) {
	b.Actions <- action
}

func (b *BrowserTab) UpdateBufferSize(w, h int) {

	b.Pause.Lock()
	defer b.Pause.Unlock()

	hint := sdl.GetHint(sdl.HINT_RENDER_SCALE_QUALITY)
	sdl.SetHint(sdl.HINT_RENDER_SCALE_QUALITY, "2") // Smooth filtering when creating the texture for web card for increased readability

	deviceInfo := device.Reset.Device()
	deviceInfo.UserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36 Edg/91.0.864.59"
	// deviceInfo.UserAgent = "Mozilla/5.0 (Linux; Android 11; Pixel 5) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/99.0.4812.0 Mobile Safari/537.36"

	deviceInfo.Width = int64(w)
	deviceInfo.Height = int64(h)
	deviceInfo.Scale = 1
	deviceInfo.Touch = false
	deviceInfo.Mobile = false
	deviceInfo.Landscape = true
	width := float64(deviceInfo.Width)
	height := float64(deviceInfo.Height)

	b.BufferWidth = int(width)
	b.BufferHeight = int(height)
	b.RawImage = make([]byte, int(width*height*4))
	b.DeviceInfo = deviceInfo
	b.ImageBuffer = []byte{}

	if b.ImageTexture != nil {
		b.ImageTexture.Destroy()
	}

	if tex, err := globals.Renderer.CreateTexture(sdl.PIXELFORMAT_RGBA8888, sdl.TEXTUREACCESS_STREAMING, int32(width), int32(height)); err != nil {
		globals.EventLog.Log(err.Error(), true)
	} else {
		b.ImageTexture = tex
	}

	b.Do(chromedp.Emulate(b.DeviceInfo))

	sdl.SetHint(sdl.HINT_RENDER_SCALE_QUALITY, hint)

}

func (b *BrowserTab) UpdateTexture() {

	if b.ImageTexture != nil && b.ImageChanged.CompareAndSwap(true, false) {

		b.Pause.Lock()
		data, _, err := b.ImageTexture.Lock(nil)
		if err != nil {
			log.Println(err)
		} else {
			copy(data, b.RawImage)
		}
		b.ImageTexture.Unlock()
		// b.ImageTexture.Update(nil, unsafe.Pointer(&b.RawImage[0]), b.BufferWidth*4)
		b.Pause.Unlock()

	}
}

func (b *BrowserTab) NavigateBack() {

	// Some sites hang indefinitely with chromedp.NavigateBack() - see issue: https://github.com/chromedp/chromedp/issues/1346
	// entries := []*page.NavigationEntry{}
	// currentEntry := int64(0)
	// chromedp.Run(b.Context, chromedp.NavigationEntries(&currentEntry, &entries))
	// if int(currentEntry) >= 1 {
	// 	b.Do(chromedp.NavigateBack())
	// }

	b.Do(chromedp.EvaluateAsDevTools("history.back()", nil))

}

func (b *BrowserTab) NavigateForward() {
	// entries := []*page.NavigationEntry{}
	// currentEntry := int64(0)
	// chromedp.Run(b.Context, chromedp.NavigationEntries(&currentEntry, &entries))
	// if int(currentEntry) >= 1 {
	// 	b.Do(chromedp.NavigateBack())
	// }
	b.Do(chromedp.EvaluateAsDevTools("history.forward()", nil))

}

func (b *BrowserTab) Navigate(url string) {

	b.CurrentURL = url
	parsed, err := urlx.Parse(url)

	log.Println("URL Parsed:", url)

	if err == nil {
		b.Do(chromedp.Navigate(parsed.String()))
	} else {
		globals.EventLog.Log("Error navigating to website: [ %s ];\nAre you sure the website URL is correct?\nError: [ %s ]", true, url, err.Error())
		b.LoadingWebpage.Store(false)
	}
}

func (b *BrowserTab) Valid() bool {
	return b.Context != nil && b.ImageTexture != nil
}

func (b *BrowserTab) UpdateZoom() {
	zoomLevel := strconv.FormatFloat(globals.Settings.Get(SettingsWebCardZoomLevel).AsFloat(), 'f', 2, 64)
	b.Do(chromedp.Evaluate("document.body.style.zoom = "+zoomLevel+";", nil))
}
