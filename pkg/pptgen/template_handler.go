package pptgen

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/yockii/dify_tools/pkg/logger"
)

// 从模板创建PPTX
func (g *PPTGenerator) createFromTemplate(zipWriter *zip.Writer, slides []SlideContent, config TemplateConfig) error {
	// 打开模板文件
	templateFile, err := os.Open(config.TemplatePath)
	if err != nil {
		logger.Error("打开模板文件失败", logger.F("templatePath", config.TemplatePath), logger.F("error", err))
		return err
	}
	defer templateFile.Close()

	// 创建模板ZIP读取器
	templateZip, err := zip.NewReader(templateFile, getFileSize(templateFile))
	if err != nil {
		logger.Error("创建模板ZIP读取器失败", logger.F("error", err))
		return err
	}

	// 获取模板中已有的幻灯片数量和需要的幻灯片数量
	slideCountInTemplate := countSlidesInTemplate(templateZip)
	slidesNeeded := len(slides)

	// 先保存模板中所有文件的数据，稍后会按需修改
	templateFiles := make(map[string][]byte)

	// 从模板中读取所有文件
	for _, file := range templateZip.File {
		data, err := readZipFile(file)
		if err != nil {
			logger.Error("读取模板文件失败", logger.F("filename", file.Name), logger.F("error", err))
			continue
		}
		templateFiles[file.Name] = data
	}

	// 强制删除所有模板幻灯片，使用新的幻灯片
	for i := 1; i <= slideCountInTemplate; i++ {
		slidePath := fmt.Sprintf("ppt/slides/slide%d.xml", i)
		delete(templateFiles, slidePath)

		slideRelPath := fmt.Sprintf("ppt/slides/_rels/slide%d.xml.rels", i)
		delete(templateFiles, slideRelPath)
	}

	// 重写 [Content_Types].xml 文件
	contentTypes := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
    <Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>
    <Default Extension="xml" ContentType="application/xml"/>
    <Override PartName="/ppt/presentation.xml" ContentType="application/vnd.openxmlformats-officedocument.presentationml.presentation.main+xml"/>
    <Override PartName="/ppt/slideMasters/slideMaster1.xml" ContentType="application/vnd.openxmlformats-officedocument.presentationml.slideMaster+xml"/>
    <Override PartName="/ppt/slideLayouts/slideLayout1.xml" ContentType="application/vnd.openxmlformats-officedocument.presentationml.slideLayout+xml"/>
    <Override PartName="/ppt/theme/theme1.xml" ContentType="application/vnd.openxmlformats-officedocument.theme+xml"/>`

	// 为每个幻灯片添加内容类型
	for i := 1; i <= slidesNeeded; i++ {
		contentTypes += fmt.Sprintf(`
    <Override PartName="/ppt/slides/slide%d.xml" ContentType="application/vnd.openxmlformats-officedocument.presentationml.slide+xml"/>`, i)
	}

	contentTypes += `
</Types>`

	templateFiles["[Content_Types].xml"] = []byte(contentTypes)

	// 重写 presentation.xml 文件
	presentation := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<p:presentation xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships" xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main">
    <p:sldMasterIdLst>
        <p:sldMasterId id="2147483648" r:id="rId1"/>
    </p:sldMasterIdLst>
    <p:sldIdLst>`

	// 添加所有幻灯片引用
	for i := 0; i < slidesNeeded; i++ {
		// 每个幻灯片 ID 从 256 开始递增，关系 ID 从 rId2 开始递增
		presentation += fmt.Sprintf(`
        <p:sldId id="%d" r:id="rId%d"/>`, 256+i, 2+i)
	}

	presentation += `
    </p:sldIdLst>
    <p:sldSz cx="9144000" cy="6858000"/>
    <p:notesSz cx="6858000" cy="9144000"/>
</p:presentation>`

	templateFiles["ppt/presentation.xml"] = []byte(presentation)

	// 重写 presentation.xml.rels 文件
	presRels := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
    <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/slideMaster" Target="slideMasters/slideMaster1.xml"/>`

	// 添加所有幻灯片关系
	for i := 0; i < slidesNeeded; i++ {
		presRels += fmt.Sprintf(`
    <Relationship Id="rId%d" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/slide" Target="slides/slide%d.xml"/>`, 2+i, i+1)
	}

	presRels += `
</Relationships>`

	templateFiles["ppt/_rels/presentation.xml.rels"] = []byte(presRels)

	// 生成所有新幻灯片内容
	for i, slide := range slides {
		slideNum := i + 1
		slidePath := fmt.Sprintf("ppt/slides/slide%d.xml", slideNum)
		slideRelPath := fmt.Sprintf("ppt/slides/_rels/slide%d.xml.rels", slideNum)

		// 生成幻灯片内容XML
		slideContent := g.generateSlideXML(slide, slideNum)
		templateFiles[slidePath] = []byte(slideContent)

		// 生成幻灯片关系XML
		slideRelContent := g.generateSlideRelXML(slide)
		templateFiles[slideRelPath] = []byte(slideRelContent)
	}

	// 将所有文件写入ZIP
	for filename, content := range templateFiles {
		fw, err := zipWriter.Create(filename)
		if err != nil {
			logger.Error("创建文件失败", logger.F("filename", filename), logger.F("error", err))
			return err
		}

		_, err = fw.Write(content)
		if err != nil {
			logger.Error("写入文件内容失败", logger.F("filename", filename), logger.F("error", err))
			return err
		}
	}

	return nil
}

