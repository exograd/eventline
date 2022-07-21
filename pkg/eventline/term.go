package eventline

import (
	"bytes"
	"fmt"
	"html/template"
	"strings"

	"github.com/leaanthony/go-ansi-parser"
)

func RenderTermData(rawData string) (string, error) {
	options := []ansi.ParseOption{
		ansi.WithIgnoreInvalidCodes(),
	}

	fragments, err := ansi.Parse(rawData, options...)
	if err != nil {
		return "", err
	}

	var htmlData bytes.Buffer

	for _, f := range fragments {
		renderTermDataFragment(f, &htmlData)
	}

	return htmlData.String(), nil
}

func renderTermDataFragment(f *ansi.StyledText, buf *bytes.Buffer) {
	// See eventline.scss for CSS class names

	var classes []string
	var styles []string

	if f.Bold() {
		classes = append(classes, "tb")
	}

	if f.Faint() {
		classes = append(classes, "tf")
	}

	if f.Italic() {
		classes = append(classes, "ti")
	}

	if f.Blinking() {
		classes = append(classes, "tk")
	}

	if f.Invisible() {
		classes = append(classes, "tv")
	}

	if f.Underlined() {
		classes = append(classes, "tu")
	}

	if f.Inversed() {
		classes = append(classes, "trv")
	}

	if f.Strikethrough() {
		classes = append(classes, "tco")
	}

	if c := f.FgCol; c != nil {
		switch f.ColourMode {
		case ansi.Default:
			id := c.Id
			if (f.Bold() || f.Bright()) && id > 7 && id < 16 {
				id -= 8
			}

			switch id {
			case 0:
				classes = append(classes, "tfg-black")
			case 1:
				classes = append(classes, "tfg-red")
			case 2:
				classes = append(classes, "tfg-green")
			case 3:
				classes = append(classes, "tfg-yellow")
			case 4:
				classes = append(classes, "tfg-blue")
			case 5:
				classes = append(classes, "tfg-magenta")
			case 6:
				classes = append(classes, "tfg-cyan")
			case 7:
				classes = append(classes, "tfg-white")
			}

		case ansi.TrueColour:
			styles = append(styles,
				fmt.Sprintf("color: %02x%02x02x",
					c.Rgb.R, c.Rgb.G, c.Rgb.B))
		}
	}

	if c := f.BgCol; c != nil {
		switch f.ColourMode {
		case ansi.Default:
			id := c.Id
			if (f.Bold() || f.Bright()) && id > 7 && id < 16 {
				id -= 8
			}

			switch id {
			case 0:
				classes = append(classes, "tbg-black")
			case 1:
				classes = append(classes, "tbg-red")
			case 2:
				classes = append(classes, "tbg-green")
			case 3:
				classes = append(classes, "tbg-yellow")
			case 4:
				classes = append(classes, "tbg-blue")
			case 5:
				classes = append(classes, "tbg-magenta")
			case 6:
				classes = append(classes, "tbg-cyan")
			case 7:
				classes = append(classes, "tbg-white")
			}

		case ansi.TrueColour:
			styles = append(styles,
				fmt.Sprintf("background-color: %02x%02x02x",
					c.Rgb.R, c.Rgb.G, c.Rgb.B))
		}
	}

	buf.WriteString(`<span`)
	if len(classes) > 0 {
		fmt.Fprintf(buf, ` class="%s"`, strings.Join(classes, " "))
	}
	if len(styles) > 0 {
		fmt.Fprintf(buf, ` style="%s"`, strings.Join(styles, ";"))
	}
	buf.WriteByte('>')

	template.HTMLEscape(buf, []byte(f.Label))

	buf.WriteString(`</span>`)
}
