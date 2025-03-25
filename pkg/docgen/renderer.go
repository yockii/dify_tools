package docgen

import (
	"archive/zip"
	"bytes"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
)

// WordRenderer Markdown到Word的渲染器
type WordRenderer struct {
	mermaidRenderer *MermaidRenderer
}

// NewWordRenderer 创建一个新的Word渲染器
func NewWordRenderer() *WordRenderer {
	return &WordRenderer{
		mermaidRenderer: NewMermaidRenderer(),
	}
}

// Render 将Markdown AST渲染为Word文档
func (r *WordRenderer) Render(node ast.Node, source []byte) ([]byte, error) {
	// 步骤1：将Markdown渲染为HTML
	htmlContent, mermaidImages, err := r.convertMarkdownToHTML(node, source)
	if err != nil {
		return nil, err
	}

	// 步骤2：将HTML转换为DOCX格式
	docxData, err := r.convertHTMLToDOCX(htmlContent, mermaidImages)
	if err != nil {
		return nil, err
	}

	return docxData, nil
}

// convertMarkdownToHTML 将Markdown转换为HTML，并处理mermaid图表
func (r *WordRenderer) convertMarkdownToHTML(root ast.Node, source []byte) (string, map[string][]byte, error) {
	// 创建HTML缓冲区
	var htmlBuf bytes.Buffer

	// 创建markdown到HTML的渲染器
	md := goldmark.New(
		goldmark.WithExtensions(extension.GFM),
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
		),
		goldmark.WithRendererOptions(
			html.WithHardWraps(),
			html.WithXHTML(),
		),
	)

	// 首先将正常的markdown内容渲染为HTML
	err := md.Renderer().Render(&htmlBuf, source, root)
	if err != nil {
		return "", nil, err
	}

	// 处理所有mermaid代码块，将它们渲染为图片
	mermaidImages := make(map[string][]byte)

	err = ast.Walk(root, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}

		if n.Kind() == ast.KindFencedCodeBlock {
			cb := n.(*ast.FencedCodeBlock)
			language := string(cb.Language(source))

			// 只处理mermaid代码块
			if language == "mermaid" {
				// 提取mermaid代码
				code := getCodeBlockContent(cb, source)

				// 渲染mermaid图表
				imgData, err := r.mermaidRenderer.RenderMermaid(code)
				if err != nil {
					return ast.WalkContinue, nil // 忽略错误，继续处理
				}

				// 生成唯一的图片ID
				imageID := fmt.Sprintf("mermaid-%d", time.Now().UnixNano())
				mermaidImages[imageID] = imgData

				// 在HTML中用图片占位符替换mermaid代码块
				replacedBytes := bytes.Replace(
					htmlBuf.Bytes(),
					[]byte(fmt.Sprintf("<pre><code class=\"language-mermaid\">%s</code></pre>", code)),
					[]byte(fmt.Sprintf("<p><img src=\"%s\" alt=\"Mermaid Diagram\" /></p>", imageID)),
					1,
				)

				// 重置缓冲区并写入新内容
				htmlBuf.Reset()
				htmlBuf.Write(replacedBytes)
			}
		}

		return ast.WalkContinue, nil
	})

	if err != nil {
		return "", nil, err
	}

	// 调试输出HTML内容
	fmt.Println("HTML Content:", htmlBuf.String())

	return htmlBuf.String(), mermaidImages, nil
}

// getCodeBlockContent 获取代码块内容
func getCodeBlockContent(n *ast.FencedCodeBlock, source []byte) string {
	var content bytes.Buffer
	lines := n.Lines()
	for i := 0; i < lines.Len(); i++ {
		line := lines.At(i)
		content.Write(line.Value(source))
	}
	return content.String()
}

