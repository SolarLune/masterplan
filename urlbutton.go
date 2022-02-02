package main

import (
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/goware/urlx"
	"github.com/veandco/go-sdl2/sdl"
)

type ParsedResult struct {
	Title       string
	Description string
	FavIcon     *Resource
}

func NewParsedResult(title, desc string) *ParsedResult {
	return &ParsedResult{
		Title:       title,
		Description: desc,
	}
}

type URLButton struct {
	URLButtons *URLButtons
	Pos        Point
	Size       Point
	Text       string
	Link       string
	Result     *ParsedResult
}

func (urlButton *URLButton) MousedOver() bool {
	rect := &sdl.FRect{
		urlButton.Pos.X,
		urlButton.Pos.Y,
		urlButton.Size.X,
		urlButton.Size.Y,
	}
	rect = urlButton.URLButtons.Card.Page.Project.Camera.TranslateRect(rect)
	return globals.Mouse.Position().Inside(rect)
}

func (urlButton *URLButton) Parse() {

	result := NewParsedResult("---", "---")

	resp, err := globals.HTTPClient.Get(urlButton.Link)
	if err != nil {
		globals.EventLog.Log(err.Error())
	} else {

		doc, _ := goquery.NewDocumentFromReader(resp.Body)
		if t := doc.Find("title"); t.Length() > 0 {
			result.Title = t.Text()
		}

		doc.Find("meta").Each(func(i int, s *goquery.Selection) {
			if name, exists := s.Attr("name"); exists && name == "description" {
				t, _ := s.Attr("content")
				result.Description = t
				return
			}
		})

		// doc.Find("link").Each(func(i int, s *goquery.Selection) {
		// 	fmt.Println("link", s, s.Text())
		// 	if rel, exists := s.Attr("rel"); exists && strings.Contains(rel, "icon") {
		// 		fmt.Println("found favicon")
		// 		fmt.Println(s.Attr("href"))
		// 		return
		// 	}
		// })

	}

	parsedURL, err := urlx.Parse(urlButton.Link)

	if err == nil {

		result.FavIcon = globals.Resources.Get("http://icons.duckduckgo.com/ip3/" + parsedURL.Host + ".ico")

		// faviconData, err := globals.HTTPClient.Get("http://icons.duckduckgo.com/ip3/" + parsedURL.Host + ".ico")

		// if err == nil {

		// 	imgData, err := io.ReadAll(faviconData.Body)

		// 	if err == nil {
		// 		fmt.Println(imgData)
		// 	}

		// }

	}

	urlButton.Result = result

}

type URLButtons struct {
	Card        *Card
	Buttons     []URLButton
	ScannedText string
}

func NewURLButtons(card *Card) *URLButtons {

	buttons := &URLButtons{Card: card}
	return buttons

}

func (buttons *URLButtons) ScanText(text string) {

	if buttons.ScannedText == text {
		return
	}

	buttons.Buttons = []URLButton{}

	currentURLButton := URLButton{
		URLButtons: buttons,
	}
	wordStart := Point{}

	for i, letter := range []rune(text) {

		validRune := true

		if letter != ' ' && letter != '\n' {

			if validRune {
				currentURLButton.Text += string(letter)
			}
			// wordStart.X += rl.MeasureTextEx(font, string(letter), float32(programSettings.FontSize), spacing).X + 1
			wordStart.X += globals.TextRenderer.MeasureText([]rune{letter}, 1).X + 1

		}

		if letter == ' ' || letter == '\n' || i == len(text)-1 {

			if len(currentURLButton.Text) > 0 {

				size := globals.TextRenderer.MeasureText([]rune(currentURLButton.Text), 1)

				currentURLButton.Size.X = size.X
				currentURLButton.Size.Y = size.Y

				urlText := strings.Trim(strings.Trim(strings.TrimSpace(currentURLButton.Text), "."), ":")

				if strings.Contains(urlText, ".") || strings.Contains(urlText, ":") {

					if url, err := urlx.Parse(urlText); err == nil && url.Host != "" && url.Scheme != "" {

						currentURLButton.Link = url.String()
						currentURLButton.Parse()
						buttons.Buttons = append(buttons.Buttons, currentURLButton)

					}

				}

			}

			if letter == '\n' {
				height := globals.TextRenderer.MeasureText([]rune{'A'}, 1).Y
				wordStart.Y += height
				wordStart.X = 0
			} else if letter == ' ' {
				wordStart.X += globals.TextRenderer.MeasureText([]rune{letter}, 1).X + 1
			}

			currentURLButton = URLButton{
				URLButtons: buttons,
			}
			currentURLButton.Pos = wordStart

		}

	}

	buttons.ScannedText = text

}

// func (buttons *URLButtons) Draw(pos Point) {

// 	// project := buttons.Card.Page.Project

// 	for _, urlButton := range buttons.Buttons {

// 		// if project.IsInNeutralState() && (project.AlwaysShowURLButtons.Checked || globals.Keybindings.Pressed(KBURLButton)) {
// 		if globals.State == StateNeutral {

// 			if ImmediateButton(pos.X+urlButton.Pos.X, pos.Y+urlButton.Pos.Y, urlButton.Text) {

// 				fmt.Println("click?")

// 				// We delay opening the URL by a few milliseconds to try to ensure you have time to let go of the mouse button
// 				go func() {
// 					time.Sleep(time.Millisecond * 100)
// 					browser.OpenURL(urlButton.Link)
// 				}()

// 			}

// 		}

// 	}

// }
