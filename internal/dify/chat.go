package dify

import (
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
	ResponseMode     string                 `json:"response_mode"`
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

func (c *ChatClient) GetConversations(user string) ([]*Conversation, error) {
	req, err := http.NewRequest("GET", c.baseUrl+"/conversations", nil)
	if err != nil {
		logger.Error("创建请求失败", logger.F("err", err))
		return nil, err
	}
	if c.defaultAPISecret != "" {
		req.Header.Set("Authorization", "Bearer "+c.defaultAPISecret)
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
