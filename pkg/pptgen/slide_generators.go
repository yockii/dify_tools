package pptgen

import (
	"fmt"
	"strings"
)

// 生成幻灯片XML内容
func (g *PPTGenerator) generateSlideXML(slide SlideContent, slideIndex int) string {
	var contentXML string

	switch slide.Layout {
	case LayoutTitle:
		contentXML = g.generateTitleSlideXML(slide)
	case LayoutContent:
		contentXML = g.generateContentSlideXML(slide)
	case LayoutQuote:
		contentXML = g.generateQuoteSlideXML(slide)
	case LayoutThankYou:
		contentXML = g.generateThankYouSlideXML(slide)
	case LayoutSubsection:
		contentXML = g.generateSubsectionSlideXML(slide)
	case LayoutTwoColumn: // 添加两栏布局处理
		contentXML = g.generateTwoColumnSlideXML(slide)
	default:
		contentXML = g.generateContentSlideXML(slide)
	}

	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<p:sld xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships" xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main">
    <p:cSld>
        <p:spTree>
            <p:nvGrpSpPr>
                <p:cNvPr id="1" name=""/>
                <p:cNvGrpSpPr/>
                <p:nvPr/>
            </p:nvGrpSpPr>
            <p:grpSpPr>
                <a:xfrm>
                    <a:off x="0" y="0"/>
                    <a:ext cx="0" cy="0"/>
                    <a:chOff x="0" y="0"/>
                    <a:chExt cx="0" cy="0"/>
                </a:xfrm>
            </p:grpSpPr>
            %s
        </p:spTree>
    </p:cSld>
    <p:clrMapOvr>
        <a:masterClrMapping/>
    </p:clrMapOvr>
</p:sld>`, contentXML)
}

// 生成标题幻灯片内容
func (g *PPTGenerator) generateTitleSlideXML(slide SlideContent) string {
	return fmt.Sprintf(`
            <p:sp>
                <p:nvSpPr>
                    <p:cNvPr id="2" name="Title"/>
                    <p:cNvSpPr>
                        <a:spLocks noGrp="1"/>
                    </p:cNvSpPr>
                    <p:nvPr>
                        <p:ph type="title"/>
                    </p:nvPr>
                </p:nvSpPr>
                <p:spPr>
                    <a:xfrm>
                        <a:off x="1474200" y="1296000"/>
                        <a:ext cx="6096000" cy="1296000"/>
                    </a:xfrm>
                </p:spPr>
                <p:txBody>
                    <a:bodyPr/>
                    <a:lstStyle/>
                    <a:p>
                        <a:r>
                            <a:rPr lang="en-US" sz="4400" b="1"/>
                            <a:t>%s</a:t>
                        </a:r>
                    </a:p>
                </p:txBody>
            </p:sp>`, slide.Title)
}

// 生成内容幻灯片内容
func (g *PPTGenerator) generateContentSlideXML(slide SlideContent) string {
	contentItems := ""
	for _, item := range slide.Content {
		// 也需要检查常规内容幻灯片中是否有Level 3内容（可能是解析错误或特殊情况）
		isL3Content := strings.HasPrefix(item, "L3:")
		itemText := item
		content := ""
		if isL3Content {
			itemText = strings.TrimPrefix(item, "L3:")
			content = ` i="1"`
		}

		contentItems += fmt.Sprintf(`
                    <a:p>
                        <a:pPr lvl="0">
                            <a:buChar char="•"/>
                        </a:pPr>
                        <a:r>
                            <a:rPr lang="en-US" sz="2800"%s/>
                            <a:t>%s</a:t>
                        </a:r>
                    </a:p>`, content, itemText)
	}

	return fmt.Sprintf(`
            <p:sp>
                <p:nvSpPr>
                    <p:cNvPr id="2" name="Title"/>
                    <p:cNvSpPr>
                        <a:spLocks noGrp="1"/>
                    </p:cNvSpPr>
                    <p:nvPr>
                        <p:ph type="title"/>
                    </p:nvPr>
                </p:nvSpPr>
                <p:spPr>
                    <a:xfrm>
                        <a:off x="1474200" y="457200"/>
                        <a:ext cx="6096000" cy="914400"/>
                    </a:xfrm>
                </p:spPr>
                <p:txBody>
                    <a:bodyPr/>
                    <a:lstStyle/>
                    <a:p>
                        <a:r>
                            <a:rPr lang="en-US" sz="3600" b="1"/>
                            <a:t>%s</a:t>
                        </a:r>
                    </a:p>
                </p:txBody>
            </p:sp>
            <p:sp>
                <p:nvSpPr>
                    <p:cNvPr id="3" name="Content"/>
                    <p:cNvSpPr>
                        <a:spLocks noGrp="1"/>
                    </p:cNvSpPr>
                    <p:nvPr>
                        <p:ph type="body" idx="1"/>
                    </p:nvPr>
                </p:nvSpPr>
                <p:spPr>
                    <a:xfrm>
                        <a:off x="1474200" y="1828800"/>
                        <a:ext cx="6096000" cy="3657600"/>
                    </a:xfrm>
                </p:spPr>
                <p:txBody>
                    <a:bodyPr/>
                    <a:lstStyle/>%s
                </p:txBody>
            </p:sp>`, slide.Title, contentItems)
}

// 生成引用幻灯片内容
func (g *PPTGenerator) generateQuoteSlideXML(slide SlideContent) string {
	quoteText := ""
	if len(slide.Content) > 0 {
		quoteText = slide.Content[0]
	}

	return fmt.Sprintf(`
            <p:sp>
                <p:nvSpPr>
                    <p:cNvPr id="2" name="Quote"/>
                    <p:cNvSpPr>
                        <a:spLocks noGrp="1"/>
                    </p:cNvSpPr>
                    <p:nvPr>
                        <p:ph type="body"/>
                    </p:nvPr>
                </p:nvSpPr>
                <p:spPr>
                    <a:xfrm>
                        <a:off x="1474200" y="2743200"/>
                        <a:ext cx="6096000" cy="2743200"/>
                    </a:xfrm>
                </p:spPr>
                <p:txBody>
                    <a:bodyPr/>
                    <a:lstStyle/>
                    <a:p>
                        <a:r>
                            <a:rPr lang="en-US" sz="3600" i="1"/>
                            <a:t>%s</a:t>
                        </a:r>
                    </a:p>
                </p:txBody>
            </p:sp>`, quoteText)
}

// 生成感谢幻灯片内容
func (g *PPTGenerator) generateThankYouSlideXML(slide SlideContent) string {
	return fmt.Sprintf(`
            <p:sp>
                <p:nvSpPr>
                    <p:cNvPr id="2" name="Thank You"/>
                    <p:cNvSpPr>
                        <a:spLocks noGrp="1"/>
                    </p:cNvSpPr>
                    <p:nvPr>
                        <p:ph type="ctrTitle"/>
                    </p:nvPr>
                </p:nvSpPr>
                <p:spPr>
                    <a:xfrm>
                        <a:off x="1474200" y="2743200"/>
                        <a:ext cx="6096000" cy="2743200"/>
                    </a:xfrm>
                </p:spPr>
                <p:txBody>
                    <a:bodyPr/>
                    <a:lstStyle/>
                    <a:p>
                        <a:r>
                            <a:rPr lang="en-US" sz="3600" i="1"/>
                            <a:t>%s</a:t>
                        </a:r>
                    </a:p>
                </p:txBody>
            </p:sp>`, slide.Title)
}

// 生成子内容幻灯片内容
func (g *PPTGenerator) generateSubsectionSlideXML(slide SlideContent) string {
	contentItems := ""
	for _, item := range slide.Content {
		// 检查是否为Level 3内容（有特殊前缀）
		isL3Content := strings.HasPrefix(item, "L3:")
		itemText := item
		if isL3Content {
			itemText = strings.TrimPrefix(item, "L3:")
		}

		// 为Level 3内容应用不同的样式
		if isL3Content {
			contentItems += fmt.Sprintf(`
                    <a:p>
                        <a:pPr lvl="0">
                            <a:buChar char="◆"/>
                        </a:pPr>
                        <a:r>
                            <a:rPr lang="en-US" sz="2800" i="1"/>
                            <a:t>%s</a:t>
                        </a:r>
                    </a:p>`, itemText)
		} else {
			contentItems += fmt.Sprintf(`
                    <a:p>
                        <a:pPr lvl="0">
                            <a:buChar char="•"/>
                        </a:pPr>
                        <a:r>
                            <a:rPr lang="en-US" sz="2800"/>
                            <a:t>%s</a:t>
                        </a:r>
                    </a:p>`, itemText)
		}
	}

	return fmt.Sprintf(`
            <p:sp>
                <p:nvSpPr>
                    <p:cNvPr id="2" name="ParentTitle"/>
                    <p:cNvSpPr>
                        <a:spLocks noGrp="1"/>
                    </p:cNvSpPr>
                    <p:nvPr>
                        <p:ph type="title"/>
                    </p:nvPr>
                </p:nvSpPr>
                <p:spPr>
                    <a:xfrm>
                        <a:off x="1474200" y="457200"/>
                        <a:ext cx="6096000" cy="914400"/>
                    </a:xfrm>
                </p:spPr>
                <p:txBody>
                    <a:bodyPr/>
                    <a:lstStyle/>
                    <a:p>
                        <a:r>
                            <a:rPr lang="en-US" sz="3600" b="1"/>
                            <a:t>%s</a:t>
                        </a:r>
                    </a:p>
                </p:txBody>
            </p:sp>
            <p:sp>
                <p:nvSpPr>
                    <p:cNvPr id="3" name="Subtitle"/>
                    <p:cNvSpPr>
                        <a:spLocks noGrp="1"/>
                    </p:cNvSpPr>
                    <p:nvPr/>
                </p:nvSpPr>
                <p:spPr>
                    <a:xfrm>
                        <a:off x="1474200" y="1371600"/>
                        <a:ext cx="6096000" cy="457200"/>
                    </a:xfrm>
                </p:spPr>
                <p:txBody>
                    <a:bodyPr/>
                    <a:lstStyle/>
                    <a:p>
                        <a:r>
                            <a:rPr lang="en-US" sz="3200" b="1"/>
                            <a:t>%s</a:t>
                        </a:r>
                    </a:p>
                </p:txBody>
            </p:sp>
            <p:sp>
                <p:nvSpPr>
                    <p:cNvPr id="4" name="Content"/>
                    <p:cNvSpPr>
                        <a:spLocks noGrp="1"/>
                    </p:cNvSpPr>
                    <p:nvPr>
                        <p:ph type="body" idx="1"/>
                    </p:nvPr>
                </p:nvSpPr>
                <p:spPr>
                    <a:xfrm>
                        <a:off x="1474200" y="1828800"/>
                        <a:ext cx="6096000" cy="3657600"/>
                    </a:xfrm>
                </p:spPr>
                <p:txBody>
                    <a:bodyPr/>
                    <a:lstStyle/>%s
                </p:txBody>
            </p:sp>`, slide.Title, slide.Subtitle, contentItems)
}

// 生成两栏布局幻灯片内容
func (g *PPTGenerator) generateTwoColumnSlideXML(slide SlideContent) string {
	// 处理左栏内容
	leftColumnItems := ""
	for _, item := range slide.LeftColumn {
		if strings.HasPrefix(item, "QUOTE:") {
			// 处理引用
			quoteText := strings.TrimPrefix(item, "QUOTE:")
			leftColumnItems += fmt.Sprintf(`
                    <a:p>
                        <a:pPr algn="ctr"/>
                        <a:r>
                            <a:rPr lang="en-US" sz="2600" i="1"/>
                            <a:t>"%s"</a:t>
                        </a:r>
                    </a:p>`, quoteText)
		} else if strings.HasPrefix(item, "IMAGE:") {
			// 处理图片（此处仅为占位符，实际需要更复杂的图片处理逻辑）
			// 在实际实现中需要添加图片的引用和关系
			imageURL := strings.TrimPrefix(item, "IMAGE:")
			leftColumnItems += fmt.Sprintf(`
                    <a:p>
                        <a:pPr algn="ctr"/>
                        <a:r>
                            <a:rPr lang="en-US" sz="2400"/>
                            <a:t>[图片: %s]</a:t>
                        </a:r>
                    </a:p>`, imageURL)
		} else {
			// 处理普通文本
			leftColumnItems += fmt.Sprintf(`
                    <a:p>
                        <a:r>
                            <a:rPr lang="en-US" sz="2400"/>
                            <a:t>%s</a:t>
                        </a:r>
                    </a:p>`, item)
		}
	}

	// 处理右栏内容
	rightColumnItems := ""
	for _, item := range slide.RightColumn {
		// 检查是否为L3内容
		isL3Content := strings.HasPrefix(item, "L3:")
		itemText := item
		if isL3Content {
			itemText = strings.TrimPrefix(item, "L3:")
		}

		// 为L3内容应用不同的样式
		if isL3Content {
			rightColumnItems += fmt.Sprintf(`
                    <a:p>
                        <a:pPr lvl="0">
                            <a:buChar char="◆"/>
                        </a:pPr>
                        <a:r>
                            <a:rPr lang="en-US" sz="2400" i="1"/>
                            <a:t>%s</a:t>
                        </a:r>
                    </a:p>`, itemText)
		} else {
			rightColumnItems += fmt.Sprintf(`
                    <a:p>
                        <a:pPr lvl="0">
                            <a:buChar char="•"/>
                        </a:pPr>
                        <a:r>
                            <a:rPr lang="en-US" sz="2400"/>
                            <a:t>%s</a:t>
                        </a:r>
                    </a:p>`, itemText)
		}
	}

	return fmt.Sprintf(`
            <p:sp>
                <p:nvSpPr>
                    <p:cNvPr id="2" name="Title"/>
                    <p:cNvSpPr>
                        <a:spLocks noGrp="1"/>
                    </p:cNvSpPr>
                    <p:nvPr>
                        <p:ph type="title"/>
                    </p:nvPr>
                </p:nvSpPr>
                <p:spPr>
                    <a:xfrm>
                        <a:off x="1474200" y="457200"/>
                        <a:ext cx="6096000" cy="914400"/>
                    </a:xfrm>
                </p:spPr>
                <p:txBody>
                    <a:bodyPr/>
                    <a:lstStyle/>
                    <a:p>
                        <a:r>
                            <a:rPr lang="en-US" sz="3600" b="1"/>
                            <a:t>%s</a:t>
                        </a:r>
                    </a:p>
                </p:txBody>
            </p:sp>
            <p:sp>
                <p:nvSpPr>
                    <p:cNvPr id="3" name="Left Column"/>
                    <p:cNvSpPr>
                        <a:spLocks noGrp="1"/>
                    </p:cNvSpPr>
                    <p:nvPr/>
                </p:nvSpPr>
                <p:spPr>
                    <a:xfrm>
                        <a:off x="1474200" y="1828800"/>
                        <a:ext cx="2858575" cy="3657600"/>
                    </a:xfrm>
                </p:spPr>
                <p:txBody>
                    <a:bodyPr/>
                    <a:lstStyle/>%s
                </p:txBody>
            </p:sp>
            <p:sp>
                <p:nvSpPr>
                    <p:cNvPr id="4" name="Right Column"/>
                    <p:cNvSpPr>
                        <a:spLocks noGrp="1"/>
                    </p:cNvSpPr>
                    <p:nvPr/>
                </p:nvSpPr>
                <p:spPr>
                    <a:xfrm>
                        <a:off x="4711625" y="1828800"/>
                        <a:ext cx="2858575" cy="3657600"/>
                    </a:xfrm>
                </p:spPr>
                <p:txBody>
                    <a:bodyPr/>
                    <a:lstStyle/>%s
                </p:txBody>
            </p:sp>`, slide.Title, leftColumnItems, rightColumnItems)
}
