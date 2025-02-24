# Dify Tools

数据库工具集，提供数据查询、监控和分析功能。

## 功能特性

- 数据源管理
  * 支持MySQL和PostgreSQL
  * 数据库连接管理
  * 表结构同步
  * 权限控制

- 查询功能
  * SQL查询执行
  * SQL语句验证
  * 查询计划分析
  * 查询历史记录

- 性能监控
  * 慢查询监控
  * 资源使用监控
  * 自动告警通知
  * 性能统计分析

- 权限管理
  * 基于角色的访问控制
  * 表级别白名单
  * 列级别数据脱敏
  * 权限继承关系

## 快速开始

1. 安装依赖
```bash
go mod tidy
```

2. 配置
复制配置文件模板并修改：
```bash
cp config.example.yaml config.yaml
```

3. 启动服务
```bash
go run cmd/server/main.go
```

4. 访问API文档
```
http://localhost:8080/swagger/
```

## 配置说明

数据库配置：
```yaml
database:
  type: mysql # mysql or postgres
  host: localhost
  port: 3306
  user: root
  password: root
  dbname: dify_tools
```

邮件通知配置：
```yaml
email:
  enabled: true
  host: smtp.example.com
  port: 587
  from: noreply@example.com
  password: your-password
```

监控配置：
```yaml
monitor:
  slow_query:
    threshold: 1000 # 慢查询阈值（毫秒）
    max_rows: 10000 # 最大行数限制
  resource:
    cpu_threshold: 80 # CPU使用率告警阈值
    memory_threshold: 80 # 内存使用率告警阈值
```

## API文档

项目使用Swagger生成API文档，启动服务后访问：
```
http://localhost:8080/swagger/
```

主要API接口：
- `/api/v1/datasources` - 数据源管理
- `/api/v1/datasources/{id}/execute` - 执行查询
- `/api/v1/datasources/{id}/explain` - 查询计划分析
- `/api/v1/monitor` - 性能监控
- `/api/v1/history` - 查询历史

## 开发

1. 目录结构
```
.
├── cmd/                # 入口程序
├── docs/              # 文档
├── internal/          # 内部包
│   ├── app/          # 应用逻辑
│   │   ├── api/     # API处理器
│   │   ├── model/   # 数据模型
│   │   └── service/ # 业务逻辑
│   ├── core/        # 核心功能
│   │   ├── middleware/ # 中间件
│   │   └── server/    # 服务器
│   ├── pkg/         # 公共包
│   └── user/        # 用户模块
├── config.yaml      # 配置文件
└── go.mod          # Go模块文件
```

2. 测试
```bash
go test ./...
```

## 贡献

欢迎提交Issue和Pull Request。

## 许可证

MIT License