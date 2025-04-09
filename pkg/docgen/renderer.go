package docgen

import (
	"github.com/yuin/goldmark/ast"
)

// WordRenderer Markdown到Word的渲染器
type WordRenderer struct {
	mermaidRenderer *MermaidRenderer
	htmlConverter   *HtmlConverter
	docxBuilder     *DocxBuilder
	elementHandler  *WordElementHandler
}

// NewWordRenderer 创建一个新的Word渲染器
func NewWordRenderer() *WordRenderer {
	mermaidRenderer := NewMermaidRenderer()
	elementHandler := NewWordElementHandler()
	htmlConverter := NewHtmlConverter(mermaidRenderer)
	docxBuilder := NewDocxBuilder(elementHandler)

	return &WordRenderer{
		mermaidRenderer: mermaidRenderer,
		htmlConverter:   htmlConverter,
		docxBuilder:     docxBuilder,
		elementHandler:  elementHandler,
	}
}

// Render 将Markdown AST渲染为Word文档
func (r *WordRenderer) Render(node ast.Node, source []byte) ([]byte, error) {
	// 步骤1：将Markdown渲染为HTML
	htmlContent, mermaidImages, err := r.htmlConverter.ConvertMarkdownToHTML(node, source)
	if err != nil {
		return nil, err
	}

	// 步骤2：将HTML转换为DOCX格式
	docxData, err := r.docxBuilder.BuildDocx(htmlContent, mermaidImages)
	if err != nil {
		return nil, err
	}

	return docxData, nil
}
