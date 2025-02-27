package dify

import (
	"bytes"
	"encoding/json"
	"net/http"
)

type KnowledgeBaseClient struct {
	baseUrl          string
	defaultAPISecret string
	httpClient       *http.Client
}

func NewKnowLedgeBaseClient(baseUrl, defaultAPISecret string) *KnowledgeBaseClient {
	httpClient := &http.Client{}

	return &KnowledgeBaseClient{
		baseUrl:          baseUrl,
		defaultAPISecret: defaultAPISecret,
		httpClient:       httpClient,
	}
}

func (c *KnowledgeBaseClient) buildPostRequest(url string, body []byte) (*http.Request, error) {
	req, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	if c.defaultAPISecret != "" {
		req.Header.Set("Authorization", "Bearer "+c.defaultAPISecret)
	}

	return req, nil
}

func (c *KnowledgeBaseClient) CreateDocumentByText(ID, docName, docContent string, docMetadata map[string]string) error {
	body := map[string]interface{}{
		"name":               docName,
		"text":               docContent,
		"doc_metadata":       docMetadata,
		"indexing_technique": "high_quality",
		"doc_form":           "hierarchical_model",
		"process_rule": map[string]interface{}{
			"mode": "automatic",
		},
		"retrieval_model": map[string]interface{}{
			"search_method":           "hybrid_search",
			"reranking_enable":        false,
			"top_k":                   5,
			"score_threshold_enabled": false,
		},
	}
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := c.buildPostRequest(c.baseUrl+"/datasets/"+ID+"/documents/create-by-text", bodyBytes)
	if err != nil {
		return err
	}
	c.httpClient.Do(req)
	return nil
}
