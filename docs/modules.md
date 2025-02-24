# 系统模块设计

## 模块总览

### 1. 核心模块 (core)
- 配置管理
- 日志管理
- 错误处理
- 工具类库
- 中间件
- 通用常量

### 2. 用户模块 (user)
- 用户管理
- 角色管理
- 权限管理
- 认证授权
- 操作日志
- 行为分析

### 3. 应用管理模块 (app)
- 应用信息管理
- 应用配置管理
- 应用密钥管理
- 数据源管理
- 数据源同步
- 应用使用统计

### 4. AI应用模块 (ai)
- AI模型管理
- 应用场景配置
- 模型调用管理
- 效果反馈
- 参数优化
- 会话管理

### 5. 工具模块 (tool)
- 数据库查询工具
- 网络搜索工具
- 工具注册管理
- 工具调用链
- 调用记录
- 性能分析

### 6. 集成模块 (integration)
- 钉钉机器人
- 定时任务
- 三方系统对接
- 消息通知
- 数据同步
- 接口适配

### 7. 监控模块 (monitor)
- 系统监控
- 性能监控
- 业务监控
- 告警管理
- 资源使用统计
- 健康检查

## 详细设计

### 1. 核心模块 (core)
#### 1.1 配置管理
```go
package config

type Config struct {
    Server   ServerConfig
    Database DatabaseConfig
    Redis    RedisConfig
    Log      LogConfig
    Security SecurityConfig
}

type ServerConfig struct {
    Port     int
    Mode     string
    TimeZone string
}

type SecurityConfig struct {
    PasswordSalt    string
    JWTSecret      string
    APIKeyPrefix   string
    AllowedOrigins []string
}
```

#### 1.2 中间件
```go
package middleware

// JWT认证中间件
func JWTAuth() fiber.Handler

// API密钥认证中间件
func APIKeyAuth() fiber.Handler

// 访问日志中间件
func AccessLog() fiber.Handler

// 错误处理中间件
func ErrorHandler() fiber.Handler

// 请求追踪中间件
func RequestTracing() fiber.Handler
```

### 2. 用户模块 (user)
#### 2.1 数据模型
```go
package user

type User struct {
    ID          string    `json:"id"`
    Username    string    `json:"username"`
    Password    string    `json:"-"`
    Role        string    `json:"role"`
    Status      int       `json:"status"`
    LastLoginAt time.Time `json:"lastLoginAt"`
    CreatedAt   time.Time `json:"createdAt"`
    UpdatedAt   time.Time `json:"updatedAt"`
}

type Role struct {
    ID          string   `json:"id"`
    Name        string   `json:"name"`
    Permissions []string `json:"permissions"`
}

type UserLog struct {
    ID        string    `json:"id"`
    UserID    string    `json:"userId"`
    Action    string    `json:"action"`
    Resource  string    `json:"resource"`
    Status    string    `json:"status"`
    IP        string    `json:"ip"`
    UserAgent string    `json:"userAgent"`
    CreatedAt time.Time `json:"createdAt"`
}
```

#### 2.2 服务接口
```go
package user

type UserService interface {
    Create(user *User) error
    Update(user *User) error
    Delete(id string) error
    FindByID(id string) (*User, error)
    List(params QueryParams) ([]*User, int64, error)
    LogAction(log *UserLog) error
    GetUserStats(userID string) (*UserStats, error)
}

type AuthService interface {
    Login(username, password string) (string, error)
    Verify(token string) (*Claims, error)
    Refresh(token string) (string, error)
    ValidatePermission(userID string, resource string, action string) bool
}
```