// convertHTMLToDOCX 将HTML转换为DOCX格式 - 完全在内存中操作
func (r *WordRenderer) convertHTMLToDOCX(htmlContent string, mermaidImages map[string][]byte) ([]byte, error) {
	// 创建一个用于存储输出的缓冲区
	outputBuffer := new(bytes.Buffer)
	zipWriter := zip.NewWriter(outputBuffer)

	// 更新Content Types以包含numbering.xml
	contentTypesXML := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
  <Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>
  <Default Extension="xml" ContentType="application/xml"/>
  <Default Extension="png" ContentType="image/png"/>
  <Default Extension="jpg" ContentType="image/jpeg"/>
  <Default Extension="jpeg" ContentType="image/jpeg"/>
  <Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/>
  <Override PartName="/word/styles.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.styles+xml"/>
  <Override PartName="/word/numbering.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.numbering+xml"/>
</Types>`

	relsXML := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="word/document.xml"/>
</Relationships>`

	// 更新Word关系XML以包含numbering.xml
	wordRelsXML := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/styles" Target="styles.xml"/>
  <Relationship Id="rId2" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/numbering" Target="numbering.xml"/>
`

	// 添加更多样式支持，包括列表、引用和表格样式
	stylesXML := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:styles xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:style w:type="paragraph" w:styleId="Normal">
    <w:name w:val="Normal"/>
    <w:qFormat/>
    <w:pPr>
      <w:spacing w:after="0" w:line="240" w:lineRule="auto"/>
    </w:pPr>
  </w:style>
  <w:style w:type="paragraph" w:styleId="Heading1">
    <w:name w:val="heading 1"/>
    <w:basedOn w:val="Normal"/>
    <w:next w:val="Normal"/>
    <w:qFormat/>
    <w:pPr>
      <w:keepNext/>
      <w:spacing w:before="480" w:after="0"/>
      <w:outlineLvl w:val="0"/>
    </w:pPr>
    <w:rPr>
      <w:b/>
      <w:sz w:val="36"/>
      <w:szCs w:val="36"/>
    </w:rPr>
  </w:style>
  <w:style w:type="paragraph" w:styleId="Heading2">
    <w:name w:val="heading 2"/>
    <w:basedOn w:val="Normal"/>
    <w:next w:val="Normal"/>
    <w:qFormat/>
    <w:pPr>
      <w:keepNext/>
      <w:spacing w:before="360" w:after="0"/>
      <w:outlineLvl w:val="1"/>
    </w:pPr>
    <w:rPr>
      <w:b/>
      <w:sz w:val="32"/>
      <w:szCs w:val="32"/>
    </w:rPr>
  </w:style>
  <w:style w:type="paragraph" w:styleId="Heading3">
    <w:name w:val="heading 3"/>
    <w:basedOn w:val="Normal"/>
    <w:next w:val="Normal"/>
    <w:qFormat/>
    <w:pPr>
      <w:keepNext/>
      <w:spacing w:before="280" w:after="0"/>
      <w:outlineLvl w:val="2"/>
    </w:pPr>
    <w:rPr>
      <w:b/>
      <w:sz w:val="28"/>
      <w:szCs w:val="28"/>
    </w:rPr>
  </w:style>
  <w:style w:type="paragraph" w:styleId="ListParagraph">
    <w:name w:val="List Paragraph"/>
    <w:basedOn w:val="Normal"/>
    <w:qFormat/>
    <w:pPr>
      <w:ind w:left="720"/>
      <w:contextualSpacing/>
    </w:pPr>
  </w:style>
  <w:style w:type="paragraph" w:styleId="Quote">
    <w:name w:val="Quote"/>
    <w:basedOn w:val="Normal"/>
    <w:qFormat/>
    <w:pPr>
      <w:ind w:left="720" w:right="720"/>
      <w:spacing w:before="120" w:after="120"/>
    </w:pPr>
    <w:rPr>
      <w:i/>
      <w:color w:val="666666"/>
    </w:rPr>
  </w:style>
  <w:style w:type="table" w:styleId="TableGrid">
    <w:name w:val="Table Grid"/>
    <w:uiPriority w:val="59"/>
    <w:rsid w:val="00000000"/>
    <w:pPr>
      <w:spacing w:after="0" w:line="240" w:lineRule="auto"/>
    </w:pPr>
    <w:tblPr>
      <w:tblBorders>
        <w:top w:val="single" w:sz="4" w:space="0" w:color="auto"/>
        <w:left w:val="single" w:sz="4" w:space="0" w:color="auto"/>
        <w:bottom w:val="single" w:sz="4" w:space="0" w:color="auto"/>
        <w:right w:val="single" w:sz="4" w:space="0" w:color="auto"/>
        <w:insideH w:val="single" w:sz="4" w:space="0" w:color="auto"/>
        <w:insideV w:val="single" w:sz="4" w:space="0" w:color="auto"/>
      </w:tblBorders>
    </w:tblPr>
  </w:style>
</w:styles>`

	// 添加numbering.xml文件来支持列表
	numberingXML := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:numbering xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:abstractNum w:abstractNumId="0">
    <w:nsid w:val="12345678"/>
    <w:multiLevelType w:val="hybridMultilevel"/>
    <w:lvl w:ilvl="0">
      <w:start w:val="1"/>
      <w:numFmt w:val="bullet"/>
      <w:lvlText w:val="•"/>
      <w:lvlJc w:val="left"/>
      <w:pPr>
        <w:ind w:left="720" w:hanging="360"/>
      </w:pPr>
      <w:rPr>
        <w:rFonts w:ascii="Symbol" w:hAnsi="Symbol" w:hint="default"/>
      </w:rPr>
    </w:lvl>
    <w:lvl w:ilvl="1">
      <w:start w:val="1"/>
      <w:numFmt w:val="bullet"/>
      <w:lvlText w:val="○"/>
      <w:lvlJc w:val="left"/>
      <w:pPr>
        <w:ind w:left="1440" w:hanging="360"/>
      </w:pPr>
      <w:rPr>
        <w:rFonts w:ascii="Symbol" w:hAnsi="Symbol" w:hint="default"/>
      </w:rPr>
    </w:lvl>
  </w:abstractNum>
  <w:num w:numId="1">
    <w:abstractNumId w:val="0"/>
  </w:num>
</w:numbering>`

	// 处理嵌入的图片
	imgIdx := 1
	mediaRels := ""
	for imageID, imageData := range mermaidImages {
		// 添加图片关系
		relID := fmt.Sprintf("rId%d", imgIdx+2) // +2是因为rId1和rId2已用于styles和numbering
		imageName := fmt.Sprintf("image%d.png", imgIdx)
		mediaRels += fmt.Sprintf(`  <Relationship Id="%s" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/image" Target="media/%s"/>
`, relID, imageName)

		// 替换HTML中的图片引用
		htmlContent = strings.Replace(
			htmlContent,
			fmt.Sprintf(`<img src="%s"`, imageID),
			fmt.Sprintf(`<img src="media/%s" relId="%s"`, imageName, relID),
			1,
		)

		// 在内存中保存图片文件到ZIP
		imageEntry, err := zipWriter.Create("word/media/" + imageName)
		if err != nil {
			return nil, err
		}
		_, err = imageEntry.Write(imageData)
		if err != nil {
			return nil, err
		}

		imgIdx++
	}

	// 完成word关系XML
	wordRelsXML += mediaRels + "</Relationships>"

	// 预处理HTML内容，移除HTML属性，然后将HTML转换为Word XML
	htmlContent = removeHtmlAttributes(htmlContent)
	documentXML := r.htmlToWordXML(htmlContent)

	// 在内存中写入所有XML文件到ZIP
	files := map[string]string{
		"[Content_Types].xml":          contentTypesXML,
		"_rels/.rels":                  relsXML,
		"word/_rels/document.xml.rels": wordRelsXML,
		"word/styles.xml":              stylesXML,
		"word/numbering.xml":           numberingXML,
		"word/document.xml":            documentXML,
	}

	for path, content := range files {
		entry, err := zipWriter.Create(path)
		if err != nil {
			return nil, err
		}
		_, err = entry.Write([]byte(content))
		if err != nil {
			return nil, err
		}
	}

	// 关闭ZIP writer
	err := zipWriter.Close()
	if err != nil {
		return nil, err
	}

	return outputBuffer.Bytes(), nil
}

// removeHtmlAttributes 移除HTML标签中的属性
func removeHtmlAttributes(html string) string {
	// 移除id属性
	idRegExp := regexp.MustCompile(` id="[^"]*"`)
	html = idRegExp.ReplaceAllString(html, "")

	// 移除class属性
	classRegExp := regexp.MustCompile(` class="[^"]*"`)
	html = classRegExp.ReplaceAllString(html, "")

	return html
}

