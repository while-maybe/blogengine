package content

import (
	"bytes"

	"github.com/yuin/goldmark"
	emoji "github.com/yuin/goldmark-emoji"
	highlighting "github.com/yuin/goldmark-highlighting/v2"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
)

// func mdToHTML(md []byte) []byte {
// 	extensions := parser.CommonExtensions | parser.AutoHeadingIDs | parser.NoEmptyLineBeforeBlock
// 	p := parser.NewWithExtensions(extensions)
// 	doc := p.Parse(md)

// 	htmlFlags := html.CommonFlags | html.HrefTargetBlank
// 	opts := html.RendererOptions{Flags: htmlFlags}
// 	renderer := html.NewRenderer(opts)
// 	bodyContent := markdown.Render(doc, renderer)

// 	// htmlTemplate := `%s`

// 	// return []byte(fmt.Sprintf(htmlTemplate, bodyContent))
// 	return bodyContent
// }

func mdToHTML(source []byte) []byte {
	var buf bytes.Buffer
	// html output is larger than MD so add 50% to the buffer
	buf.Grow(len(source) + (len(source) / 2))

	md := goldmark.New(
		goldmark.WithExtensions(
			extension.Table,
			extension.Strikethrough,
			extension.Linkify,
			extension.TaskList,
			emoji.Emoji,
			highlighting.NewHighlighting(
				// Common themes: "monokai", "dracula", "github", "solarized-dark"
				highlighting.WithStyle("solarized-dark"),

				// Optional: Fallback if language isn't recognized
				highlighting.WithGuessLanguage(true),
			),
		),
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
		),
	)

	if err := md.Convert(source, &buf); err != nil {
		panic(err)
	}

	// worth trading CPU time for RAM?
	return bytes.Clone(buf.Bytes())
}