// 更新内容类型文件，添加额外的幻灯片条目
func (g *PPTGenerator) updateContentTypes(files map[string][]byte, templateSlideCount, neededSlideCount int) error {
	if contentTypesData, exists := files["[Content_Types].xml"]; exists {
		contentTypes := string(contentTypesData)

		// 如果需要添加额外的幻灯片
		if neededSlideCount > templateSlideCount {
			lastEntryPos := strings.LastIndex(contentTypes, "</Types>")
			if lastEntryPos > 0 {
				extraSlideEntries := ""
				for i := templateSlideCount + 1; i <= neededSlideCount; i++ {
					extraSlideEntries += fmt.Sprintf(`
    <Override PartName="/ppt/slides/slide%d.xml" ContentType="application/vnd.openxmlformats-officedocument.presentationml.slide+xml"/>`, i)
				}

				// 插入额外的内容类型
				newContentTypes := contentTypes[:lastEntryPos] + extraSlideEntries + contentTypes[lastEntryPos:]
				files["[Content_Types].xml"] = []byte(newContentTypes)
			}
		} else if neededSlideCount < templateSlideCount {
			// 如果需要删除多余的幻灯片条目
			for i := neededSlideCount + 1; i <= templateSlideCount; i++ {
				pattern := fmt.Sprintf(`\s*<Override PartName="/ppt/slides/slide%d.xml"[^>]*/>`, i)
				re := regexp.MustCompile(pattern)
				contentTypes = re.ReplaceAllString(contentTypes, "")
			}
			files["[Content_Types].xml"] = []byte(contentTypes)
		}
	}

	return nil
}

// 更新演示文稿文件，包含所有需要的幻灯片
func (g *PPTGenerator) updatePresentation(files map[string][]byte, slidesNeeded int) error {
	if presentationData, exists := files["ppt/presentation.xml"]; exists {
		presentation := string(presentationData)

		sldIdListStartPos := strings.Index(presentation, "<p:sldIdLst>")
		sldIdListEndPos := strings.Index(presentation, "</p:sldIdLst>")

		if sldIdListStartPos > 0 && sldIdListEndPos > sldIdListStartPos {
			// 创建幻灯片ID列表
			newSldIdList := "<p:sldIdLst>"
			baseRId := g.findBaseRIdForSlides(files)

			for i := 0; i < slidesNeeded; i++ {
				slideId := 256 + i
				rId := baseRId + i
				newSldIdList += fmt.Sprintf(`<p:sldId id="%d" r:id="rId%d"/>`, slideId, rId)
			}
			newSldIdList += "</p:sldIdLst>"

			// 替换幻灯片ID列表
			newPresentation := presentation[:sldIdListStartPos] + newSldIdList + presentation[sldIdListEndPos+12:]
			files["ppt/presentation.xml"] = []byte(newPresentation)
		}
	}

	return nil
}