// htmlToWordXML 将HTML内容转换为Word XML
func (r *WordRenderer) htmlToWordXML(htmlContent string) string {
	// 添加XML头和文档开始标记
	docXML := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main" 
            xmlns:wp="http://schemas.openxmlformats.org/drawingml/2006/wordprocessingDrawing"
            xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main"
            xmlns:pic="http://schemas.openxmlformats.org/drawingml/2006/picture"
            xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships"
            xmlns:ns9="http://schemas.openxmlformats.org/schemaLibrary/2006/main">
  <w:body>
`

	// 预处理：确保HTML标签之间的空白不会影响转换
	htmlContent = strings.ReplaceAll(htmlContent, "> <", "><")

	// 处理表格 - 在其他处理之前
	htmlContent = processTables(htmlContent)

	// 处理列表 - 在处理其他标签之前
	htmlContent = processLists(htmlContent)

	// 处理引用块
	htmlContent = strings.ReplaceAll(htmlContent, "<blockquote>", "<w:p><w:pPr><w:pStyle w:val=\"Quote\"/></w:pPr><w:r><w:t>")
	htmlContent = strings.ReplaceAll(htmlContent, "</blockquote>", "</w:t></w:r></w:p>")

	// 处理标题和段落
	htmlContent = strings.ReplaceAll(htmlContent, "<h1>", "<w:p><w:pPr><w:pStyle w:val=\"Heading1\"/></w:pPr><w:r><w:t>")
	htmlContent = strings.ReplaceAll(htmlContent, "</h1>", "</w:t></w:r></w:p>")
	htmlContent = strings.ReplaceAll(htmlContent, "<h2>", "<w:p><w:pPr><w:pStyle w:val=\"Heading2\"/></w:pPr><w:r><w:t>")
	htmlContent = strings.ReplaceAll(htmlContent, "</h2>", "</w:t></w:r></w:p>")
	htmlContent = strings.ReplaceAll(htmlContent, "<h3>", "<w:p><w:pPr><w:pStyle w:val=\"Heading3\"/></w:pPr><w:r><w:t>")
	htmlContent = strings.ReplaceAll(htmlContent, "</h3>", "</w:t></w:r></w:p>")
	htmlContent = strings.ReplaceAll(htmlContent, "<h4>", "<w:p><w:pPr><w:pStyle w:val=\"Heading3\"/></w:pPr><w:r><w:t>")
	htmlContent = strings.ReplaceAll(htmlContent, "</h4>", "</w:t></w:r></w:p>")

	// 处理段落和文本格式
	htmlContent = strings.ReplaceAll(htmlContent, "<p>", "<w:p><w:r><w:t>")
	htmlContent = strings.ReplaceAll(htmlContent, "</p>", "</w:t></w:r></w:p>")
	htmlContent = strings.ReplaceAll(htmlContent, "<strong>", "<w:r><w:rPr><w:b/></w:rPr><w:t>")
	htmlContent = strings.ReplaceAll(htmlContent, "</strong>", "</w:t></w:r>")
	htmlContent = strings.ReplaceAll(htmlContent, "<em>", "<w:r><w:rPr><w:i/></w:rPr><w:t>")
	htmlContent = strings.ReplaceAll(htmlContent, "</em>", "</w:t></w:r>")

	// 处理换行
	htmlContent = strings.ReplaceAll(htmlContent, "<br>", "</w:t></w:r></w:p><w:p><w:r><w:t>")
	htmlContent = strings.ReplaceAll(htmlContent, "<br/>", "</w:t></w:r></w:p><w:p><w:r><w:t>")

	// 处理字符实体
	htmlContent = strings.ReplaceAll(htmlContent, "&lt;", "<")
	htmlContent = strings.ReplaceAll(htmlContent, "&gt;", ">")
	htmlContent = strings.ReplaceAll(htmlContent, "&amp;", "&")
	htmlContent = strings.ReplaceAll(htmlContent, "&quot;", "\"")

	// 处理图片
	for {
		imgStartIdx := strings.Index(htmlContent, "<img src=\"")
		if imgStartIdx == -1 {
			break
		}

		imgEndIdx := strings.Index(htmlContent[imgStartIdx:], "/>")
		if imgEndIdx == -1 {
			break
		}
		imgEndIdx += imgStartIdx

		imgTag := htmlContent[imgStartIdx : imgEndIdx+2]

		// 解析图片标签
		relId := ""
		if relIdStartIdx := strings.Index(imgTag, "relId=\""); relIdStartIdx != -1 {
			relIdStartIdx += 7 // len("relId=\"")
			relIdEndIdx := strings.Index(imgTag[relIdStartIdx:], "\"")
			if relIdEndIdx != -1 {
				relIdEndIdx += relIdStartIdx
				relId = imgTag[relIdStartIdx:relIdEndIdx]
			}
		}

		// 创建图片的Word XML
		imgXML := fmt.Sprintf(`<w:p>
  <w:r>
    <w:drawing>
      <wp:inline>
        <wp:extent cx="4572000" cy="3429000"/>
        <wp:docPr id="1" name="Image"/>
        <a:graphic>
          <a:graphicData uri="http://schemas.openxmlformats.org/drawingml/2006/picture">
            <pic:pic>
              <pic:blipFill>
                <a:blip r:embed="%s"/>
                <a:stretch>
                  <a:fillRect/>
                </a:stretch>
              </pic:blipFill>
              <pic:spPr>
                <a:xfrm>
                  <a:off x="0" y="0"/>
                  <a:ext cx="4572000" cy="3429000"/>
                </a:xfrm>
                <a:prstGeom prst="rect">
                  <a:avLst/>
                </a:prstGeom>
              </pic:spPr>
            </pic:pic>
          </a:graphicData>
        </a:graphic>
      </wp:inline>
    </w:drawing>
  </w:r>
</w:p>`, relId)

		// 替换图片标签
		htmlContent = htmlContent[:imgStartIdx] + imgXML + htmlContent[imgEndIdx+2:]
	}

	// 调试输出最终的XML内容
	fmt.Println("Final Word XML:", htmlContent)

	// 添加文档结束标记
	docXML += htmlContent + `  </w:body>
</w:document>`

	return docXML
}

// processLists 处理HTML中的列表
func processLists(html string) string {
	// 首先移除无序列表标记
	html = strings.ReplaceAll(html, "<ul>", "")
	html = strings.ReplaceAll(html, "</ul>", "")

	// 处理列表项
	listItemRegex := regexp.MustCompile(`<li>(.*?)</li>`)
	html = listItemRegex.ReplaceAllStringFunc(html, func(match string) string {
		// 提取列表项内容
		content := listItemRegex.FindStringSubmatch(match)[1]

		// 返回Word格式的列表项
		return fmt.Sprintf(`<w:p>
  <w:pPr>
    <w:pStyle w:val="ListParagraph"/>
    <w:numPr>
      <w:ilvl w:val="0"/>
      <w:numId w:val="1"/>
    </w:numPr>
  </w:pPr>
  <w:r>
    <w:t>%s</w:t>
  </w:r>
</w:p>`, content)
	})

	return html
}

// processTables 处理HTML中的表格
func processTables(html string) string {
	// 查找所有表格
	tableRegex := regexp.MustCompile(`<table>([\s\S]*?)</table>`)

	return tableRegex.ReplaceAllStringFunc(html, func(tableHTML string) string {
		// 提取表格内容
		tableContent := tableRegex.FindStringSubmatch(tableHTML)[1]

		// 解析表格行
		rows := regexp.MustCompile(`<tr>([\s\S]*?)</tr>`).FindAllStringSubmatch(tableContent, -1)
		if len(rows) == 0 {
			return tableHTML // 没有行，返回原始HTML
		}

		// 开始构建Word表格
		tableXML := `<w:tbl>
  <w:tblPr>
    <w:tblStyle w:val="TableGrid"/>
    <w:tblW w:w="5000" w:type="pct"/>
    <w:tblBorders>
      <w:top w:val="single" w:sz="4" w:space="0" w:color="auto"/>
      <w:left w:val="single" w:sz="4" w:space="0" w:color="auto"/>
      <w:bottom w:val="single" w:sz="4" w:space="0" w:color="auto"/>
      <w:right w:val="single" w:sz="4" w:space="0" w:color="auto"/>
      <w:insideH w:val="single" w:sz="4" w:space="0" w:color="auto"/>
      <w:insideV w:val="single" w:sz="4" w:space="0" w:color="auto"/>
    </w:tblBorders>
  </w:tblPr>
  <w:tblGrid>`

		// 计算列数（基于第一行的单元格数量）
		firstRow := rows[0][1]
		cols := regexp.MustCompile(`<t[hd]>([\s\S]*?)</t[hd]>`).FindAllString(firstRow, -1)
		colCount := len(cols)

		// 添加列定义
		for i := 0; i < colCount; i++ {
			tableXML += fmt.Sprintf(`
    <w:gridCol w:w="%d"/>`, 5000/colCount)
		}

		tableXML += `
  </w:tblGrid>`

		// 处理每一行
		for rowIndex, row := range rows {
			rowContent := row[1]
			tableXML += `
  <w:tr>
    <w:trPr>
      <w:cantSplit/>
    </w:trPr>`

			// 处理单元格
			cells := regexp.MustCompile(`<t[hd]>([\s\S]*?)</t[hd]>`).FindAllStringSubmatch(rowContent, -1)
			for _, cell := range cells {
				cellContent := cell[1]

				// 检查是否为标题单元格 - 第一行或者是th标签
				isHeader := rowIndex == 0 || strings.Contains(cell[0], "<th>")

				tableXML += `
    <w:tc>
      <w:tcPr>
        <w:tcW w:w="0" w:type="auto"/>`

				if isHeader {
					tableXML += `
        <w:shd w:val="clear" w:color="auto" w:fill="E7E6E6"/>`
				}

				tableXML += `
      </w:tcPr>
      <w:p>
        <w:pPr>
          <w:spacing w:before="0" w:after="0"/>
          <w:jc w:val="center"/>
        </w:pPr>
        <w:r>
          <w:rPr>`

				if isHeader {
					tableXML += `
            <w:b/>`
				}

				tableXML += `
          </w:rPr>
          <w:t>` + cellContent + `</w:t>
        </w:r>
      </w:p>
    </w:tc>`
			}

			tableXML += `
  </w:tr>`
		}

		tableXML += `
</w:tbl>`

		return tableXML
	})
}
