# 贡献指南

感谢您对Dify Tools项目的关注！我们欢迎任何形式的贡献，包括但不限于：

- 报告问题
- 提交功能建议
- 改进文档
- 提交代码

## 开发环境设置

1. Fork并克隆代码库：
```bash
git clone https://github.com/your-username/dify_tools.git
cd dify_tools
```

2. 安装依赖：
```bash
go mod tidy
```

3. 启动开发服务器：
```bash
go run cmd/server/main.go
```

## 代码风格

我们使用golangci-lint来保证代码质量，请确保您的代码符合以下标准：

1. 运行代码检查：
```bash
golangci-lint run
```

2. 代码格式化：
```bash
go fmt ./...
```

3. 代码风格要求：
- 使用gofmt标准格式
- 添加适当的注释
- 函数长度不超过50行
- 文件长度不超过500行
- 包名使用小写字母

## 提交规范

1. 分支命名：
- feature/xxx：新功能
- fix/xxx：修复bug
- docs/xxx：文档更新
- refactor/xxx：重构
- test/xxx：测试相关

2. 提交信息格式：
```
<类型>(<范围>): <描述>

[可选的正文]

[可选的脚注]
```

类型包括：
- feat：新功能
- fix：修复bug
- docs：文档更改
- style：代码格式（不影响代码运行的变动）
- refactor：重构（既不是新增功能，也不是修改bug的代码变动）
- test：增加测试
- chore：构建过程或辅助工具的变动

示例：
```
feat(monitor): 添加CPU使用率监控

添加了对数据库服务器CPU使用率的实时监控功能：
- 每分钟采集一次CPU使用率
- 当使用率超过80%时发出告警
- 支持配置告警阈值

Closes #123
```

## 测试规范

1. 单元测试要求：
- 所有新功能必须包含测试
- 测试覆盖率不低于80%
- 使用table-driven tests方式编写测试用例

2. 运行测试：
```bash
# 运行所有测试
go test ./...

# 运行特定包的测试
go test ./internal/app/...

# 生成测试覆盖率报告
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

## Pull Request流程

1. 创建Issue描述要解决的问题

2. 创建分支：
```bash
git checkout -b feature/xxx
```

3. 提交代码：
```bash
git add .
git commit -m "feat(xxx): add some feature"
git push origin feature/xxx
```

4. 创建Pull Request：
- 标题清晰描述变更内容
- 关联相关Issue
- 描述测试方法和结果
- 更新相关文档

5. 代码审查：
- 等待维护者审查
- 根据反馈进行修改
- 确保CI检查通过

## 文档规范

1. 代码注释：
- 包注释：包的主要功能说明
- 公开函数：函数功能、参数、返回值说明
- 复杂逻辑：说明实现思路和注意事项

2. API文档：
- 使用Swagger注释
- 包含请求参数说明
- 包含响应格式说明
- 提供示例

3. 更新文档：
- README.md：项目概述和快速开始
- deployment.md：部署和配置说明
- api.md：API使用说明

## 问题反馈

1. 提交Issue时请包含：
- 问题描述
- 复现步骤
- 期望结果
- 实际结果
- 环境信息
- 相关日志

2. 安全问题：
- 不要在Issue中提交安全漏洞
- 发送邮件到维护者邮箱
- 等待安全补丁发布

## 行为准则

- 尊重所有贡献者
- 接受建设性批评
- 关注问题本身
- 遵守开源协议

## 获取帮助

- 查阅文档
- 搜索已有Issue
- 创建新的Issue
- 联系维护者