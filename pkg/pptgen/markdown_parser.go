package pptgen

import (
	"strings"
)

// SlideContent 表示单个幻灯片内容
type SlideContent struct {
	Title       string
	Subtitle    string // Added for level 3 headings
	Content     []string
	Layout      SlideLayout
	ImageURL    string   // 如果有图片
	QuoteText   string   // 添加引用文本字段
	Level       int      // 标题级别
	ParentTitle string   // 父标题
	UseColumns  bool     // 是否使用两栏布局
	LeftColumn  []string // 左栏内容
	RightColumn []string // 右栏内容
}

// 解析Markdown大纲为幻灯片内容
func (g *PPTGenerator) parseMarkdownOutline(markdownText string) ([]SlideContent, error) {
	// 基础实现：将Markdown按标题分割成幻灯片
	var slides []SlideContent

	lines := strings.Split(markdownText, "\n")
	var currentSlide *SlideContent
	var currentL2Title string // 记录当前二级标题
	var isL3Content bool      // 标记是否在处理三级标题的内容

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		if strings.HasPrefix(line, "# ") {
			// 一级标题 - 创建标题幻灯片
			if currentSlide != nil {
				// 处理已收集的幻灯片内容
				finalizeSlide(currentSlide)
				slides = append(slides, *currentSlide)
			}
			currentSlide = &SlideContent{
				Title:  strings.TrimPrefix(line, "# "),
				Layout: LayoutTitle,
				Level:  1,
			}
			isL3Content = false
		} else if strings.HasPrefix(line, "### ") {
			// 三级标题 - 创建子内容幻灯片
			if currentSlide != nil {
				// 处理已收集的幻灯片内容
				finalizeSlide(currentSlide)
				slides = append(slides, *currentSlide)
			}
			currentSlide = &SlideContent{
				Title:       currentL2Title,                   // 保留父标题
				Subtitle:    strings.TrimPrefix(line, "### "), // 三级标题作为副标题
				Layout:      LayoutSubsection,                 // 需要新增一种布局类型
				Level:       3,
				ParentTitle: currentL2Title,
			}
			isL3Content = true
		} else if strings.HasPrefix(line, "## ") {
			// 二级标题 - 创建内容幻灯片
			if currentSlide != nil {
				// 处理已收集的幻灯片内容
				finalizeSlide(currentSlide)
				slides = append(slides, *currentSlide)
			}
			currentL2Title = strings.TrimPrefix(line, "## ") // 记录当前二级标题
			currentSlide = &SlideContent{
				Title:  currentL2Title,
				Layout: LayoutContent,
				Level:  2,
			}
			isL3Content = false
		} else if strings.HasPrefix(line, "- ") || strings.HasPrefix(line, "* ") {
			// 列表项 - 添加到当前幻灯片内容
			if currentSlide != nil {
				content := strings.TrimPrefix(line, "- ")
				content = strings.TrimPrefix(content, "* ")

				// 根据内容层级设置不同的前缀或格式
				if isL3Content {
					// 为三级标题下的内容添加特殊标记，以便在生成XML时能区别对待
					currentSlide.Content = append(currentSlide.Content, "L3:"+content)
				} else {
					currentSlide.Content = append(currentSlide.Content, content)
				}
			}
		} else if strings.HasPrefix(line, "> ") {
			// 引用 - 添加到当前幻灯片的引用部分，而不是创建新幻灯片
			quoteText := strings.TrimPrefix(line, "> ")
			if currentSlide != nil {
				currentSlide.QuoteText = quoteText

				// 检测是否有足够的内容来使用两栏布局
				if len(currentSlide.Content) > 0 || currentSlide.ImageURL != "" {
					currentSlide.UseColumns = true
				}
			}
		} else if strings.HasPrefix(line, "![") && strings.Contains(line, "](") {
			// 图片 - 添加到当前幻灯片，而不是创建新幻灯片
			start := strings.Index(line, "](") + 2
			end := strings.LastIndex(line, ")")
			if start > 0 && end > start {
				imageURL := line[start:end]
				if currentSlide != nil {
					currentSlide.ImageURL = imageURL

					// 检测是否有足够的内容来使用两栏布局
					if len(currentSlide.Content) > 0 || currentSlide.QuoteText != "" {
						currentSlide.UseColumns = true
					}
				}
			}
		} else {
			// 其他内容 - 添加到当前幻灯片
			if currentSlide != nil {
				// 根据内容层级设置不同的前缀或格式
				if isL3Content {
					// 为三级标题下的内容添加特殊标记
					currentSlide.Content = append(currentSlide.Content, "L3:"+line)
				} else {
					currentSlide.Content = append(currentSlide.Content, line)
				}
			}
		}
	}

	// 添加最后一个幻灯片
	if currentSlide != nil {
		finalizeSlide(currentSlide)
		slides = append(slides, *currentSlide)
	}

	// 添加结束页
	slides = append(slides, SlideContent{
		Title:  "谢谢观看",
		Layout: LayoutThankYou,
	})

	return slides, nil
}

// 处理幻灯片的最终布局和内容分配
func finalizeSlide(slide *SlideContent) {
	// 如果有引用或图片，且有内容，则使用两栏布局
	if slide.UseColumns || ((slide.QuoteText != "" || slide.ImageURL != "") && len(slide.Content) > 0) {
		slide.Layout = LayoutTwoColumn

		// 为两栏布局准备内容
		// 引用和图片放在左栏，内容放在右栏
		if slide.QuoteText != "" {
			slide.LeftColumn = append(slide.LeftColumn, "QUOTE:"+slide.QuoteText)
		}

		// 如果有图片，也添加到左栏
		if slide.ImageURL != "" {
			slide.LeftColumn = append(slide.LeftColumn, "IMAGE:"+slide.ImageURL)
		}

		// 内容放在右栏
		slide.RightColumn = append(slide.RightColumn, slide.Content...)

		// 清空常规内容，因为已经分配到两栏了
		slide.Content = nil
	}
}
