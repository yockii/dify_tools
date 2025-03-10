package dify

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/tidwall/gjson"
	"github.com/yockii/dify_tools/pkg/logger"
)

type ChatClient struct {
	baseUrl          string
	defaultAPISecret string
	httpClient       *http.Client
}

func NewChatClient(baseUrl, defaultAPISecret string) *ChatClient {
	httpClient := &http.Client{}

	return &ChatClient{
		baseUrl:          baseUrl,
		defaultAPISecret: defaultAPISecret,
		httpClient:       httpClient,
	}
}

type ChatFile struct {
	Type           string `json:"type"`            // document/image/audio/video/custom
	TransferMethod string `json:"transfer_method"` // remote_url/local_file
	URL            string `json:"url"`             // remote_url时必填
	UploadFileID   string `json:"upload_file_id"`  // local_file时必填
}

type ChatMessageRequest struct {
	Query            string                 `json:"query"`
	Inputs           map[string]interface{} `json:"inputs"`
	ResponseMode     string                 `json:"response_mode"` // streaming/blocking，根据不同的模式处理不同的响应方式
	User             string                 `json:"user"`
	ConversationID   string                 `json:"conversation_id"`
	Files            []ChatFile             `json:"files"`
	AutoGenerateName bool                   `json:"auto_generate_name"`
}

type Conversation struct {
	ID        string                 `json:"id"`
	Name      string                 `json:"name"`
	Inputs    map[string]interface{} `json:"inputs"`
	Status    string                 `json:"status"`
	CreatedAt string                 `json:"created_at"`
	UpdatedAt string                 `json:"updated_at"`
}

func (c *ChatClient) GetConversations(user, apiSecret string) ([]*Conversation, error) {
	req, err := http.NewRequest("GET", c.baseUrl+"/conversations", nil)
	if err != nil {
		logger.Error("创建请求失败", logger.F("err", err))
		return nil, err
	}
	if apiSecret != "" {
		req.Header.Set("Authorization", "Bearer "+apiSecret)
	} else if c.defaultAPISecret != "" {
		req.Header.Set("Authorization", "Bearer "+c.defaultAPISecret)
	} else {
		logger.Error("未提供API密钥")
		return nil, fmt.Errorf("未提供API密钥")
	}
	// ?user=xxx&last_id=&limit=
	q := req.URL.Query()
	q.Add("user", user)
	req.URL.RawQuery = q.Encode()

	resp, err := c.httpClient.Do(req)
	if err != nil {
		logger.Error("请求失败", logger.F("err", err))
		return nil, err
	}
	defer resp.Body.Close()
	response, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Error("读取响应失败", logger.F("err", err))
		return nil, err
	}
	j := gjson.ParseBytes(response)
	conversations := make([]*Conversation, 0)
	j.Get("data").ForEach(func(key, value gjson.Result) bool {
		conversation := &Conversation{
			ID:        value.Get("id").String(),
			Name:      value.Get("name").String(),
			Status:    value.Get("status").String(),
			CreatedAt: value.Get("created_at").String(),
			UpdatedAt: value.Get("updated_at").String(),
		}
		inputs := make(map[string]interface{})
		value.Get("inputs").ForEach(func(key, value gjson.Result) bool {
			inputs[key.String()] = value.Value()
			return true
		})
		conversation.Inputs = inputs
		conversations = append(conversations, conversation)
		return true
	})

	return conversations, nil
}

// 同时处理流式SSE响应或者阻塞式响应
func (c *ChatClient) SendChatMessage(req *ChatMessageRequest, apiSecret string, streamCallback func([]byte) error) ([]byte, error) {
	// 将请求体序列化为JSON
	reqBody, err := json.Marshal(req)
	if err != nil {
		logger.Error("序列化请求体失败", logger.F("err", err))
		return nil, err
	}
	// 创建HTTP请求
	httpReq, err := http.NewRequest("POST", c.baseUrl+"/chat-messages", bytes.NewBuffer(reqBody))
	if err != nil {
		logger.Error("创建请求失败", logger.F("err", err))
		return nil, err
	}

	// 设置请求头
	httpReq.Header.Set("Content-Type", "application/json")
	if apiSecret != "" {
		httpReq.Header.Set("Authorization", "Bearer "+apiSecret)
	} else if c.defaultAPISecret != "" {
		httpReq.Header.Set("Authorization", "Bearer "+c.defaultAPISecret)
	} else {
		logger.Error("未提供API密钥")
		return nil, fmt.Errorf("未提供API密钥")
	}

	// 发送请求
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		logger.Error("发送请求失败", logger.F("err", err))
		return nil, err
	}
	defer resp.Body.Close()

	// 检查响应状态码
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		logger.Error("API返回错误", logger.F("statusCode", resp.StatusCode), logger.F("response", string(body)))
		return nil, fmt.Errorf("API错误: %d, %s", resp.StatusCode, string(body))
	}

	// 根据不同的响应模式处理
	if req.ResponseMode == "streaming" {
		// 处理流式响应
		if streamCallback == nil {
			logger.Error("流式响应模式下必须提供streamCallback")
			return nil, fmt.Errorf("流式响应模式下必须提供streamCallback")
		}

		// 创建扫描器来读取SSE事件
		scanner := bufio.NewScanner(resp.Body)
		// 自定义分割函数，按照SSE格式的行进行分割
		scanner.Split(func(data []byte, atEOF bool) (advance int, token []byte, err error) {
			if atEOF && len(data) == 0 {
				return 0, nil, nil
			}
			if i := bytes.Index(data, []byte("\n\n")); i >= 0 {
				return i + 2, data[0:i], nil
			}
			if atEOF {
				return len(data), data, nil
			}
			return 0, nil, nil
		})

		var fullResponse bytes.Buffer
		// 逐行读取SSE事件
		for scanner.Scan() {
			eventData := scanner.Bytes()
			if len(eventData) == 0 {
				continue
			}

			// 解析事件数据
			if bytes.HasPrefix(eventData, []byte("data: ")) {
				// 提取data字段的内容
				data := eventData[6:] // 跳过 "data: "
				fullResponse.Write(data)

				// 通过回调函数处理数据
				if err := streamCallback(data); err != nil {
					logger.Error("处理流式响应失败", logger.F("err", err))
					return nil, err
				}
			}
		}

		if err := scanner.Err(); err != nil {
			logger.Error("读取流式响应失败", logger.F("err", err))
			return nil, err
		}

		return fullResponse.Bytes(), nil
	} else {
		// 阻塞式响应，直接读取完整响应
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			logger.Error("读取响应失败", logger.F("err", err))
			return nil, err
		}
		return body, nil
	}
}
