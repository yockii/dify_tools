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

func (h *KnowledgeBaseHandler) RegisterRoutesV1_1(router fiber.Router) {
	router.Post("/retrieval", h.RetrievalV1_1)
}

func (h *KnowledgeBaseHandler) RegisterRoutesV1(router fiber.Router) {
	router.Post("/retrieval", h.Retrieval)
}

type RetrievalSetting struct {
	TopK           int     `json:"top_k"`
	ScoreThreshold float64 `json:"score_threshold"`
}

type Condition struct {
	Name               string `json:"name"`
	ComparisonOperator string `json:"comparison_operator"`
	Value              string `json:"value"`
}

type MetadataCondition struct {
	LogicalOperator string    `json:"logical_operator"`
	Conditions      Condition `json:"conditions"`
}

type DifyRetrievalRequest struct {
	KnowledgeID       string            `json:"knowledge_id"`
	Query             string            `json:"query"`
	RetrievalSetting  RetrievalSetting  `json:"retrieval_setting"`
	MetadataCondition MetadataCondition `json:"metadata_condition"`
}

type Record struct {
	Content  string         `json:"content"`
	Score    float64        `json:"score"`
	Title    string         `json:"title"`
	Metadata map[string]any `json:"metadata"`
}

type DifyRetrievalResponse struct {
	Records []Record `json:"records"`
}

func (h *KnowledgeBaseHandler) RetrievalV1_1(c *fiber.Ctx) error {
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
		knowledgeBase, err = h.knowledgeBaseService.GetByApplicationIDAndCustomID(c.Context(), app.ID, "")
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error_code": 500,
				"error_msg":  "get knowledge base failed",
			})
		}
	} else {
		var err error
		knowledgeBase, err = h.knowledgeBaseService.GetByApplicationIDAndCustomID(c.Context(), 0, "")
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

	resp, err := kbClient.Retrieve(knowledgeBase.OuterID, query, req.RetrievalSetting.TopK, req.RetrievalSetting.ScoreThreshold, map[string]interface{}{
		"mode": "manual",
		"condition": map[string]interface{}{
			"operator": "or",
			"conditions": []map[string]interface{}{
				{
					"field_name": "custom_id",
					"operator":   "is",
					"value":      customID,
				},
				{
					"field_name": "custom_id",
					"operator":   "not exists",
				},
			},
		},
	})
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
			Content: record.Get("segment.content").String(),
			Score:   record.Get("score").Float(),
			Title:   record.Get("segment.document.name").String(),
			// Metadata: record.Get("segment.document.doc_metadata").Value().(map[string]interface{}),
		}
		metaData := map[string]interface{}{}
		if record.Get("segment.document.doc_metadata").Exists() {
			metaData = record.Get("segment.document.doc_metadata").Value().(map[string]interface{})
		}
		r.Metadata = metaData
		result = append(result, r)
	}

	return c.JSON(&DifyRetrievalResponse{
		Records: result,
	})
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
	var publickKnowledgeBase *model.KnowledgeBase
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
		publickKnowledgeBase, err = h.knowledgeBaseService.GetByApplicationIDAndCustomID(c.Context(), app.ID, "")
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
		publickKnowledgeBase, err = h.knowledgeBaseService.GetByApplicationIDAndCustomID(c.Context(), 0, "")
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error_code": 500,
				"error_msg":  "get knowledge base failed",
			})
		}
	}
	if knowledgeBase == nil && publickKnowledgeBase == nil {
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

	var result []Record

	if knowledgeBase != nil {
		resp, err := kbClient.Retrieve(knowledgeBase.OuterID, query, req.RetrievalSetting.TopK, req.RetrievalSetting.ScoreThreshold, nil)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error_code": 500,
				"error_msg":  "retrieve failed",
			})
		}
		result = append(result, h.getRecordsFromResponse(resp)...)
	}

	if publickKnowledgeBase != nil {
		resp, err := kbClient.Retrieve(publickKnowledgeBase.OuterID, query, req.RetrievalSetting.TopK, req.RetrievalSetting.ScoreThreshold, nil)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error_code": 500,
				"error_msg":  "retrieve failed",
			})
		}
		result = append(result, h.getRecordsFromResponse(resp)...)
	}

	return c.JSON(&DifyRetrievalResponse{
		Records: result,
	})
}

func (h *KnowledgeBaseHandler) getRecordsFromResponse(resp string) []Record {
	respJson := gjson.Parse(resp)
	records := respJson.Get("records").Array()

	var result []Record
	for _, record := range records {
		r := Record{
			Content: record.Get("segment.content").String(),
			Score:   record.Get("score").Float(),
			Title:   record.Get("segment.document.name").String(),
			// Metadata: record.Get("segment.document.doc_metadata").Value().(map[string]interface{}),
		}
		metaData := map[string]any{}
		if record.Get("segment.document.doc_metadata").Exists() {
			// metaData = record.Get("segment.document.doc_metadata").Value().(map[string]any)
		}
		r.Metadata = metaData
		result = append(result, r)
	}
	return result
}
