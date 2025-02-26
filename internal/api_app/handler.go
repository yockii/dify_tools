package appapi

import "github.com/gofiber/fiber/v2"

var Handlers []Handler

type Handler interface {
	RegisterRoutes(router fiber.Router)
}

/*
对应用的接口，应用可以调用这些接口实现：
1、新增文档到知识库
2、查询知识库文档状态（是否已经处理）
3、用户问答聊天及回复（流式）
4、查询token使用量
*/
