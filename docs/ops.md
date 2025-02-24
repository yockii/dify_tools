# 部署和运维指南

## 系统要求

### 最低配置
- CPU: 2核
- 内存: 4GB
- 磁盘: 20GB
- 操作系统: Ubuntu 20.04+ / CentOS 7+

### 推荐配置
- CPU: 4核
- 内存: 8GB
- 磁盘: 50GB SSD
- 操作系统: Ubuntu 22.04 LTS

## 环境准备

### 安装依赖

```bash
# Ubuntu
apt update
apt install -y postgresql-14 nodejs npm nginx

# CentOS
yum install -y postgresql14-server nodejs nginx
```

### 初始化数据库

```bash
# 初始化数据库
sudo -u postgres psql -f configs/init.sql

# 创建数据库用户
sudo -u postgres psql
postgres=# CREATE USER dify WITH PASSWORD 'your-password';
postgres=# GRANT ALL PRIVILEGES ON DATABASE dify_tools TO dify;
```

## 部署步骤

### 1. 编译

```bash
# 编译后端
go build -o dify-tools cmd/server/main.go

# 编译前端
cd web/dify-tools-web
npm install
npm run build
```

### 2. 配置文件

```bash
# 复制并修改配置文件
cp configs/config.yaml.example /etc/dify-tools/config.yaml
vim /etc/dify-tools/config.yaml
```

### 3. 系统服务

创建服务文件：

```bash
cat > /etc/systemd/system/dify-tools.service << 'EOF'
[Unit]
Description=Dify Tools Server
After=network.target

[Service]
Type=simple
User=dify
Group=dify
WorkingDirectory=/opt/dify-tools
ExecStart=/opt/dify-tools/dify-tools --config /etc/dify-tools/config.yaml
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
EOF
```

### 4. Nginx 配置

```nginx
server {
    listen 80;
    server_name tools.example.com;

    # SSL配置（推荐）
    listen 443 ssl;
    ssl_certificate /etc/nginx/certs/cert.pem;
    ssl_certificate_key /etc/nginx/certs/key.pem;

    # HTTP 跳转 HTTPS
    if ($scheme != "https") {
        return 301 https://$server_name$request_uri;
    }

    # 前端文件
    location / {
        root /opt/dify-tools/web/dist;
        try_files $uri $uri/ /index.html;
    }

    # API代理
    location /api/ {
        proxy_pass http://localhost:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

### 5. 启动服务

```bash
# 重新加载系统服务
systemctl daemon-reload

# 启动服务
systemctl start dify-tools

# 设置开机启动
systemctl enable dify-tools
```

## 监控

### 日志监控

日志文件位置：
- 应用日志：`/var/log/dify-tools/app.log`
- Nginx访问日志：`/var/log/nginx/dify-tools.access.log`
- Nginx错误日志：`/var/log/nginx/dify-tools.error.log`

推荐使用 ELK Stack 或 Loki 进行日志收集和分析。

### 性能监控

推荐使用 Prometheus + Grafana 进行监控，主要监控指标：

1. 系统指标
   - CPU使用率
   - 内存使用率
   - 磁盘使用率
   - 网络流量

2. 应用指标
   - QPS
   - 响应时间
   - 错误率
   - 并发连接数

3. 数据库指标
   - 连接数
   - 查询性能
   - 磁盘使用

## 备份

### 数据库备份

创建定时备份脚本：

```bash
#!/bin/bash

BACKUP_DIR="/backup/dify-tools/db"
DATE=$(date +%Y%m%d_%H%M%S)
BACKUP_FILE="$BACKUP_DIR/dify_tools_$DATE.sql"

# 创建备份目录
mkdir -p $BACKUP_DIR

# 备份数据库
pg_dump -U dify dify_tools > $BACKUP_FILE

# 压缩备份文件
gzip $BACKUP_FILE

# 删除7天前的备份
find $BACKUP_DIR -name "*.sql.gz" -mtime +7 -delete
```

配置定时任务：

```bash
# 编辑crontab
crontab -e

# 添加每日备份任务（每天凌晨2点执行）
0 2 * * * /opt/dify-tools/scripts/backup.sh
```

## 问题排查

### 1. 服务无法启动

检查项：
1. 配置文件权限和内容
2. 日志文件权限
3. 数据库连接
4. 端口占用

### 2. 系统响应慢

检查项：
1. 数据库性能（慢查询）
2. 系统资源使用情况
3. 网络连接状态
4. 日志中的错误信息

### 3. 数据库连接问题

检查项：
1. PostgreSQL服务状态
2. 连接配置
3. 防火墙规则
4. 最大连接数设置

## 升级指南

### 1. 准备工作

1. 备份数据库
2. 备份配置文件
3. 通知用户系统维护

### 2. 升级步骤

```bash
# 停止服务
systemctl stop dify-tools

# 备份当前版本
mv /opt/dify-tools /opt/dify-tools.bak

# 部署新版本
# ... (按部署步骤执行)

# 恢复配置
cp /opt/dify-tools.bak/config.yaml /etc/dify-tools/

# 启动服务
systemctl start dify-tools

# 验证服务状态
systemctl status dify-tools
curl http://localhost:8080/health
```

### 3. 回滚方案

如果升级失败，执行以下步骤回滚：

```bash
# 停止新版本
systemctl stop dify-tools

# 恢复旧版本
rm -rf /opt/dify-tools
mv /opt/dify-tools.bak /opt/dify-tools

# 启动服务
systemctl start dify-tools
```

## 安全建议

1. 配置SSL证书
2. 启用防火墙
3. 定期更新系统和依赖
4. 使用强密码
5. 限制数据库访问
6. 配置请求限流
7. 启用审计日志