package docgen

import (
	"bytes"
	"fmt"
	"regexp"
	"time"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
)

// HtmlConverter 负责将Markdown转换为HTML
type HtmlConverter struct {
	mermaidRenderer *MermaidRenderer
}

// NewHtmlConverter 创建一个新的HTML转换器
func NewHtmlConverter(mermaidRenderer *MermaidRenderer) *HtmlConverter {
	return &HtmlConverter{
		mermaidRenderer: mermaidRenderer,
	}
}

// ConvertMarkdownToHTML 将Markdown转换为HTML，并处理mermaid图表
func (c *HtmlConverter) ConvertMarkdownToHTML(root ast.Node, source []byte) (string, map[string][]byte, error) {
	// 创建HTML缓冲区
	var htmlBuf bytes.Buffer

	// 创建markdown到HTML的渲染器, 添加GFM扩展以支持表格和其他GitHub风格的Markdown
	md := goldmark.New(
		goldmark.WithExtensions(
			extension.GFM, // GitHub Flavored Markdown支持表格
			extension.Linkify,
		),
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
		),
		goldmark.WithRendererOptions(
			html.WithHardWraps(),
			html.WithXHTML(),
			html.WithUnsafe(), // 允许原始HTML通过
		),
	)

	// 首先将正常的markdown内容渲染为HTML
	err := md.Renderer().Render(&htmlBuf, source, root)
	if err != nil {
		return "", nil, err
	}

	// 处理所有mermaid代码块，将它们渲染为图片
	mermaidImages := make(map[string][]byte)
	htmlContent := htmlBuf.String()

	// 1. 处理标准mermaid代码块
	standardMermaidRegex := regexp.MustCompile(`<pre><code class="language-mermaid">([\s\S]*?)</code></pre>`)
	htmlContent = standardMermaidRegex.ReplaceAllStringFunc(htmlContent, func(match string) string {
		// 提取mermaid代码
		submatch := standardMermaidRegex.FindStringSubmatch(match)
		if len(submatch) < 2 {
			return match
		}
		code := submatch[1]

		// 渲染mermaid图表
		imgData, err := c.mermaidRenderer.RenderMermaid(code)
		if err != nil {
			return match // 保持原样
		}

		// 生成唯一的图片ID并保存图片数据
		imageID := fmt.Sprintf("mermaid-%d", time.Now().UnixNano())
		mermaidImages[imageID] = imgData

		// 返回图片HTML
		return fmt.Sprintf("<p><img src=\"%s\" alt=\"Mermaid Diagram\" /></p>", imageID)
	})

	// 2. 处理其他mermaid变体（xychart-beta, pie等）
	otherMermaidRegex := regexp.MustCompile(`<pre><code(?:\s+class="language-([^"]*)")?>([\s\S]*?(?:xychart|pie\s+title|graph\s|sequenceDiagram|classDiagram|flowchart)[\s\S]*?)</code></pre>`)
	htmlContent = otherMermaidRegex.ReplaceAllStringFunc(htmlContent, func(match string) string {
		submatch := otherMermaidRegex.FindStringSubmatch(match)
		if len(submatch) < 3 {
			return match
		}

		// 检查是否已经是mermaid代码块(避免重复处理)
		if submatch[1] == "mermaid" {
			return match
		}

		code := submatch[2]

		// 尝试渲染为mermaid
		imgData, err := c.mermaidRenderer.RenderMermaid(code)
		if err != nil {
			return match // 保持原样
		}

		// 生成唯一的图片ID并保存图片数据
		imageID := fmt.Sprintf("mermaid-%d", time.Now().UnixNano())
		mermaidImages[imageID] = imgData

		// 返回图片HTML
		return fmt.Sprintf("<p><img src=\"%s\" alt=\"Mermaid Diagram\" /></p>", imageID)
	})

	// 调试输出HTML内容
	fmt.Println("HTML Content:", htmlContent)

	return htmlContent, mermaidImages, nil
}

// RemoveHtmlAttributes 移除HTML标签中的属性
func RemoveHtmlAttributes(html string) string {
	// 移除id属性
	idRegExp := regexp.MustCompile(` id="[^"]*"`)
	html = idRegExp.ReplaceAllString(html, "")

	// 移除class属性
	classRegExp := regexp.MustCompile(` class="[^"]*"`)
	html = classRegExp.ReplaceAllString(html, "")

	return html
}

// GetCodeBlockContent 获取代码块内容
func GetCodeBlockContent(n *ast.FencedCodeBlock, source []byte) string {
	var content bytes.Buffer
	lines := n.Lines()
	for i := 0; i < lines.Len(); i++ {
		line := lines.At(i)
		content.Write(line.Value(source))
	}
	return content.String()
}
