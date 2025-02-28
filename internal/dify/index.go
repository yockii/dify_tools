package dify

var (
	DefaultKnowledgeBaseClient *KnowledgeBaseClient
)

func InitDefaultKnowledgeBaseClient(baseUrl, defaultAPISecret string) *KnowledgeBaseClient {
	DefaultKnowledgeBaseClient = NewKnowLedgeBaseClient(baseUrl, defaultAPISecret)
	return DefaultKnowledgeBaseClient
}

func GetDefaultKnowledgeBaseClient() *KnowledgeBaseClient {
	return DefaultKnowledgeBaseClient
}
