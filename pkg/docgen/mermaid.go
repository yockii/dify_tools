package docgen

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// MermaidRenderer Mermaid图表渲染器
type MermaidRenderer struct {
	client     *http.Client
	serviceURL string // Mermaid渲染服务URL
}

// NewMermaidRenderer 创建一个新的Mermaid渲染器
func NewMermaidRenderer() *MermaidRenderer {
	return &MermaidRenderer{
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		serviceURL: "https://mermaid.ink/img/", // 默认使用公共服务
	}
}

// SetServiceURL 设置Mermaid渲染服务URL
func (r *MermaidRenderer) SetServiceURL(url string) {
	r.serviceURL = url
}

// RenderMermaid 渲染Mermaid代码为图片
func (r *MermaidRenderer) RenderMermaid(code string) ([]byte, error) {
	// 预处理：检测并添加必要的mermaid语法
	code = r.preProcessMermaidCode(code)

	// 使用两种方式尝试渲染：先尝试直接API渲染，失败则尝试下载图片
	imgBytes, err := r.renderViaMermaidAPI(code)
	if err == nil {
		return imgBytes, nil
	}

	// 如果API渲染失败，尝试通过mermaid.ink服务
	return r.renderViaMermaidInk(code)
}

// preProcessMermaidCode 预处理mermaid代码，添加必要的语法或转换
func (r *MermaidRenderer) preProcessMermaidCode(code string) string {
	// 如果是xychart-beta，确保其格式正确
	lines := strings.Split(code, "\n")
	if len(lines) > 0 && strings.TrimSpace(lines[0]) == "xychart-beta" {
		// 转换为标准的mermaid语法
		return "xychart" + code[len("xychart-beta"):]
	}

	// 如果代码中包含"pie title"但不是以pie开头，添加pie前缀
	if !strings.HasPrefix(strings.TrimSpace(code), "pie") &&
		strings.Contains(code, "pie title") {
		return "pie\n" + code
	}

	return code
}

// 通过Mermaid API渲染（如果有可用的本地或自定义API）
func (r *MermaidRenderer) renderViaMermaidAPI(code string) ([]byte, error) {
	payload := map[string]string{
		"code": code,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	// 这里假设有一个API端点可以渲染mermaid
	// 如果没有这样的API，这个函数会返回错误
	resp, err := r.client.Post(r.serviceURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("mermaid API返回错误状态码: %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}

// 通过mermaid.ink服务渲染
func (r *MermaidRenderer) renderViaMermaidInk(code string) ([]byte, error) {
	// 对代码进行特殊处理
	code = r.sanitizeMermaidCode(code)

	// 对代码进行base64编码
	encoded := base64.URLEncoding.EncodeToString([]byte(code))

	// 构建mermaid.ink URL
	url := "https://mermaid.ink/img/" + encoded + "?bgColor=white"

	// 获取图片
	resp, err := r.client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("无法从mermaid.ink获取图片: %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}

// sanitizeMermaidCode 清理并格式化mermaid代码
func (r *MermaidRenderer) sanitizeMermaidCode(code string) string {
	// 确保编码前的简单清理，处理特殊字符等
	return strings.ReplaceAll(code, "\r\n", "\n")
}

// RenderSvg 渲染Mermaid代码为SVG（可选功能）
func (r *MermaidRenderer) RenderSvg(code string) (string, error) {
	// 对代码进行base64编码
	encoded := base64.URLEncoding.EncodeToString([]byte(code))

	// 构建mermaid.ink SVG URL
	url := "https://mermaid.ink/svg/" + encoded

	// 获取SVG
	resp, err := r.client.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("无法从mermaid.ink获取SVG: %d", resp.StatusCode)
	}

	svgBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(svgBytes), nil
}
