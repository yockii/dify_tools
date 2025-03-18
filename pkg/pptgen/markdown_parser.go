package pptgen

import (
	"strings"
)

// SlideContent 表示单个幻灯片内容
type SlideContent struct {
	Title    string
	Content  []string
	Layout   SlideLayout
	ImageURL string // 如果有图片
}

// 解析Markdown大纲为幻灯片内容
func (g *PPTGenerator) parseMarkdownOutline(markdownText string) ([]SlideContent, error) {
	// 基础实现：将Markdown按标题分割成幻灯片
	var slides []SlideContent

	lines := strings.Split(markdownText, "\n")
	var currentSlide *SlideContent

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
			}
		} else if strings.HasPrefix(line, "## ") {
			// 二级标题 - 创建内容幻灯片
			if currentSlide != nil {
				slides = append(slides, *currentSlide)
			}
			currentSlide = &SlideContent{
				Title:  strings.TrimPrefix(line, "## "),
				Layout: LayoutContent,
			}
		} else if strings.HasPrefix(line, "- ") || strings.HasPrefix(line, "* ") {
			// 列表项 - 添加到当前幻灯片内容
			if currentSlide != nil {
				content := strings.TrimPrefix(line, "- ")
				content = strings.TrimPrefix(content, "* ")
				currentSlide.Content = append(currentSlide.Content, content)
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
			}
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
				currentSlide.Content = append(currentSlide.Content, line)
			}
		}
	}

	// 添加最后一个幻灯片
	if currentSlide != nil {
		slides = append(slides, *currentSlide)
	}

	return slides, nil
}
