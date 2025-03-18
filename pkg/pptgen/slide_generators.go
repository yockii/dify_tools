package pptgen

import "fmt"

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
		contentItems += fmt.Sprintf(`
                    <a:p>
                        <a:pPr lvl="0">
                            <a:buChar char="•"/>
                        </a:pPr>
                        <a:r>
                            <a:rPr lang="en-US" sz="2800"/>
                            <a:t>%s</a:t>
                        </a:r>
                    </a:p>`, item)
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
