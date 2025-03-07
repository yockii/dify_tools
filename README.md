# Dify Tools

DIFY 工具集，提供：数据查询、文档处理及应用管理的能力。

由于dify本身的局限性，在我的实际应用场景中，希望能给一些应用赋予AI能力的时候，会遇到：
- 应用本身的数据查询整合
- 应用的公共知识库构建
- 应用的用户私人知识库
- 使用AI能力时，不可能为每个应用每个人去开发一套dify流程（因为dify目前的知识库在流程中直接引用对应的知识库）

本工具通过外部工具的方式解决以上问题，可以让第三方应用通过本系统接入dify，获得AI的能力

## 功能特性

- 数据源管理
  * 支持MySQL和PostgreSQL
  * 数据库连接管理
  * 表结构同步
  * SQL查询执行

- 知识库管理
  * 应用知识库
  * 应用用户私人知识库
  * dify调用

- 应用管理
  * 三方应用快速接入
  * 三方应用接口

- 权限管理
  * 基于角色的访问控制
  * 表级别白名单
  * 列级别数据脱敏
  * 权限继承关系

## 给应用的接口
1. 新增文档到知识库
2. 查询知识库文档状态（是否已经处理）
3. 删除知识库文档
3. 用户问答聊天及回复（流式）
4. 查询token使用量

## 快速开始

1. 启动本程序（当然要配置一下，相信使用者应该对配置不陌生）
本系统需要数据库及redis

2. 启动dify

3. 配置一下
dify中需要配置如下信息：
- 配置外部知识库API：进入知识库，右侧外部知识库API处，添加外部知识库API，在API endpoint处填入本系统dify的端点地址 `http://192.168.x.y:z/dify_api/v1`，apikey自定
- 知识库中选择API（左侧），并在右上角的API密钥中创建密钥，将密钥填入本系统（无界面的情况下，直接写入数据库对应字典值即可）
- 工具中，创建自定义工具，名称自定义（数据库检索查询的工具），schema填入(url根据自己的修改一下)：
```json
{
	"openapi": "3.1.0",
	"info": {
		"title": "应用数据查询",
		"description": "查询应用数据信息.",
		"version": "v1.0.0"
	},
	"servers": [{
		"url": "http://192.168.x.y:z/dify_api/v1"
	}],
	"paths": {
		"/databases": {
			"get": {
				"description": "获取应用所有数据源",
				"operationId": "GetDatabasesForApp",
				"parameters": [{
					"name": "Authorization",
					"in": "header",
					"description": "应用密钥",
					"required": true,
					"schema": {
						"type": "string"
					}
				}],
				"deprecated": false
			}
		},
		"/schema": {
			"get": {
				"description": "获取应用数据库结构信息",
				"operationId": "GetDatabaseSchema",
				"parameters": [{
					"name": "Authorization",
					"in": "header",
					"description": "应用密钥",
					"required": true,
					"schema": {
						"type": "string"
					}
				}, {
					"name": "datasourceId",
					"in": "query",
					"description": "数据源ID",
					"required": true,
					"schema": {
						"type": "string"
					}
				}],
				"deprecated": false
			}
		},
		"/executeSql": {
			"post": {
				"description": "执行SQL",
				"operationId": "executeSql",
				"parameters": [{
					"name": "Authorization",
					"in": "header",
					"description": "应用密钥",
					"required": true,
					"schema": {
						"type": "string"
					}
				}],
                "requestBody": {
                	"content": {
                    	"application/json": {
                        	"schema": {
                            	"type": "object",
                                "properties": {
                                	"sql": {
                                    	"type": "string",
                                        "description": "SQL语句"
                                    },
                                    "datasourceId": {
                                         "type": "string",
                                         "description": "数据源ID"
                                    }
                                }
                            }
                        }
                    }
                },
				"deprecated": false
			}
		}
	},
	"components": {
		"schemas": {}
	}
}
```
应该会自动识别可用工具，没问题直接保存即可
- 在工作室中创建自定义的会话工作流，在需要的地方引用外部知识库及对应工具即可
- 将对应工作流的API密钥（工作流编排下面的<访问API>，右上角有API密钥，创建后，填入系统对应的字典中。
- 注意填入dify调用地址字典值

4. 系统中创建应用，并根据系统接口开发应用接入，此时就可以让这个系统拥有dify的能力，并且扩展了可以查询知识库和应用对应数据了

TIPS：真的要查询应用的数据的话，还需要在系统中配置数据源，并同步数据结构，可以的话，看看表和字段的注释是否正确并修改。

## 相关界面
暂未放出，待完善后提供

## 贡献

欢迎提交Issue和Pull Request。

## 许可证

MIT License