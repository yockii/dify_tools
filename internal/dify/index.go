package dify

var (
	DefaultKnowledgeBaseClient *KnowledgeBaseClient
	DefaultChatClient          *ChatClient
)

func InitDefaultKnowledgeBaseClient(baseUrl, defaultAPISecret string) *KnowledgeBaseClient {
	DefaultKnowledgeBaseClient = NewKnowLedgeBaseClient(baseUrl, defaultAPISecret)
	return DefaultKnowledgeBaseClient
}

func GetDefaultKnowledgeBaseClient() *KnowledgeBaseClient {
	return DefaultKnowledgeBaseClient
}

func InitDefaultChatClient(baseUrl, defaultAPISecret string) *ChatClient {
	DefaultChatClient = NewChatClient(baseUrl, defaultAPISecret)
	return DefaultChatClient
}

func GetDefaultChatClient() *ChatClient {
	return DefaultChatClient
}
