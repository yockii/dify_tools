package difyapi

import (
	"github.com/gofiber/fiber/v2"
	"github.com/tidwall/gjson"
	"github.com/yockii/dify_tools/internal/model"
	"github.com/yockii/dify_tools/internal/service"
)

type KnowledgeBaseHandler struct {
	applicationService   service.ApplicationService
	knowledgeBaseService service.KnowledgeBaseService
}

func RegisterKnowledgeBaseHandler(
	applicationService service.ApplicationService,
	knowledgeBaseService service.KnowledgeBaseService,
) {
	handler := &KnowledgeBaseHandler{
		applicationService:   applicationService,
		knowledgeBaseService: knowledgeBaseService,
	}
	Handlers = append(Handlers, handler)
}

func (h *KnowledgeBaseHandler) RegisterRoutes(router fiber.Router) {
	router.Post("/retrieval", h.Retrieval)
}

type RetrievalSetting struct {
	TopK           int     `json:"top_k"`
	ScoreThreshold float64 `json:"score_threshold"`
}

type DifyRetrievalRequest struct {
	KnowledgeID      string           `json:"knowledgeId"`
	Query            string           `json:"query"`
	RetrievalSetting RetrievalSetting `json:"retrievalSetting"`
}

type Record struct {
	Content  string                 `json:"content"`
	Score    float64                `json:"score"`
	Title    string                 `json:"title"`
	Metadata map[string]interface{} `json:"metadata"`
}

type DifyRetrievalResponse struct {
	Records []interface{} `json:"records"`
}

func (h *KnowledgeBaseHandler) Retrieval(c *fiber.Ctx) error {
	var req DifyRetrievalRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error_code": 500,
			"error_msg":  "invalid request",
		})
	}
	if req.Query == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error_code": 400,
			"error_msg":  "query is required",
		})
	}
	qj := gjson.Parse(req.Query)
	ak := qj.Get("app_secret").String()
	customID := qj.Get("custom_id").String()
	query := qj.Get("query").String()
	if ak == "" && customID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error_code": 400,
			"error_msg":  "app_secret or custom_id are required",
		})
	}

	var knowledgeBase *model.KnowledgeBase
	if ak != "" {
		app, err := h.applicationService.GetByApiKey(c.Context(), ak)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error_code": 500,
				"error_msg":  "invalid app_secret",
			})
		}
		if app == nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error_code": 400,
				"error_msg":  "invalid app_secret",
			})
		}
		knowledgeBase, err = h.knowledgeBaseService.GetByApplicationIDAndCustomID(c.Context(), app.ID, customID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error_code": 500,
				"error_msg":  "get knowledge base failed",
			})
		}
	} else {
		var err error
		knowledgeBase, err = h.knowledgeBaseService.GetByApplicationIDAndCustomID(c.Context(), 0, customID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error_code": 500,
				"error_msg":  "get knowledge base failed",
			})
		}
	}
	if knowledgeBase == nil {
		return c.JSON(fiber.Map{
			"error_code": 2001,
			"error_msg":  "知识库不存在：" + ak + " | " + customID,
		})
	}

	kbClient, err := h.knowledgeBaseService.GetDifyKnowledgeBaseClient(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error_code": 500,
			"error_msg":  "get dify knowledge base client failed",
		})
	}

	resp, err := kbClient.Retrieve(knowledgeBase.OuterID, query, req.RetrievalSetting.TopK, req.RetrievalSetting.ScoreThreshold)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error_code": 500,
			"error_msg":  "retrieve failed",
		})
	}

	respJson := gjson.Parse(resp)
	records := respJson.Get("records").Array()

	var result []Record
	for _, record := range records {
		r := Record{
			Content:  record.Get("segment.content").String(),
			Score:    record.Get("score").Float(),
			Title:    record.Get("document.name").String(),
			Metadata: record.Get("metadata").Value().(map[string]interface{}),
		}
		result = append(result, r)
	}

	return c.JSON(fiber.Map{
		"records": result,
	})
}
