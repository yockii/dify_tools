package pptgen

import (
	"archive/zip"
	"bytes"
	"fmt"
	"os"

	"github.com/yockii/dify_tools/pkg/logger"
	"github.com/yuin/goldmark"
)

// TemplateType 表示PPT模板类型
type TemplateType string

const (
	TemplateBusiness   TemplateType = "business"   // 商务模板
	TemplateAcademic   TemplateType = "academic"   // 学术模板
	TemplateMinimalist TemplateType = "minimalist" // 极简模板
)

// SlideLayout 表示幻灯片布局类型
type SlideLayout int

const (
	LayoutTitle      SlideLayout = iota // 标题幻灯片
	LayoutContent                       // 内容幻灯片
	LayoutTwoColumn                     // 两栏布局
	LayoutImage                         // 图片布局
	LayoutQuote                         // 引用布局
	LayoutThankYou                      // 结束页
	LayoutSubsection                    // 子内容布局，用于三级标题
)

// TemplateConfig 表示模板配置
type TemplateConfig struct {
	Type         TemplateType      // 模板类型
	TemplatePath string            // 模板文件路径，如果使用内置模板则为空
	ThemeColor   string            // 主题色
	FontFamily   string            // 字体系列
	CustomProps  map[string]string // 自定义属性
}

// PPTGenerator 处理PPT生成
type PPTGenerator struct {
	templates map[TemplateType]string // 模板路径映射
	markdown  goldmark.Markdown       // Markdown解析器
}

// NewPPTGenerator 创建一个新的PPT生成器
func NewPPTGenerator() *PPTGenerator {
	return &PPTGenerator{
		templates: map[TemplateType]string{
			TemplateBusiness:   "./assets/templates/template.pptx",
			TemplateAcademic:   "./assets/templates/template.pptx",
			TemplateMinimalist: "./assets/templates/template.pptx",
		},
		markdown: goldmark.New(
			goldmark.WithExtensions( /* 可添加所需扩展 */ ),
		),
	}
}

// RegisterTemplate 注册自定义模板
func (g *PPTGenerator) RegisterTemplate(templateType TemplateType, path string) {
	g.templates[templateType] = path
}

// GeneratePPTX 根据Markdown大纲生成PPTX
func (g *PPTGenerator) GeneratePPTX(config TemplateConfig, markdownOutline string) ([]byte, error) {
	// 解析Markdown大纲
	slides, err := g.parseMarkdownOutline(markdownOutline)
	if err != nil {
		return nil, err
	}

	// 创建一个内存缓冲区用于保存ZIP文件
	buf := new(bytes.Buffer)
	zipWriter := zip.NewWriter(buf)

	// 添加必要的PPTX文件
	if err := g.addPresentationFiles(zipWriter, slides, config); err != nil {
		return nil, err
	}

	// 关闭ZIP写入器
	if err := zipWriter.Close(); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// 添加演示文稿文件到ZIP
func (g *PPTGenerator) addPresentationFiles(zipWriter *zip.Writer, slides []SlideContent, config TemplateConfig) error {
	// 如果提供了模板路径，则基于模板创建
	if config.TemplatePath != "" {
		return g.createFromTemplate(zipWriter, slides, config)
	}

	if templatePath, exists := g.templates[config.Type]; exists {
		templateConfig := config
		templateConfig.TemplatePath = templatePath
		return g.createFromTemplate(zipWriter, slides, templateConfig)
	}

	// 否则使用基本模板
	if err := g.addContentTypes(zipWriter, slides); err != nil {
		return err
	}

	if err := g.addRels(zipWriter); err != nil {
		return err
	}

	if err := g.addPresentation(zipWriter, slides, config); err != nil {
		return err
	}

	if err := g.addSlides(zipWriter, slides); err != nil {
		return err
	}

	// 添加其他必要的文件
	return g.addMiscFiles(zipWriter, config)
}

// WriteToFile 将PPTX写入文件
func (g *PPTGenerator) WriteToFile(pptxBytes []byte, filePath string) error {
	// 将生成的PPTX字节写入文件
	file, err := os.Create(filePath)
	if err != nil {
		logger.Error("创建PPTX文件失败", logger.F("filePath", filePath), logger.F("error", err))
		return err
	}
	defer file.Close()

	_, err = file.Write(pptxBytes)
	if err != nil {
		logger.Error("写入PPTX文件失败", logger.F("filePath", filePath), logger.F("error", err))
		return err
	}

	return nil
}

// addSlides adds all slide files to the PPTX structure
// This method handles the generation of slide XML files and their relationships
func (g *PPTGenerator) addSlides(zipWriter *zip.Writer, slides []SlideContent) error {
	// 将调试功能设为可选，避免因日志错误中断主要功能
	debugEnabled := false // 可以通过配置参数控制

	if debugEnabled {
		// 尝试输出调试信息，但忽略任何错误
		_ = g.DumpSlideStructure(slides, "./debug_slides.json")
	}

	for i, slide := range slides {
		// 调试日志也设为可选
		if debugEnabled {
			g.LogSlideGeneration(i, slide)
		}

		slideNum := i + 1
		slidePath := fmt.Sprintf("ppt/slides/slide%d.xml", slideNum)

		slideContent := g.generateSlideXML(slide, slideNum)

		slideWriter, err := zipWriter.Create(slidePath)
		if err != nil {
			// 使用fmt.Printf作为后备，避免依赖logger
			fmt.Printf("创建幻灯片XML失败: %s, 错误: %v\n", slidePath, err)
			return err
		}

		_, err = slideWriter.Write([]byte(slideContent))
		if err != nil {
			logger.Error("写入幻灯片内容失败", logger.F("slidePath", slidePath), logger.F("error", err))
			return err
		}

		// Add slide relationship file
		slideRelPath := fmt.Sprintf("ppt/slides/_rels/slide%d.xml.rels", slideNum)
		slideRelContent := g.generateSlideRelXML(slide)

		slideRelWriter, err := zipWriter.Create(slideRelPath)
		if err != nil {
			logger.Error("创建幻灯片关系XML失败", logger.F("slideRelPath", slideRelPath), logger.F("error", err))
			return err
		}

		_, err = slideRelWriter.Write([]byte(slideRelContent))
		if err != nil {
			logger.Error("写入幻灯片关系内容失败", logger.F("slideRelPath", slideRelPath), logger.F("error", err))
			return err
		}
	}

	return nil
}

// generateSlideRelXML generates the relationship XML for a slide
func (g *PPTGenerator) generateSlideRelXML(slide SlideContent) string {
	return `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
	<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/slideLayout" Target="../slideLayouts/slideLayout1.xml"/>
</Relationships>`
}
