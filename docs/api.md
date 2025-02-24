# API 文档

## 认证

所有API请求（除登录接口外）都需要在请求头中携带Token：

```
Authorization: Bearer <token>
```

### 获取Token

```http
POST /api/v1/auth/login
Content-Type: application/json

{
    "username": "admin",
    "password": "admin123"
}
```

响应：

```json
{
    "code": 200,
    "message": "success",
    "data": {
        "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
    }
}
```

### 刷新Token

```http
POST /api/v1/auth/refresh
Authorization: Bearer <old-token>
```

响应：

```json
{
    "code": 200,
    "message": "success",
    "data": {
        "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
    }
}
```

## 用户管理

### 获取用户列表

```http
GET /api/v1/users?offset=0&limit=10
```

查询参数：
- offset: 偏移量
- limit: 每页数量，默认10，最大100

响应：

```json
{
    "code": 200,
    "message": "success",
    "data": {
        "total": 100,
        "items": [
            {
                "id": "uuid",
                "username": "admin",
                "roleId": "role-uuid",
                "role": {
                    "id": "role-uuid",
                    "name": "管理员",
                    "code": "ADMIN",
                    "permissions": ["SYSTEM", "SYSTEM_USER", ...]
                },
                "status": 1,
                "lastLogin": "2025-02-17T10:00:00Z",
                "createdAt": "2025-02-17T10:00:00Z",
                "updatedAt": "2025-02-17T10:00:00Z"
            }
        ],
        "offset": 0,
        "limit": 10
    }
}
```

### 创建用户

```http
POST /api/v1/users
Content-Type: application/json

{
    "username": "user1",
    "password": "password123",
    "roleId": "role-uuid"
}
```

## 应用管理

### 获取应用列表

```http
GET /api/v1/apps?offset=0&limit=10&keyword=test
```

查询参数：
- offset: 偏移量
- limit: 每页数量
- keyword: 搜索关键字

### 创建应用

```http
POST /api/v1/apps
Content-Type: application/json

{
    "name": "测试应用",
    "description": "这是一个测试应用",
    "config": {
        "aiModels": ["gpt-3.5-turbo"],
        "maxTokens": 4000,
        "rateLimit": {
            "enabled": true,
            "maxRequests": 1000,
            "duration": 3600
        },
        "allowedOrigins": ["*"]
    }
}
```

## 数据源管理

### 获取数据源列表

```http
GET /api/v1/apps/{appId}/datasources?offset=0&limit=10
```

### 创建数据源

```http
POST /api/v1/apps/{appId}/datasources
Content-Type: application/json

{
    "name": "主数据库",
    "type": "postgres",
    "config": {
        "host": "localhost",
        "port": 5432,
        "user": "postgres",
        "password": "postgres",
        "database": "test"
    }
}
```

### 同步数据源Schema

```http
POST /api/v1/apps/{appId}/datasources/{id}/sync
```

### 执行SQL查询

```http
POST /api/v1/apps/{appId}/datasources/{id}/query
Content-Type: application/json

{
    "sql": "SELECT * FROM users LIMIT 10"
}
```

## 使用统计

### 获取应用使用统计

```http
GET /api/v1/apps/{id}/usage?startDate=2025-02-01&endDate=2025-02-17
```

### 获取排行榜

```http
GET /api/v1/statistics/top-apps?limit=10
```

## 错误码

| 错误码 | 说明 |
|--------|------|
| 200 | 成功 |
| 400 | 请求参数错误 |
| 401 | 未认证或认证失败 |
| 403 | 权限不足 |
| 404 | 资源不存在 |
| 500 | 服务器内部错误 |

## 常见问题

### 如何处理Token过期？

当Token过期时，API会返回401状态码。此时应该：

1. 调用刷新Token接口获取新Token
2. 使用新Token重试之前的请求

如果刷新Token也失败，则需要重新登录。

### 如何处理限流？

当请求被限流时，API会返回429状态码。建议：

1. 实现指数退避重试
2. 合理设置限流参数
3. 在客户端实现本地限流