package pptgen

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/yockii/dify_tools/pkg/logger"
)

// DumpSlideStructure outputs the slide structure to a file for debugging
func (g *PPTGenerator) DumpSlideStructure(slides []SlideContent, filePath string) error {
	// 确保输出目录存在
	dir := filePath[:strings.LastIndex(filePath, "/")]
	if dir != "" {
		if err := os.MkdirAll(dir, 0755); err != nil {
			// 如果无法创建目录，只打印错误但不中断流程
			fmt.Printf("无法创建目录 %s: %v\n", dir, err)
		}
	}

	// 将幻灯片结构序列化为JSON
	data, err := json.MarshalIndent(slides, "", "  ")
	if err != nil {
		fmt.Printf("序列化幻灯片数据失败: %v\n", err)
		return err
	}

	// 写入文件
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		fmt.Printf("写入调试文件失败: %v\n", err)
		return err
	}

	// 安全地使用logger，避免nil panic
	logSlides := func() {
		defer func() {
			if r := recover(); r != nil {
				fmt.Printf("日志记录时发生恢复性错误: %v\n", r)
			}
		}()

		// 尝试记录摘要
		logger.Info("Slide summary", logger.F("slideCount", len(slides)))
		for i, slide := range slides {
			logger.Info("Slide",
				logger.F("index", i),
				logger.F("layout", slide.Layout),
				logger.F("title", slide.Title),
				logger.F("subtitle", slide.Subtitle),
				logger.F("contentCount", len(slide.Content)))
		}
	}

	// 在安全的环境中执行日志记录
	logSlides()

	return nil
}

// LogSlideGeneration logs information during the generation of each slide
func (g *PPTGenerator) LogSlideGeneration(index int, slide SlideContent) {
	// 使用闭包和恢复机制避免logger未初始化导致的panic
	func() {
		defer func() {
			if r := recover(); r != nil {
				fmt.Printf("幻灯片生成日志记录过程中发生恢复性错误: %v\n", r)
			}
		}()

		contentPreview := ""
		l3ContentCount := 0

		if len(slide.Content) > 0 {
			if len(slide.Content[0]) > 30 {
				contentPreview = slide.Content[0][:30] + "..."
			} else if len(slide.Content) > 0 {
				contentPreview = slide.Content[0]
			}

			// Count L3 content items
			for _, item := range slide.Content {
				if strings.HasPrefix(item, "L3:") {
					l3ContentCount++
				}
			}
		}

		logger.Info("Generating slide",
			logger.F("index", index),
			logger.F("layout", slide.Layout),
			logger.F("level", slide.Level),
			logger.F("title", slide.Title),
			logger.F("subtitle", slide.Subtitle),
			logger.F("contentPreview", contentPreview),
			logger.F("contentCount", len(slide.Content)),
			logger.F("l3ContentCount", l3ContentCount))
	}()
}
