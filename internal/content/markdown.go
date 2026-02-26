package content

import (
	"bytes"
	"fmt"
	"net/url"
	"strings"

	"github.com/yuin/goldmark"
	emoji "github.com/yuin/goldmark-emoji"
	highlighting "github.com/yuin/goldmark-highlighting/v2"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

type MarkDownRenderer struct {
	assets MediaService
	engine goldmark.Markdown
}

func NewMarkDownRenderer(assets MediaService) *MarkDownRenderer {
	m := &MarkDownRenderer{assets: assets}

	m.engine = goldmark.New(
		goldmark.WithRendererOptions(
			renderer.WithNodeRenderers(
				util.Prioritized(m, 100),
			),
		),
		goldmark.WithExtensions(
			extension.Table,
			extension.Strikethrough,
			extension.Linkify,
			extension.TaskList,
			emoji.Emoji,
			highlighting.NewHighlighting(
				// Common themes: "monokai", "dracula", "github", "solarized-dark"
				highlighting.WithStyle("solarized-dark"),
				highlighting.WithGuessLanguage(true),
			),
		),
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
			parser.WithASTTransformers(util.Prioritized(&assetTransformer{assets: assets}, 100)),
		),
	)
	return m
}

func (m *MarkDownRenderer) Render(source []byte) ([]byte, error) {
	var buf bytes.Buffer
	// html output is larger than markdown add 50% to the buffer
	buf.Grow(len(source) + (len(source) / 2))

	if err := m.engine.Convert(source, &buf); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrMDConversion, err)
	}

	// worth trading CPU time for RAM?
	return bytes.Clone(buf.Bytes()), nil
}

type assetTransformer struct {
	assets MediaService
}

func (a *assetTransformer) Transform(node *ast.Document, reader text.Reader, pc parser.Context) {
	ast.Walk(node, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		// walk has finished
		if !entering {
			return ast.WalkContinue, nil
		}

		img, ok := n.(*ast.Image)
		if !ok {
			return ast.WalkContinue, nil
		}

		originPath := string(img.Destination)
		if isExternalLink(originPath) {
			return ast.WalkContinue, nil
		}

		id, err := a.assets.Obfuscate(originPath)
		if err != nil {
			return ast.WalkContinue, err
		}

		newPath, err := url.JoinPath("/assets/", id.String())
		if err != nil {
			return ast.WalkContinue, err
		}

		img.Destination = []byte(newPath)

		return ast.WalkContinue, nil
	})

}

func isExternalLink(s string) bool {
	s = strings.ToLower(s)

	for _, prefix := range []string{"http", "https", "ftp", "ftps", "sftp"} {
		if strings.HasPrefix(s, prefix) {
			return true
		}
	}
	return false
}

func (m *MarkDownRenderer) renderImage(w util.BufWriter, _ []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if !entering {
		return ast.WalkContinue, nil
	}
	n := node.(*ast.Image)
	dest := string(n.Destination)

	srcset := fmt.Sprintf("%[1]s_800 800w, %[1]s_1200 1200w, %[1]s_1920 1920w", dest)
	// TODO check the 1200px width is accurate for desktop screens
	sizes := "(max-width: 800px) 100vw, (max-width: 1200px) 90vw, 1200px"

	title := string(n.Title)
	altText := ""

	imgTag := fmt.Sprintf(
		`<img src="%s_800" srcset="%s" sizes="%s" alt="%s" title="%s" loading="lazy" class="post-image">`,
		dest, srcset, sizes, altText, title,
	)

	_, _ = w.WriteString(imgTag)

	return ast.WalkSkipChildren, nil
}

func (m *MarkDownRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(ast.KindImage, m.renderImage)
}
