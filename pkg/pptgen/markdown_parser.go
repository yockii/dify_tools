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
	ImageURL    string // 如果有图片
	Level       int    // 标题级别
	ParentTitle string // 父标题
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
			// 引用 - 创建引用幻灯片
			if currentSlide != nil {
				slides = append(slides, *currentSlide)
			}
			currentSlide = &SlideContent{
				Title:   "",
				Content: []string{strings.TrimPrefix(line, "> ")},
				Layout:  LayoutQuote,
				Level:   0, // 特殊级别表示引用
			}
			isL3Content = false
		} else if strings.HasPrefix(line, "![") && strings.Contains(line, "](") {
			// 图片 - 创建图片幻灯片
			start := strings.Index(line, "](") + 2
			end := strings.LastIndex(line, ")")
			if start > 0 && end > start {
				imageURL := line[start:end]
				if currentSlide != nil {
					currentSlide.ImageURL = imageURL
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
		slides = append(slides, *currentSlide)
	}

	// Check if we have at least one quote slide
	hasQuote := false
	for _, slide := range slides {
		if slide.Layout == LayoutQuote {
			hasQuote = true
			break
		}
	}

	// Add a placeholder quote slide if none exists
	if !hasQuote {
		slides = append(slides, SlideContent{
			Title:   "",
			Content: []string{"This presentation was created with xhxjj-yockii"},
			Layout:  LayoutQuote,
		})
	}

	// 添加结束页
	slides = append(slides, SlideContent{
		Title:  "谢谢观看",
		Layout: LayoutThankYou,
	})

	return slides, nil
}
