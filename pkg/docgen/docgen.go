package docgen

import (
	"bytes"
	"io"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
)

// DocGenerator Word文档生成器
type DocGenerator struct {
	markdown goldmark.Markdown
	renderer *WordRenderer
}

// NewDocGenerator 创建一个新的Word文档生成器
func NewDocGenerator() *DocGenerator {
	return &DocGenerator{
		markdown: goldmark.New(
			goldmark.WithParserOptions(
				parser.WithAutoHeadingID(),
			),
		),
		renderer: NewWordRenderer(),
	}
}

// RenderBytes 从Markdown字节数据生成Word文档
func (g *DocGenerator) RenderBytes(source []byte) ([]byte, error) {
	// 创建文本Reader
	reader := text.NewReader(source)

	// 解析Markdown
	doc := g.markdown.Parser().Parse(reader)

	// 渲染为Word
	output, err := g.renderer.Render(doc, source)
	if err != nil {
		return nil, err
	}

	return output, nil
}

// RenderString 从Markdown字符串生成Word文档
func (g *DocGenerator) RenderString(source string) ([]byte, error) {
	return g.RenderBytes([]byte(source))
}

// RenderReader 从Reader读取Markdown并生成Word文档
func (g *DocGenerator) RenderReader(reader io.Reader) ([]byte, error) {
	buf := new(bytes.Buffer)
	_, err := io.Copy(buf, reader)
	if err != nil {
		return nil, err
	}
	return g.RenderBytes(buf.Bytes())
}
