package server

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	middlewareLogger "github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	appapi "github.com/yockii/dify_tools/internal/api_app"
	difyapi "github.com/yockii/dify_tools/internal/api_dify"
	sysapi "github.com/yockii/dify_tools/internal/api_sys"
	"github.com/yockii/dify_tools/internal/middleware"
	"github.com/yockii/dify_tools/internal/service"
	"github.com/yockii/dify_tools/pkg/config"
	"github.com/yockii/dify_tools/pkg/logger"
)

type Server struct {
	app *fiber.App

	// 各个service
	userSrv          service.UserService
	roleSrv          service.RoleService
	sessionSrv       service.SessionService
	authSrv          service.AuthService
	logSrv           service.LogService
	applicationSrv   service.ApplicationService
	dataSourceSrv    service.DataSourceService
	tableInfoSrv     service.TableInfoService
	columnInfoSrv    service.ColumnInfoService
	dictSrv          service.DictService
	knowledgeBaseSrv service.KnowledgeBaseService
	documentSrv      service.DocumentService
	agentSrv         service.AgentService
	usageSrv         service.UsageService
}

func New() *Server {
	return &Server{}
}

func (s *Server) Start() error {
	// 创建Fiber实例
	s.app = fiber.New(fiber.Config{
		AppName:               config.GetString("server.app_name"),
		EnablePrintRoutes:     config.GetBool("server.print_routes"),
		DisableStartupMessage: true,
	})

	s.setupServices()

	// 配置中间件
	s.setupMiddleware()

	// 注册系统路由
	s.registerSysHandler()
	s.registerDifyHandler()

	// 配置系统路由
	s.setupSystemRoutesV1()
	s.setupSystemRoutesV1_1()
	// 配置DIFY路由
	s.setupDifyRoutesV1()
	s.setupDifyRoutesV1_1()
	// 配置应用路由
	s.setupApplicationRoutesV1()

	// 启动服务器
	addr := config.GetServerAddress()
	logger.Info("服务监听地址", logger.F("address", addr))

	// 优雅关闭
	go s.gracefulShutdown()

	if err := s.app.Listen(addr); err != nil {
		logger.Error("服务停止", logger.F("error", err))
		return err
	}
	return nil
}

func (s *Server) gracefulShutdown() {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("服务关闭中...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := s.app.ShutdownWithContext(ctx); err != nil {
		logger.Error("服务关闭失败", logger.F("error", err))
	}

	logger.Info("服务已关闭")
}

// setupServices 配置服务层
func (s *Server) setupServices() {
	// 创建服务实例
	s.dictSrv = service.NewDictService()

	s.userSrv = service.NewUserService()
	s.roleSrv = service.NewRoleService()
	s.sessionSrv = service.NewSessionService()
	s.authSrv = service.NewAuthService(s.userSrv, s.sessionSrv)
	s.logSrv = service.NewLogService()

	s.applicationSrv = service.NewApplicationService(s.dictSrv)
	s.dataSourceSrv = service.NewDataSourceService()
	s.tableInfoSrv = service.NewTableInfoService()
	s.columnInfoSrv = service.NewColumnInfoService()

	s.knowledgeBaseSrv = service.NewKnowledgeBaseService(s.dictSrv, s.applicationSrv)
	s.documentSrv = service.NewDocumentService(s.dictSrv, s.applicationSrv, s.knowledgeBaseSrv)

	s.agentSrv = service.NewAgentService()
	s.usageSrv = service.NewUsageService()
}

// setupMiddleware 配置中间件
func (s *Server) setupMiddleware() {
	// 异常恢复
	s.app.Use(recover.New())

	// CORS
	s.app.Use(cors.New(cors.Config{
		AllowOrigins: config.GetString("security.allowed_origins"),
		AllowHeaders: "Origin, Content-Type, Accept, Authorization",
	}))

	// 访问日志
	s.app.Use(middlewareLogger.New(middlewareLogger.Config{
		Format:     "[${ip}]-${time} ${status} ${latency} ${method} ${path} | ${error}\n",
		TimeFormat: "2006-01-02 15:04:05",
		TimeZone:   "Local",
	}))
}