### 3. 应用管理模块 (app)
#### 3.1 数据模型
```go
package app

type Application struct {
    ID          string    `json:"id"`
    Name        string    `json:"name"`
    Description string    `json:"description"`
    APIKey      string    `json:"apiKey"`
    Status      int       `json:"status"`
    Config      AppConfig `json:"config"`
    CreatedAt   time.Time `json:"createdAt"`
    UpdatedAt   time.Time `json:"updatedAt"`
}

type AppConfig struct {
    AIModels       []string         `json:"aiModels"`
    MaxTokens      int              `json:"maxTokens"`
    Database       DatabaseConfig   `json:"database"`
    AllowedOrigins []string        `json:"allowedOrigins"`
    Limits         RateLimitConfig `json:"limits"`
}

type DataSource struct {
    ID        string       `json:"id"`
    AppID     string       `json:"appId"`
    Name      string       `json:"name"`
    Type      string       `json:"type"`
    Config    interface{}  `json:"config"`
    Tables    []TableInfo `json:"tables"`
    SyncTime  time.Time   `json:"syncTime"`
    Status    int         `json:"status"`
}
```

#### 3.2 服务接口
```go
package app

type AppService interface {
    Create(app *Application) error
    Update(app *Application) error
    UpdateConfig(id string, config AppConfig) error
    Delete(id string) error
    FindByID(id string) (*Application, error)
    FindByAPIKey(apiKey string) (*Application, error)
    List(params QueryParams) ([]*Application, int64, error)
    GetUsageStats(appID string) (*AppStats, error)
}

type DataSourceService interface {
    Create(ds *DataSource) error
    Update(ds *DataSource) error
    SyncSchema(dsID string) error
    ValidateConnection(config interface{}) error
    GetTables(dsID string) ([]TableInfo, error)
}
```

### 4. AI应用模块 (ai)
#### 4.1 数据模型
```go
package ai

type AIApp struct {
    ID        string      `json:"id"`
    Name      string      `json:"name"`
    Type      string      `json:"type"`
    APIKey    string      `json:"apiKey"`
    Config    AIConfig    `json:"config"`
    Status    int         `json:"status"`
    CreatedAt time.Time   `json:"createdAt"`
    UpdatedAt time.Time   `json:"updatedAt"`
}

type AIConfig struct {
    Model       string                 `json:"model"`
    Temperature float64                `json:"temperature"`
    MaxTokens   int                    `json:"maxTokens"`
    Stop        []string               `json:"stop"`
    Extra       map[string]interface{} `json:"extra"`
}

type Feedback struct {
    ID        string    `json:"id"`
    AppID     string    `json:"appId"`
    SessionID string    `json:"sessionId"`
    Prompt    string    `json:"prompt"`
    Response  string    `json:"response"`
    Rating    int       `json:"rating"`
    Comment   string    `json:"comment"`
    CreatedAt time.Time `json:"createdAt"`
}

type Session struct {
    ID           string    `json:"id"`
    AppID        string    `json:"appId"`
    UserID       string    `json:"userId"`
    Context      []Message `json:"context"`
    LastActivity time.Time `json:"lastActivity"`
    CreatedAt    time.Time `json:"createdAt"`
}
```

#### 4.2 服务接口
```go
package ai

type AIService interface {
    Generate(ctx context.Context, prompt string, config AIConfig) (string, error)
    Chat(ctx context.Context, message string, sessionID string) (string, error)
    Stream(ctx context.Context, prompt string, config AIConfig) (<-chan string, error)
    SaveFeedback(feedback *Feedback) error
    GetModelStats(modelID string) (*ModelStats, error)
}

type SessionService interface {
    Create(session *Session) error
    Update(session *Session) error
    GetContext(sessionID string) ([]Message, error)
    CleanupExpired() error
}
```

