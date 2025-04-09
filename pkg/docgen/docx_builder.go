package docgen

import (
	"archive/zip"
	"bytes"
	"fmt"
	"path/filepath"
	"strings"
)

// DocxBuilder 负责构建DOCX文档
type DocxBuilder struct {
	elementHandler *WordElementHandler
}

// NewDocxBuilder 创建一个新的DOCX构建器
func NewDocxBuilder(elementHandler *WordElementHandler) *DocxBuilder {
	return &DocxBuilder{
		elementHandler: elementHandler,
	}
}

// BuildDocx 构建DOCX文档
func (b *DocxBuilder) BuildDocx(htmlContent string, mermaidImages map[string][]byte) ([]byte, error) {
	// 创建一个用于存储输出的缓冲区
	outputBuffer := new(bytes.Buffer)
	zipWriter := zip.NewWriter(outputBuffer)

	// 预处理HTML内容
	htmlContent = RemoveHtmlAttributes(htmlContent)

	// 清理<!-- raw HTML omitted -->标记
	htmlContent = strings.ReplaceAll(htmlContent, "<!-- raw HTML omitted -->", "")

	// 将HTML转换为Word XML
	documentXML := b.elementHandler.ConvertHtmlToWordXml(htmlContent)

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
		documentXML = strings.Replace(
			documentXML,
			fmt.Sprintf(`<img:RelId>%s</img:RelId>`, imageID),
			fmt.Sprintf(`<img:RelId>%s</img:RelId>`, relID),
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
	wordRelsXML := getWordRelsXML() + mediaRels + "</Relationships>"

	// 在内存中写入所有XML文件到ZIP
	files := map[string]string{
		"[Content_Types].xml":          getContentTypesXML(),
		"_rels/.rels":                  getRelsXML(),
		"word/_rels/document.xml.rels": wordRelsXML,
		"word/styles.xml":              getStylesXML(),
		"word/numbering.xml":           getNumberingXML(),
		"word/document.xml":            documentXML,
	}

	for path, content := range files {
		// 确保目录存在于ZIP中
		dirPath := filepath.Dir(path)
		if dirPath != "." && !strings.Contains(path, "/") {
			_, err := zipWriter.Create(dirPath + "/")
			if err != nil {
				return nil, err
			}
		}

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

// XML模板函数
func getContentTypesXML() string {
	return `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
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
}

func getRelsXML() string {
	return `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="word/document.xml"/>
</Relationships>`
}

func getWordRelsXML() string {
	return `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/styles" Target="styles.xml"/>
  <Relationship Id="rId2" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/numbering" Target="numbering.xml"/>
`
}

func getStylesXML() string {
	return `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
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
}

func getNumberingXML() string {
	return `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
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
}
