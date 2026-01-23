package content

import (
	"bytes"
	"fmt"

	"github.com/yuin/goldmark"
	emoji "github.com/yuin/goldmark-emoji"
	highlighting "github.com/yuin/goldmark-highlighting/v2"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
)

var mdEngine = goldmark.New(
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

func mdToHTML(source []byte) ([]byte, error) {
	var buf bytes.Buffer
	// html output is larger than MD so add 50% to the buffer
	buf.Grow(len(source) + (len(source) / 2))

	if err := mdEngine.Convert(source, &buf); err != nil {
		return []byte{}, fmt.Errorf("%w: %v", ErrMDConversion, err)

	}

	// worth trading CPU time for RAM?
	return bytes.Clone(buf.Bytes()), nil
}