### 5. 工具模块 (tool)
#### 5.1 数据模型
```go
package tool

type Tool struct {
    ID          string      `json:"id"`
    Name        string      `json:"name"`
    Type        string      `json:"type"`
    Description string      `json:"description"`
    Config      ToolConfig  `json:"config"`
    Status      int         `json:"status"`
    CreatedAt   time.Time   `json:"createdAt"`
}

type ToolConfig struct {
    Endpoint    string                 `json:"endpoint"`
    Method      string                 `json:"method"`
    Headers     map[string]string      `json:"headers"`
    Parameters  map[string]interface{} `json:"parameters"`
}

type ToolExecution struct {
    ID        string                 `json:"id"`
    ToolID    string                `json:"toolId"`
    Input     map[string]interface{} `json:"input"`
    Output    interface{}            `json:"output"`
    Duration  int64                  `json:"duration"`
    Status    string                 `json:"status"`
    Error     string                 `json:"error"`
    CreatedAt time.Time             `json:"createdAt"`
}
```

#### 5.2 服务接口
```go
package tool

type ToolService interface {
    Execute(ctx context.Context, toolID string, params map[string]interface{}) (interface{}, error)
    Register(tool *Tool) error
    List() ([]*Tool, error)
    FindByID(id string) (*Tool, error)
    LogExecution(exec *ToolExecution) error
    GetToolStats(toolID string) (*ToolStats, error)
}

type DBQueryTool interface {
    Query(ctx context.Context, appID string, sql string) ([]map[string]interface{}, error)
    ValidateSQL(sql string) error
    GetQueryStats(appID string) (*QueryStats, error)
}

type WebSearchTool interface {
    Search(ctx context.Context, query string, maxResults int) ([]SearchResult, error)
    GetSearchStats() (*SearchStats, error)
}
```

### 6. 集成模块 (integration)
#### 6.1 钉钉集成
```go
package dingtalk

type RobotService interface {
    SendMessage(groupID string, message string) error
    HandleMessage(message *Message) (string, error)
    GetRobotStats(robotID string) (*RobotStats, error)
}

type NotifyService interface {
    SendNotification(target string, message string, msgType string) error
    GetNotifyStats(target string) (*NotifyStats, error)
}
```

#### 6.2 定时任务
```go
package scheduler

type Task struct {
    ID        string    `json:"id"`
    Name      string    `json:"name"`
    Cron      string    `json:"cron"`
    Action    string    `json:"action"`
    Params    string    `json:"params"`
    Status    int       `json:"status"`
    LastRun   time.Time `json:"lastRun"`
    NextRun   time.Time `json:"nextRun"`
    CreatedAt time.Time `json:"createdAt"`
}

type TaskService interface {
    Schedule(task *Task) error
    Cancel(taskID string) error
    List() ([]*Task, error)
    GetTaskStats(taskID string) (*TaskStats, error)
}
```

### 7. 监控模块 (monitor)
#### 7.1 监控指标
```go
package monitor

type Metrics struct {
    CPU        float64   `json:"cpu"`
    Memory     float64   `json:"memory"`
    Goroutines int       `json:"goroutines"`
    Requests   int64     `json:"requests"`
    Errors     int64     `json:"errors"`
    Timestamp  time.Time `json:"timestamp"`
}

type ResourceUsage struct {
    AppID     string    `json:"appId"`
    Resource  string    `json:"resource"`
    Usage     int64     `json:"usage"`
    Limit     int64     `json:"limit"`
    Timestamp time.Time `json:"timestamp"`
}
```

#### 7.2 服务接口
```go
package monitor

type MonitorService interface {
    CollectMetrics() (*Metrics, error)
    GetSystemStatus() (*SystemStatus, error)
    GetAPIStats() (*APIStats, error)
    SendAlert(alert *Alert) error
    TrackResourceUsage(usage *ResourceUsage) error
    GetHealthStatus() (*HealthStatus, error)
}
```

## 模块依赖关系
```
core
  ├─ user
  ├─ app
  ├─ ai
  ├─ tool
  ├─ integration
  └─ monitor

user
  └─ app

app
  ├─ ai
  └─ tool

ai
  ├─ tool
  └─ monitor

integration
  ├─ ai
  └─ tool

monitor
  ├─ app
  ├─ ai
  ├─ tool
  └─ integration