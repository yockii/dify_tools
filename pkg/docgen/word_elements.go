package docgen

import (
	"fmt"
	"regexp"
	"strings"
)

// WordElementHandler 处理Word文档元素
type WordElementHandler struct{}

// NewWordElementHandler 创建Word元素处理器
func NewWordElementHandler() *WordElementHandler {
	return &WordElementHandler{}
}

// ConvertHtmlToWordXml 将HTML内容转换为Word XML
func (h *WordElementHandler) ConvertHtmlToWordXml(htmlContent string) string {
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
	htmlContent = h.processTables(htmlContent)

	// 处理Markdown格式的表格
	htmlContent = h.processMarkdownTables(htmlContent)

	// 处理列表 - 在处理其他标签之前
	htmlContent = h.processLists(htmlContent)

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
	htmlContent = h.processImages(htmlContent)

	// 调试输出最终的XML内容
	fmt.Println("Final Word XML:", htmlContent)

	// 添加文档结束标记
	docXML += htmlContent + `  </w:body>
</w:document>`

	return docXML
}

// processImages 处理HTML中的图片
func (h *WordElementHandler) processImages(html string) string {
	for {
		imgStartIdx := strings.Index(html, "<img src=\"")
		if imgStartIdx == -1 {
			break
		}

		imgEndIdx := strings.Index(html[imgStartIdx:], "/>")
		if imgEndIdx == -1 {
			break
		}
		imgEndIdx += imgStartIdx

		imgTag := html[imgStartIdx : imgEndIdx+2]

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

		imageID := ""
		if srcStartIdx := strings.Index(imgTag, "src=\""); srcStartIdx != -1 {
			srcStartIdx += 5 // len("src=\"")
			srcEndIdx := strings.Index(imgTag[srcStartIdx:], "\"")
			if srcEndIdx != -1 {
				srcEndIdx += srcStartIdx
				imageID = imgTag[srcStartIdx:srcEndIdx]
			}
		}

		// 创建图片的Word XML，使用特殊标记保存RelId以便后续替换
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
                <img:RelId>%s</img:RelId>
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
</w:p>`, relId, imageID)

		// 替换图片标签
		html = html[:imgStartIdx] + imgXML + html[imgEndIdx+2:]
	}

	return html
}

// processLists 处理HTML中的列表
func (h *WordElementHandler) processLists(html string) string {
	// 首先移除无序列表标记
	html = strings.ReplaceAll(html, "<ul>", "")
	html = strings.ReplaceAll(html, "</ul>", "")

	// 处理列表项
	listItemRegex := regexp.MustCompile(`<li>([\s\S]*?)</li>`)
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
func (h *WordElementHandler) processTables(html string) string {
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

// processMarkdownTables 处理Markdown风格表格（文本形式）
func (h *WordElementHandler) processMarkdownTables(html string) string {
	// 寻找表格文本模式: |header1|header2|...\n|---|---| 或类似格式
	tableRegex := regexp.MustCompile(`<w:p><w:r><w:t>\|.*?\|.*?\|[\s\S]*?</w:t></w:r></w:p>`)

	return tableRegex.ReplaceAllStringFunc(html, func(match string) string {
		// 提取表格文本
		tableTextRegex := regexp.MustCompile(`<w:t>([\s\S]*?)</w:t>`)
		tableTextMatches := tableTextRegex.FindAllStringSubmatch(match, -1)

		if len(tableTextMatches) == 0 {
			return match
		}

		tableText := ""
		for _, m := range tableTextMatches {
			if len(m) > 1 {
				tableText += m[1]
			}
		}

		// 按行分割
		rows := strings.Split(tableText, "<br />")
		if len(rows) < 2 {
			rows = strings.Split(tableText, "\n")
		}

		// 检查是否像是表格（至少要有标题行和分隔行）
		if len(rows) < 2 || !strings.Contains(rows[1], "---") {
			return match
		}

		// 解析标题行
		headerRow := rows[0]
		headers := h.parseTableRow(headerRow)

		// 跳过分隔行，处理数据行
		dataRows := make([][]string, 0)
		for i := 2; i < len(rows); i++ {
			if len(strings.TrimSpace(rows[i])) == 0 {
				continue
			}

			row := h.parseTableRow(rows[i])
			if len(row) > 0 {
				dataRows = append(dataRows, row)
			}
		}

		// 生成Word表格
		return h.generateWordTable(headers, dataRows)
	})
}

// parseTableRow 解析表格行，提取单元格
func (h *WordElementHandler) parseTableRow(row string) []string {
	// 去除首尾空白
	row = strings.TrimSpace(row)

	// 移除首尾的|（如果存在）
	if strings.HasPrefix(row, "|") {
		row = row[1:]
	}
	if strings.HasSuffix(row, "|") {
		row = row[:len(row)-1]
	}

	// 按|分割单元格并清理
	cells := strings.Split(row, "|")
	for i, cell := range cells {
		cells[i] = strings.TrimSpace(cell)
	}

	return cells
}

// generateWordTable 生成Word表格XML
func (h *WordElementHandler) generateWordTable(headers []string, rows [][]string) string {
	columnCount := len(headers)
	if columnCount == 0 {
		return ""
	}

	// 构建表格
	tableXML := `<w:tbl>
  <w:tblPr>
    <w:tblStyle w:val="TableGrid"/>
    <w:tblW w:w="5000" w:type="pct"/>
    <w:tblLook w:val="04A0" w:firstRow="1" w:lastRow="0" w:firstColumn="1" w:lastColumn="0" w:noHBand="0" w:noVBand="1"/>
  </w:tblPr>
  <w:tblGrid>`

	// 列定义
	for i := 0; i < columnCount; i++ {
		tableXML += fmt.Sprintf(`
    <w:gridCol w:w="%d"/>`, 5000/columnCount)
	}

	tableXML += `
  </w:tblGrid>`

	// 表头行
	tableXML += `
  <w:tr>
    <w:trPr>
      <w:tblHeader/>
    </w:trPr>`

	for _, header := range headers {
		tableXML += `
    <w:tc>
      <w:tcPr>
        <w:shd w:val="clear" w:color="auto" w:fill="E7E6E6"/>
      </w:tcPr>
      <w:p>
        <w:pPr>
          <w:jc w:val="center"/>
        </w:pPr>
        <w:r>
          <w:rPr>
            <w:b/>
          </w:rPr>
          <w:t>` + header + `</w:t>
        </w:r>
      </w:p>
    </w:tc>`
	}

	tableXML += `
  </w:tr>`

	// 数据行
	for _, row := range rows {
		tableXML += `
  <w:tr>`

		// 确保单元格数不超过列数
		cellCount := len(row)
		if cellCount > columnCount {
			cellCount = columnCount
		}

		// 添加实际存在的单元格
		for i := 0; i < cellCount; i++ {
			tableXML += `
    <w:tc>
      <w:p>
        <w:r>
          <w:t>` + row[i] + `</w:t>
        </w:r>
      </w:p>
    </w:tc>`
		}

		// 如果单元格数不够，填充空单元格
		for i := cellCount; i < columnCount; i++ {
			tableXML += `
    <w:tc>
      <w:p>
        <w:r>
          <w:t></w:t>
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
}
