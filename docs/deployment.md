# 部署指南

本文档介绍如何部署和配置Dify Tools服务。

## 部署方式

### 1. Docker部署（推荐）

使用Docker Compose是最简单的部署方式：

```bash
# 克隆仓库
git clone https://github.com/yockii/dify_tools.git
cd dify_tools

# 复制配置文件
cp config.example.yaml config.yaml

# 修改配置文件
vim config.yaml

# 启动服务
docker-compose up -d
```

服务默认运行在 http://localhost:8080

### 2. 二进制部署

1. 下载预编译的二进制文件：
   ```bash
   # Linux
   wget https://github.com/yockii/dify_tools/releases/latest/download/dify-tools-linux-amd64
   
   # MacOS
   wget https://github.com/yockii/dify_tools/releases/latest/download/dify-tools-darwin-amd64
   
   # Windows
   wget https://github.com/yockii/dify_tools/releases/latest/download/dify-tools-windows-amd64.exe
   ```

2. 准备配置文件：
   ```bash
   wget https://raw.githubusercontent.com/yockii/dify_tools/main/config.example.yaml -O config.yaml
   ```

3. 修改配置文件后启动服务：
   ```bash
   chmod +x dify-tools-linux-amd64
   ./dify-tools-linux-amd64
   ```

### 3. 源码部署

```bash
# 克隆仓库
git clone https://github.com/yockii/dify_tools.git
cd dify_tools

# 安装依赖
go mod tidy

# 编译
go build -o dify-tools ./cmd/server

# 准备配置文件
cp config.example.yaml config.yaml

# 启动服务
./dify-tools
```

## 配置说明

### 数据库配置

```yaml
database:
  type: mysql  # mysql 或 postgres
  host: localhost
  port: 3306
  user: root
  password: root
  dbname: dify_tools
  max_idle_conns: 10
  max_open_conns: 100
  conn_max_lifetime: 3600
```

### Redis配置

```yaml
cache:
  type: redis
  redis:
    host: localhost
    port: 6379
    password: redis123
    db: 0
```

### 监控配置

```yaml
monitor:
  slow_query:
    enabled: true
    threshold: 1000  # 慢查询阈值（毫秒）
    notify:
      type: email    # 告警通知方式
      target: admin@example.com
```

### 邮件配置

```yaml
email:
  enabled: true
  host: smtp.example.com
  port: 587
  from: noreply@example.com
  password: your-password
```

## 系统要求

- CPU: 2核心及以上
- 内存: 4GB及以上
- 磁盘: 20GB及以上
- 操作系统: Linux、MacOS或Windows
- 数据库: MySQL 8.0+ 或 PostgreSQL 14+
- Redis: 7.0+

## 安全建议

1. 修改默认密码
   - MySQL密码
   - Redis密码
   - 应用密钥

2. 配置防火墙
   ```bash
   # 只允许必要端口
   sudo ufw allow 8080/tcp
   sudo ufw enable
   ```

3. 配置SSL证书
   建议使用Nginx反向代理并配置SSL：

   ```nginx
   server {
       listen 443 ssl;
       server_name your-domain.com;

       ssl_certificate /path/to/cert.pem;
       ssl_certificate_key /path/to/key.pem;

       location / {
           proxy_pass http://localhost:8080;
           proxy_set_header Host $host;
           proxy_set_header X-Real-IP $remote_addr;
       }
   }
   ```

## 监控和日志

1. 日志位置
   - 应用日志: ./logs/app.log
   - 访问日志: ./logs/access.log

2. 监控指标
   - 访问 http://localhost:8080/metrics 查看Prometheus格式的监控指标

## 常见问题

1. 数据库连接失败
   - 检查数据库配置信息
   - 确认数据库服务状态
   - 检查防火墙设置

2. Redis连接失败
   - 验证Redis密码配置
   - 确认Redis服务状态
   - 检查网络连接

3. 邮件发送失败
   - 检查SMTP配置
   - 验证邮箱账号密码
   - 确认网络连接

## 维护指南

1. 数据库备份
   ```bash
   # MySQL备份
   mysqldump -u root -p dify_tools > backup.sql

   # PostgreSQL备份
   pg_dump -U postgres dify_tools > backup.sql
   ```

2. 日志清理
   ```bash
   # 清理30天前的日志
   find ./logs -name "*.log" -mtime +30 -delete
   ```

3. 版本升级
   ```bash
   # Docker部署
   docker-compose pull
   docker-compose up -d

   # 二进制部署
   wget https://github.com/yockii/dify_tools/releases/latest/download/dify-tools-linux-amd64
   systemctl restart dify-tools