// 更新演示文稿关系文件
func (g *PPTGenerator) updatePresentationRels(files map[string][]byte, templateSlideCount, neededSlideCount int) error {
	if presRelsData, exists := files["ppt/_rels/presentation.xml.rels"]; exists {
		presRels := string(presRelsData)

		// 找到基础关系ID
		baseRId := g.findBaseRIdForSlides(files)

		// 找到关系列表结束位置
		lastRelPos := strings.LastIndex(presRels, "</Relationships>")

		if lastRelPos > 0 {
			// 如果需要添加额外的关系
			if neededSlideCount > templateSlideCount {
				extraRels := ""
				for i := templateSlideCount; i < neededSlideCount; i++ {
					rId := baseRId + i
					slideNum := i + 1
					extraRels += fmt.Sprintf(`
    <Relationship Id="rId%d" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/slide" Target="slides/slide%d.xml"/>`, rId, slideNum)
				}

				// 插入额外的关系
				newPresRels := presRels[:lastRelPos] + extraRels + presRels[lastRelPos:]
				files["ppt/_rels/presentation.xml.rels"] = []byte(newPresRels)
			} else if neededSlideCount < templateSlideCount {
				// 如果需要删除多余的关系
				for i := neededSlideCount; i < templateSlideCount; i++ {
					rId := baseRId + i
					pattern := fmt.Sprintf(`\s*<Relationship Id="rId%d"[^>]*/>`, rId)
					re := regexp.MustCompile(pattern)
					presRels = re.ReplaceAllString(presRels, "")
				}
				files["ppt/_rels/presentation.xml.rels"] = []byte(presRels)
			}
		}
	}

	return nil
}

// 查找幻灯片关系的基础ID
func (g *PPTGenerator) findBaseRIdForSlides(files map[string][]byte) int {
	// 默认假设从rId2开始
	baseRId := 2

	// 尝试从presentation.xml.rels中查找
	if presRelsData, exists := files["ppt/_rels/presentation.xml.rels"]; exists {
		presRels := string(presRelsData)
		// 查找幻灯片关系
		slideRelPattern := regexp.MustCompile(`<Relationship Id="rId(\d+)" Type="[^"]*?/slide"`)
		matches := slideRelPattern.FindAllStringSubmatch(presRels, -1)

		if len(matches) > 0 {
			firstMatch := matches[0]
			if len(firstMatch) > 1 {
				if rId, err := strconv.Atoi(firstMatch[1]); err == nil {
					baseRId = rId
				}
			}
		}
	}

	return baseRId
}

// 获取文件大小
func getFileSize(file *os.File) int64 {
	info, err := file.Stat()
	if err != nil {
		return 0
	}
	return info.Size()
}

// 从ZIP文件中读取内容
func readZipFile(file *zip.File) ([]byte, error) {
	rc, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer rc.Close()

	return io.ReadAll(rc)
}

func countSlidesInTemplate(templateZip *zip.Reader) int {
	slidePattern := regexp.MustCompile(`ppt/slides/slide\d+\.xml$`)
	count := 0

	for _, file := range templateZip.File {
		if slidePattern.MatchString(file.Name) {
			count++
		}
	}

	return count
}

// 添加其他可能需要的辅助函数来处理模板文件
func getSlideIndexFromFilename(filename string) int {
	base := filepath.Base(filename)
	parts := strings.Split(base, "slide")
	if len(parts) != 2 {
		return -1
	}

	numStr := strings.TrimSuffix(parts[1], ".xml")
	num, err := strconv.Atoi(numStr)
	if err != nil {
		return -1
	}

	return num
}

// 应用模板样式
func (g *PPTGenerator) applyTemplate(ppt *zip.Writer, config TemplateConfig) error {
	// 这是一个占位实现，实际应用中可以根据config参数设置不同的样式
	// 例如根据ThemeColor更改主题颜色，根据FontFamily更改字体等
	return nil
}
