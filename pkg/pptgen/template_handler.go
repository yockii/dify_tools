package pptgen

import (
	"archive/zip"
	"fmt"
	"io"
	"strings"
)

// 从模板创建
func (g *PPTGenerator) createFromTemplate(zipWriter *zip.Writer, slides []SlideContent, config TemplateConfig) error {
	templatePath := config.TemplatePath
	if templatePath == "" {
		if path, ok := g.templates[config.Type]; ok {
			templatePath = path
		} else {
			return fmt.Errorf("template not found for type: %s", config.Type)
		}
	}

	// 打开模板文件
	templateReader, err := zip.OpenReader(templatePath)
	if err != nil {
		return err
	}
	defer templateReader.Close()

	// 复制模板文件并修改必要的部分
	for _, file := range templateReader.File {
		// 跳过幻灯片文件，因为我们会创建新的
		if strings.HasPrefix(file.Name, "ppt/slides/slide") && strings.HasSuffix(file.Name, ".xml") {
			continue
		}

		// 读取文件内容
		rc, err := file.Open()
		if err != nil {
			return err
		}

		content, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			return err
		}

		// 创建新文件
		fw, err := zipWriter.Create(file.Name)
		if err != nil {
			return err
		}

		// 写入内容
		_, err = fw.Write(content)
		if err != nil {
			return err
		}
	}

	// 添加幻灯片
	return g.addSlides(zipWriter, slides)
}

// 应用模板样式
func (g *PPTGenerator) applyTemplate(ppt *zip.Writer, config TemplateConfig) error {
	// 这是一个占位实现，实际应用中可以根据config参数设置不同的样式
	// 例如根据ThemeColor更改主题颜色，根据FontFamily更改字体等
	return nil
}
