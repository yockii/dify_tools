package dify

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"

	"github.com/tidwall/gjson"
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

func (c *KnowledgeBaseClient) CreateDocumentByText(ID, docName, docContent string, docMetadata map[string]string) (string, error) {
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
		return "", err
	}
	req, err := c.buildPostRequest(c.baseUrl+"/datasets/"+ID+"/documents/create-by-text", bodyBytes)
	if err != nil {
		return "", err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	response, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	respJson := gjson.ParseBytes(response)
	docId := respJson.Get("batch").String()
	return docId, nil
}

func (c *KnowledgeBaseClient) CreateDocumentByFile(ID string, fileHeader multipart.FileHeader, docMetadata map[string]string) (string, error) {
	fileBody := &bytes.Buffer{}
	writer := multipart.NewWriter(fileBody)
	part, err := writer.CreateFormFile("file", fileHeader.Filename)
	if err != nil {
		return "", err
	}
	file, err := fileHeader.Open()
	if err != nil {
		return "", err
	}
	_, err = io.Copy(part, file)
	if err != nil {
		return "", err
	}
	// data=json, file=upload
	body := map[string]interface{}{
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
	// body json序列化后的字符串放入data字段
	dataBytes, err := json.Marshal(body)
	if err != nil {
		return "", err
	}
	err = writer.WriteField("data", string(dataBytes))
	if err != nil {
		return "", err
	}
	err = writer.Close()
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", c.baseUrl+"/datasets/"+ID+"/documents/create-by-file", fileBody)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	if c.defaultAPISecret != "" {
		req.Header.Set("Authorization", "Bearer "+c.defaultAPISecret)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	response, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	respJson := gjson.ParseBytes(response)
	docId := respJson.Get("batch").String()
	return docId, nil
}

func (c *KnowledgeBaseClient) CreateKnowledgeBase(name, description string) (string, error) {
	body := map[string]interface{}{
		"name":               name,
		"description":        description,
		"indexing_technique": "high_quality",
		"permission":         "all_team_members",
		"provider":           "vendor",
	}
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return "", err
	}
	req, err := c.buildPostRequest(c.baseUrl+"/datasets", bodyBytes)
	if err != nil {
		return "", err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	response, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	respJson := gjson.ParseBytes(response)
	datasetId := respJson.Get("id").String()
	return datasetId, nil
}

func (c *KnowledgeBaseClient) DeleteKnowledgeBase(ID string) error {
	req, err := http.NewRequest("DELETE", c.baseUrl+"/datasets/"+ID, nil)
	if err != nil {
		return err
	}
	if c.defaultAPISecret != "" {
		req.Header.Set("Authorization", "Bearer "+c.defaultAPISecret)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("delete knowledge base failed, status code: %d", resp.StatusCode)
	}
	return nil
}

func (c *KnowledgeBaseClient) DocumentBatchIndexingStatus(ID, batchID string) (string, error) {
	req, err := http.NewRequest("GET", c.baseUrl+"/datasets/"+ID+"/documents/"+batchID+"/indexing-status", nil)
	if err != nil {
		return "", err
	}
	if c.defaultAPISecret != "" {
		req.Header.Set("Authorization", "Bearer "+c.defaultAPISecret)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	response, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	respJson := gjson.ParseBytes(response)
	status := respJson.Get("status").String()
	return status, nil
}
