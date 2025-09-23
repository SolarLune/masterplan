package main

import (
	"bytes"
	"context"
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

	"github.com/Zyko0/go-sdl3/sdl"
)

type BrowserTab struct {
	Contents *InternetContents
	// CancelFunc      context.CancelFunc
	Context         context.Context
	Actions         chan chromedp.Action
	Initialized     atomic.Bool
	LoadingWebpage  atomic.Bool
	ForceRefresh    atomic.Bool
	CreationTime    time.Time
	UpdateFrametime time.Time
	Target          *chromedp.Target

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

	CreateNewTab atomic.Bool
	NewTabURL    string
}

func NewBrowserTab(w, h int, contents *InternetContents) (*BrowserTab, error) {

	if globals.BrowserContext == nil {

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

		globals.BrowserContext = browserContext

		go func() {

			for {

				frameStart := time.Now()

				tabs := make([]*BrowserTab, len(globals.BrowserTabs))

				globals.BrowserLock.Lock()
				copy(tabs, globals.BrowserTabs)
				globals.BrowserLock.Unlock()

				for _, browserTab := range tabs {

					if quit {
						break
					}

					deadlinedTabContext, _ := context.WithTimeout(browserTab.Context, time.Second)

					if !browserTab.Initialized.Load() {
						continue
					}

					// log.Println("loop start")

					// Force refreshing for the first few seconds to hopefully ensure cards display when loading a project
					if time.Since(browserTab.CreationTime) < time.Second*5 || browserTab.ForceRefresh.Load() {
						browserTab.ForceRefresh.Store(false) // If we're forcing refresh, we refresh this frame, but unset it for the future
					} else {

						updateOnlyWhen := browserTab.Contents.Card.Properties.Get("update only when").AsString()

						switch updateOnlyWhen {
						case InternetCardUpdateOptionWhenRecordingInputs:
							if !browserTab.Contents.RecordInput {
								continue
							}
						case InternetCardUpdateOptionWhenSelected:
							if !browserTab.Contents.Card.selected {
								continue
							}
							// case WebCardUpdateOptionAlways:
						}

						updateFPS := browserTab.Contents.Card.Properties.Get("update framerate").AsString()
						if updateFPS != InternetCardFPSAsOftenAsPossible {

							switch updateFPS {
							case InternetCardFPS1FPS:
								if time.Since(browserTab.UpdateFrametime) < time.Second {
									continue
								}
							case InternetCardFPS10FPS:
								if time.Since(browserTab.UpdateFrametime) < time.Second/10 {
									continue
								}
							case InternetCardFPS20FPS:
								if time.Since(browserTab.UpdateFrametime) < time.Second/20 {
									continue
								}
							}

						}

					}

					if browserTab.Target != nil {
						chromedp.Run(deadlinedTabContext, target.ActivateTarget(browserTab.Target.TargetID))
					}

					// log.Println("action start")

					// c := chromedp.FromContext(tab.Context)
					// chromedp.Run(deadlinedTabContext, target.ActivateTarget(c.Target.TargetID))

					// for len(tab.Actions) > 0 { // This is laggier when it comes to selecting text for some reason...?

					var actionError error

					for len(browserTab.Actions) > 0 {

						action := <-browserTab.Actions

						err := chromedp.Run(deadlinedTabContext, action)

						if err == context.Canceled {
							actionError = err
							break
						} else if err != nil && err != context.DeadlineExceeded {
							actionError = err
							globals.EventLog.Log("error: %s", false, err.Error())
						}

					}

					if actionError != nil {
						continue
					}

					browserTab.UpdateFrametime = time.Now()

					// log.Println("take picture")

					err := chromedp.Run(deadlinedTabContext, chromedp.CaptureScreenshot(&browserTab.ImageBuffer))

					if err == context.Canceled {
						break
					} else if err != nil && err != context.DeadlineExceeded {
						globals.EventLog.Log("error: %s", false, err.Error())
					} else if err == nil {

						decoded, err := png.Decode(bytes.NewReader(browserTab.ImageBuffer))

						if err != nil {
							globals.EventLog.Log(err.Error(), true)
							break
						}

						browserTab.Pause.Lock()

						imgWidth := decoded.Bounds().Dx()
						imgHeight := decoded.Bounds().Dy()

						waitGroup := &sync.WaitGroup{}

						decode := func(wg *sync.WaitGroup, yStart, yEnd int) {

							defer wg.Done()

							i := yStart * imgWidth * 4

							for y := yStart; y < yEnd; y++ {
								for x := 0; x < decoded.Bounds().Dx(); x++ {
									r, g, b, a := decoded.At(x, y).RGBA()
									if i >= len(browserTab.RawImage) {
										return
									}
									browserTab.RawImage[i] = byte(a)
									browserTab.RawImage[i+1] = byte(b)
									browserTab.RawImage[i+2] = byte(g)
									browserTab.RawImage[i+3] = byte(r)
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

						browserTab.Pause.Unlock()

						browserTab.ImageChanged.Store(true)

						if time.Now().After(browserTab.urlTime) {
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
								browserTab.CurrentURL = currentLocation
								browserTab.urlTime = time.Now().Add(time.Second / 4)
							}
						}

					}

				}

				if quit {
					return
				}

				// Sleep for one frame max, but subtract the time since the frame start
				time.Sleep((time.Second / time.Duration(targetFPS)) - time.Since(frameStart))

			}

		}()

		chromedp.WaitNewTarget(globals.BrowserContext, func(i *target.Info) bool {

			ctx, _ := chromedp.NewContext(globals.BrowserContext, chromedp.WithTargetID(i.TargetID))
			if err := chromedp.Run(ctx, page.Close()); err != nil {
				log.Println(err)
			}

			if i.URL != "" {

				for _, card := range globals.Project.CurrentPage.Cards {
					if card.selected && card.ContentType == ContentTypeInternet {
						browserTab := card.Contents.(*InternetContents).BrowserTab
						browserTab.CreateNewTab.Store(true)
						browserTab.NewTabURL = i.URL
						break
					}
				}

			}

			// id := i.TargetID

			// ctx, _ := chromedp.NewContext(globals.BrowserContext, chromedp.WithTargetID(id))

			// intendedLocation := ""

			// if err := chromedp.Run(ctx, chromedp.Location(&intendedLocation), page.Close()); err != nil {
			// 	log.Println(err)
			// }

			// ctxID := fmt.Sprintf("%p", browserTab.Context)

			// fmt.Println("new target", ctxID, intendedLocation)

			// if intendedLocation != "" {
			// 	browserTab.NewTabURL = intendedLocation
			// 	browserTab.CreateNewTab.Store(true)
			// }

			return false
		})

	}

	browserTab := &BrowserTab{
		Pause:        sync.Mutex{},
		Actions:      make(chan chromedp.Action, 9999),
		Contents:     contents,
		CreationTime: time.Now(),
	}

	newTab, _ := chromedp.NewContext(globals.BrowserContext)
	// newTab, cancel := chromedp.NewContext(globals.BrowserContext)

	// Attempt to run something; this should create a new tab
	if err := chromedp.Run(newTab, chromedp.Reload()); err != nil {
		globals.EventLog.Log("Error creating web card: %s", true, err.Error())
		return nil, err
	}

	browserTab.Target = chromedp.FromContext(newTab).Target

	chromedp.ListenTarget(newTab, func(ev interface{}) {
		switch e := ev.(type) {

		case *page.EventLifecycleEvent:
			if e.Name == "DOMContentLoaded" {
				browserTab.UpdateZoom()
				browserTab.LoadingWebpage.Store(true)
			}
			if e.Name == "networkIdle" {
				browserTab.LoadingWebpage.Store(false)
			}
		}
	})

	// browserTab.CancelFunc = cancel

	browserTab.Context = newTab

	browserTab.UpdateBufferSize(w, h)

	browserTab.Initialized.Store(true)

	globals.BrowserLock.Lock()
	globals.BrowserTabs = append(globals.BrowserTabs, browserTab)
	globals.BrowserLock.Unlock()

	// Browser Rendering Loop

	// go func(tab *BrowserTab) {

	// 	for {

	// 		// Maybe don't do this?
	// 		time.Sleep(time.Millisecond)

	// 		deadlinedTabContext, _ := context.WithTimeout(browserTab.Context, time.Millisecond*100)

	// 		if !tab.Initialized.Load() {
	// 			continue
	// 		}

	// 		// log.Println("loop start")

	// 		// Force refreshing for the first few seconds to hopefully ensure cards display when loading a project
	// 		if time.Since(tab.CreationTime) < time.Second*5 || tab.ForceRefresh.Load() {
	// 			tab.ForceRefresh.Store(false) // If we're forcing refresh, we refresh this frame, but unset it for the future
	// 		} else {

	// 			updateOnlyWhen := tab.Contents.Card.Properties.Get("update only when").AsString()

	// 			switch updateOnlyWhen {
	// 			case WebCardUpdateOptionWhenRecordingInputs:
	// 				if !tab.Contents.RecordInput {
	// 					continue
	// 				}
	// 			case WebCardUpdateOptionWhenSelected:
	// 				if !tab.Contents.Card.selected {
	// 					continue
	// 				}
	// 				// case WebCardUpdateOptionAlways:
	// 			}

	// 			updateFPS := tab.Contents.Card.Properties.Get("update framerate").AsString()
	// 			if updateFPS != WebCardFPSAsOftenAsPossible {

	// 				switch updateFPS {
	// 				case WebCardFPS1FPS:
	// 					if time.Since(tab.UpdateFrametime) < time.Second {
	// 						continue
	// 					}
	// 				case WebCardFPS10FPS:
	// 					if time.Since(tab.UpdateFrametime) < time.Second/10 {
	// 						continue
	// 					}
	// 				case WebCardFPS20FPS:
	// 					if time.Since(tab.UpdateFrametime) < time.Second/20 {
	// 						continue
	// 					}
	// 				}

	// 			}

	// 		}

	// 		// log.Println("action start")

	// 		// c := chromedp.FromContext(tab.Context)
	// 		// chromedp.Run(deadlinedTabContext, target.ActivateTarget(c.Target.TargetID))

	// 		// for len(tab.Actions) > 0 { // This is laggier when it comes to selecting text for some reason...?

	// 		if tab.Target != nil {
	// 			target.ActivateTarget(tab.Target.TargetID)
	// 		}

	// 		var actionError error

	// 		for len(tab.Actions) > 0 {

	// 			action := <-tab.Actions

	// 			err := chromedp.Run(deadlinedTabContext, action)

	// 			if err == context.Canceled {
	// 				actionError = err
	// 				break
	// 			} else if err != nil && err != context.DeadlineExceeded {
	// 				actionError = err
	// 				globals.EventLog.Log("error: %s", false, err.Error())
	// 			}

	// 		}

	// 		if actionError != nil {
	// 			break
	// 		}

	// 		tab.UpdateFrametime = time.Now()

	// 		// log.Println("take picture")

	// 		err := chromedp.Run(deadlinedTabContext, chromedp.CaptureScreenshot(&tab.ImageBuffer))

	// 		if err == context.Canceled {
	// 			break
	// 		} else if err != nil && err != context.DeadlineExceeded {
	// 			globals.EventLog.Log("error: %s", false, err.Error())
	// 		} else if err == nil {

	// 			decoded, err := png.Decode(bytes.NewReader(tab.ImageBuffer))

	// 			if err != nil {
	// 				globals.EventLog.Log(err.Error(), true)
	// 				break
	// 			}

	// 			tab.Pause.Lock()

	// 			imgWidth := decoded.Bounds().Dx()
	// 			imgHeight := decoded.Bounds().Dy()

	// 			waitGroup := &sync.WaitGroup{}

	// 			decode := func(wg *sync.WaitGroup, yStart, yEnd int) {

	// 				defer wg.Done()

	// 				i := yStart * imgWidth * 4

	// 				for y := yStart; y < yEnd; y++ {
	// 					for x := 0; x < decoded.Bounds().Dx(); x++ {
	// 						r, g, b, a := decoded.At(x, y).RGBA()
	// 						if i >= len(tab.RawImage) {
	// 							return
	// 						}
	// 						tab.RawImage[i] = byte(a)
	// 						tab.RawImage[i+1] = byte(b)
	// 						tab.RawImage[i+2] = byte(g)
	// 						tab.RawImage[i+3] = byte(r)
	// 						i += 4
	// 					}
	// 				}

	// 			}

	// 			chunkSize := runtime.NumCPU()
	// 			if chunkSize < 1 {
	// 				chunkSize = 1
	// 			}

	// 			imgChunkHeight := float32(imgHeight) / float32(chunkSize)

	// 			for i := float32(0); i < float32(chunkSize); i++ {
	// 				waitGroup.Add(1)
	// 				go decode(waitGroup, int(i*imgChunkHeight), int(imgChunkHeight*(i+1)))
	// 			}

	// 			waitGroup.Wait()

	// 			tab.Pause.Unlock()

	// 			tab.ImageChanged.Store(true)

	// 			if time.Now().After(tab.urlTime) {
	// 				currentLocation := ""

	// 				// log.Println("get url time")

	// 				// If this errors out, it's nbd; just try again later
	// 				err := chromedp.Run(deadlinedTabContext, chromedp.Location(&currentLocation))

	// 				if err == context.Canceled {
	// 					break
	// 				} else if err != nil && err != context.DeadlineExceeded {
	// 					globals.EventLog.Log("error: %s", false, err.Error())
	// 				}

	// 				if currentLocation != "" {
	// 					tab.CurrentURL = currentLocation
	// 					tab.urlTime = time.Now().Add(time.Second / 4)
	// 				}
	// 			}

	// 		}

	// 	}

	// 	log.Println("exiting web goroutine")

	// }(browserTab)

	return browserTab, nil

}

func (b *BrowserTab) Destroy() {

	globals.BrowserLock.Lock()
	for i, t := range globals.BrowserTabs {
		if t == b {
			globals.BrowserTabs[i] = nil
			globals.BrowserTabs = append(globals.BrowserTabs[:i], globals.BrowserTabs[i+1:]...)
		}
	}
	globals.BrowserLock.Unlock()

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

	if tex, err := globals.Renderer.CreateTexture(sdl.PIXELFORMAT_RGBA8888, sdl.TEXTUREACCESS_STREAMING, int(width), int(height)); err != nil {
		globals.EventLog.Log(err.Error(), true)
	} else {
		b.ImageTexture = tex
	}

	b.ImageTexture.SetScaleMode(sdl.SCALEMODE_LINEAR)

	b.Do(chromedp.Emulate(b.DeviceInfo))

}

func (b *BrowserTab) UpdateTexture() {

	if b.CreateNewTab.CompareAndSwap(true, false) {
		b.Contents.RecordInput = false

		newCard := b.Contents.Card.Page.CreateNewCard(ContentTypeNull)
		newCard.Recreate(b.Contents.Card.Rect.W, b.Contents.Card.Rect.H)
		newCard.Properties.CopyFrom(b.Contents.Card.Properties)
		newCard.Properties.Get("url").Set(b.NewTabURL)
		newCard.SetContents(ContentTypeInternet)
		placeCardInStack(newCard, false)
	}

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