func (s *Server) registerSysHandler() {
	sysapi.RegisterUserHandler(
		s.userSrv,
		s.authSrv,
		s.roleSrv,
		s.logSrv,
		s.sessionSrv,
	)
	sysapi.RegisterAppHandler(
		s.applicationSrv,
		s.dataSourceSrv,
		s.tableInfoSrv,
		s.columnInfoSrv,
		s.knowledgeBaseSrv,
		s.usageSrv,
		s.logSrv,
	)
	sysapi.RegisterDictHandler(
		s.dictSrv,
		s.logSrv,
	)
	sysapi.RegisterKnowledgeBaseHandler(
		s.knowledgeBaseSrv,
		s.documentSrv,
		s.logSrv,
	)
	sysapi.RegisterAgentHandler(
		s.agentSrv,
	)
	sysapi.RegisterChatHandler(
		s.dictSrv,
		s.applicationSrv,
		s.usageSrv,
	)
}

// setupSystemRoutesV1 配置系统路由对应dify版本 v1.0.1
func (s *Server) setupSystemRoutesV1() {
	// 中间件
	sysAuthMiddleware := middleware.NewAuthMiddleware(s.authSrv, s.sessionSrv, nil)

	// API路由组
	apiGroup := s.app.Group("/sys_api/v1")

	// 注册用户路由
	for _, handler := range sysapi.Handlers {
		handler.RegisterRoutesV1(apiGroup, sysAuthMiddleware)
	}

	// 健康检查
	s.app.Get("/health", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})
}

// setupSystemRoutesV1 配置系统路由对应dify版本 v1.1.0
func (s *Server) setupSystemRoutesV1_1() {
	// 中间件
	sysAuthMiddleware := middleware.NewAuthMiddleware(s.authSrv, s.sessionSrv, nil)

	// API路由组
	apiGroup := s.app.Group("/sys_api/v1_1")

	// 注册用户路由
	for _, handler := range sysapi.Handlers {
		handler.RegisterRoutesV1_1(apiGroup, sysAuthMiddleware)
	}

	// 健康检查
	s.app.Get("/health", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})
}

func (s *Server) registerDifyHandler() {
	difyapi.RegisterDatabaseHandler(
		s.applicationSrv,
		s.dataSourceSrv,
		s.tableInfoSrv,
		s.columnInfoSrv,
	)
	difyapi.RegisterKnowledgeBaseHandler(
		s.applicationSrv,
		s.knowledgeBaseSrv,
	)
}

// setupDifyRoutesV1 配置DIFY路由对应dify版本 v1.0.1
func (s *Server) setupDifyRoutesV1() {
	difyApiGroup := s.app.Group("/dify_api/v1")
	// 注册路由
	for _, handler := range difyapi.Handlers {
		handler.RegisterRoutesV1(difyApiGroup)
	}
}

// setupDifyRoutesV1_1 配置DIFY路由对应dify版本 v1.1.0
func (s *Server) setupDifyRoutesV1_1() {
	difyApiGroup := s.app.Group("/dify_api/v1_1")
	// 注册路由
	for _, handler := range difyapi.Handlers {
		handler.RegisterRoutesV1_1(difyApiGroup)
	}
}

// setupApplicationRoutesV1 配置应用路由对应dify版本 v1.0.1
func (s *Server) setupApplicationRoutesV1() {
	appapi.RegisterDocumentHandler(
		s.knowledgeBaseSrv,
		s.documentSrv,
	)

	appAuthMiddleware := middleware.NewAppMiddleware(s.applicationSrv)
	appApiGroup := s.app.Group("/api/v1", appAuthMiddleware)
	for _, handler := range appapi.Handlers {
		handler.RegisterRoutes(appApiGroup)
	}
}
