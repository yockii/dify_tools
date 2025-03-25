package model

import (
	"fmt"
	"strings"
	"time"

	"github.com/yockii/dify_tools/internal/constant"
	"github.com/yockii/dify_tools/pkg/config"
	"github.com/yockii/dify_tools/pkg/logger"
	"gorm.io/gorm"
)

type Model interface {
	TableComment() string
	GetID() uint64
}

type BaseModel struct {
	ID        uint64    `json:"id,string" gorm:"primaryKey"`
	CreatedAt time.Time `json:"createdAt,omitzero" gorm:"type:timestamp;not null"`
}

func (b *BaseModel) TableComment() string {
	return "基础模型"
}

func (b *BaseModel) GetID() uint64 {
	return b.ID
}

var models []Model

func AutoMigrate(db *gorm.DB) {
	if dt := config.GetString("database.type"); dt == "mysql" {
		migrator := db.Migrator()
		for _, m := range models {
			if !migrator.HasTable(m) {
				if err := db.Set("gorm:table_options", fmt.Sprintf("ENGINE=innoDB DEFAULT CHARSET=utf8mb4 COMMENT='%s';", m.TableComment())).AutoMigrate(m); err != nil {
					logger.Error("自动迁移表失败", logger.F("error", err))
				}
			} else {
				_ = migrator.AutoMigrate(m)
			}
		}
	} else if dt == "postgres" {
		var mList []interface{}
		for _, m := range models {
			mList = append(mList, m)
		}
		if err := db.AutoMigrate(mList...); err != nil {
			logger.Error("自动迁移表失败", logger.F("error", err))
		}
		// 添加表注释
		for _, m := range models {
			stmt := &gorm.Statement{DB: db}
			if err := stmt.Parse(m); err != nil {
				logger.Error("解析模型失败", logger.F("error", err))
				continue
			}
			tableName := stmt.Table
			if err := db.Exec(fmt.Sprintf("COMMENT ON TABLE %s IS '%s';", tableName, m.TableComment())).Error; err != nil {
				logger.Error("添加表注释失败", logger.F("error", err))
			}
		}
	} else {
		logger.Error("不支持的数据库类型", logger.F("type", dt))
	}
}

func InitData(db *gorm.DB) error {
	// 使用事务确保数据一致性
	return db.Transaction(func(tx *gorm.DB) error {
		// 初始化系统权限
		permissions := []*Permission{
			// 系统管理
			{Code: "system", Name: "系统管理", Type: "menu"},
			// 用户管理
			{Code: "system:user", Name: "用户管理", Type: "menu", ParentID: 0},
			{Code: "system:user:view", Name: "查看用户", Type: "button", ParentID: 0},
			{Code: "system:user:create", Name: "创建用户", Type: "button", ParentID: 0},
			{Code: "system:user:edit", Name: "编辑用户", Type: "button", ParentID: 0},
			{Code: "system:user:delete", Name: "删除用户", Type: "button", ParentID: 0},
			// 角色管理
			{Code: "system:role", Name: "角色管理", Type: "menu", ParentID: 0},
			{Code: "system:role:view", Name: "查看角色", Type: "button", ParentID: 0},
			{Code: "system:role:create", Name: "创建角色", Type: "button", ParentID: 0},
			{Code: "system:role:edit", Name: "编辑角色", Type: "button", ParentID: 0},
			{Code: "system:role:delete", Name: "删除角色", Type: "button", ParentID: 0},
			// 应用管理
			{Code: "app", Name: "应用管理", Type: "menu"},
			{Code: "app:view", Name: "查看应用", Type: "button", ParentID: 0},
			{Code: "app:create", Name: "创建应用", Type: "button", ParentID: 0},
			{Code: "app:edit", Name: "编辑应用", Type: "button", ParentID: 0},
			{Code: "app:delete", Name: "删除应用", Type: "button", ParentID: 0},
			// 数据源管理
			{Code: "app:datasource", Name: "数据源管理", Type: "menu", ParentID: 0},
			{Code: "app:datasource:view", Name: "查看数据源", Type: "button", ParentID: 0},
			{Code: "app:datasource:create", Name: "创建数据源", Type: "button", ParentID: 0},
			{Code: "app:datasource:edit", Name: "编辑数据源", Type: "button", ParentID: 0},
			{Code: "app:datasource:delete", Name: "删除数据源", Type: "button", ParentID: 0},
		}

		// 存储所有权限的map，用于后续关联，以 Code 为键
		permMap := make(map[string]*Permission)

		// 第一轮：创建所有权限记录，获取ID
		for _, perm := range permissions {
			var existPerm Permission
			if err := tx.Where(&Permission{Code: perm.Code}).Assign(perm).FirstOrCreate(&existPerm).Error; err != nil {
				return fmt.Errorf("create permission failed: %v", err)
			}
			permMap[perm.Code] = &existPerm
		}

		// 第二轮：更新父级ID
		for _, perm := range permissions {
			if !strings.Contains(perm.Code, ":") {
				continue // 跳过顶级权限
			}

			parts := strings.Split(perm.Code, ":")
			parentCode := parts[0] // 例如 system:user:view 的父级是 system:user
			if len(parts) > 2 {
				parentCode = strings.Join(parts[:len(parts)-1], ":")
			}

			if parent, exists := permMap[parentCode]; exists {
				permInMap := permMap[perm.Code]
				permInMap.ParentID = parent.ID
				if err := tx.Save(permInMap).Error; err != nil {
					return fmt.Errorf("update permission parent id failed: %v", err)
				}
			}
		}

		// 初始化管理员角色
		adminRole := &Role{
			Name:   "系统管理员",
			Code:   "ADMIN",
			Status: 1,
		}

		// 查询是否已存在管理员角色
		if err := tx.Where(&Role{Code: adminRole.Code}).Attrs(adminRole).FirstOrCreate(adminRole).Error; err != nil {
			return fmt.Errorf("create admin role failed: %v", err)
		}

		// 为管理员角色分配所有权限
		for _, perm := range permMap {
			rolePermission := &RolePermission{
				RoleID:       adminRole.ID,
				PermissionID: perm.ID,
			}
			// 检查是否存在
			if err := tx.Where("role_id = ? AND permission_id = ?", rolePermission.RoleID, rolePermission.PermissionID).First(&RolePermission{}).Error; err != nil {
				if err == gorm.ErrRecordNotFound {
					// 不存在则创建
					// 使用 Create 而不是 FirstOrCreate，因为这是复合主键
					if err := tx.Create(rolePermission).Error; err != nil {
						// 忽略唯一键冲突错误，说明记录已存在
						if !strings.Contains(err.Error(), "duplicate key") {
							return fmt.Errorf("create role permission failed: %v", err)
						}
					}
				}
			}
		}

		// 初始化管理员用户
		adminUser := &User{
			Username: config.GetString("admin.username"),
			Password: config.GetString("admin.password"),
			RoleID:   adminRole.ID,
			Status:   1,
		}

		// 查询是否已存在管理员用户
		if err := tx.Where(&User{Username: adminUser.Username}).Attrs(adminUser).FirstOrCreate(adminUser).Error; err != nil {
			return fmt.Errorf("create admin user failed: %v", err)
		}

		// 增加字典数据
		{
			// 日志类型
			logTypeDict := &Dict{
				Name:     "日志类型",
				Code:     constant.DictTypeCodeLogType,
				Value:    "日志类型",
				ParentID: 0,
				Sort:     0,
			}
			if err := tx.Model(&Dict{}).Where(&Dict{
				Code: logTypeDict.Code,
			}).Attrs(logTypeDict).FirstOrCreate(logTypeDict).Error; err != nil {
				return fmt.Errorf("create dict failed: %v", err)
			}
			{
				logMap := []struct {
					Name  string
					Value int
					Code  string
				}{
					{Name: "登录", Value: 1, Code: "log_action_login"},
					{Name: "登出", Value: 2, Code: "log_action_logout"},
					{Name: "创建用户", Value: 11, Code: "log_action_create_user"},
					{Name: "编辑用户", Value: 12, Code: "log_action_update_user"},
					{Name: "删除用户", Value: 13, Code: "log_action_delete_user"},
					{Name: "修改密码", Value: 14, Code: "log_action_update_password"},
					{Name: "修改用户状态", Value: 15, Code: "log_action_update_user_status"},
					{Name: "创建角色", Value: 21, Code: "log_action_create_role"},
					{Name: "编辑角色", Value: 22, Code: "log_action_update_role"},
					{Name: "删除角色", Value: 23, Code: "log_action_delete_role"},
					{Name: "创建应用", Value: 31, Code: "log_action_create_application"},
					{Name: "编辑应用", Value: 32, Code: "log_action_update_application"},
					{Name: "删除应用", Value: 33, Code: "log_action_delete_application"},
					{Name: "修改应用配置", Value: 34, Code: "log_action_update_application_config"},
					{Name: "创建数据源", Value: 41, Code: "log_action_create_data_source"},
					{Name: "编辑数据源", Value: 42, Code: "log_action_update_data_source"},
					{Name: "删除数据源", Value: 43, Code: "log_action_delete_data_source"},
					{Name: "同步数据源", Value: 44, Code: "log_action_sync_data_source"},
					{Name: "修改表信息", Value: 45, Code: "log_action_update_table_info"},
					{Name: "修改列信息", Value: 46, Code: "log_action_update_column_info"},
					{Name: "删除表信息", Value: 47, Code: "log_action_delete_table_info"},
					{Name: "删除列信息", Value: 48, Code: "log_action_delete_column_info"},
					{Name: "创建字典", Value: 51, Code: "log_action_create_dict"},
					{Name: "编辑字典", Value: 52, Code: "log_action_update_dict"},
					{Name: "删除字典", Value: 53, Code: "log_action_delete_dict"},
				}
				for _, log := range logMap {
					if err := tx.Where(&Dict{
						Code:     log.Code,
						ParentID: logTypeDict.ID,
					}).Attrs(&Dict{
						Name:  log.Name,
						Value: fmt.Sprintf("%d", log.Value),
						Sort:  0,
					}).FirstOrCreate(&Dict{}).Error; err != nil {
						return fmt.Errorf("create dict failed: %v", err)
					}
				}
			}

			// 数据源类型
			datasourceTypeDict := &Dict{
				Name:     "数据源类型",
				Code:     constant.DictTypeCodeDatasourceType,
				Value:    "数据源类型",
				ParentID: 0,
				Sort:     0,
			}
			if err := tx.Model(&Dict{}).Where(&Dict{
				Code: datasourceTypeDict.Code,
			}).Attrs(datasourceTypeDict).FirstOrCreate(datasourceTypeDict).Error; err != nil {
				return fmt.Errorf("create dict failed: %v", err)
			}
			{
				datasourceMap := []struct {
					Name  string
					Value string
					Code  string
				}{
					{Name: "MySQL", Value: "mysql", Code: "datasource_type_mysql"},
					{Name: "PostgreSQL", Value: "postgres", Code: "datasource_type_postgresql"},
				}
				for _, ds := range datasourceMap {
					if err := tx.Where(&Dict{
						Code:     ds.Code,
						ParentID: datasourceTypeDict.ID,
					}).Attrs(&Dict{
						Name:  ds.Name,
						Value: ds.Value,
						Sort:  0,
					}).FirstOrCreate(&Dict{}).Error; err != nil {
						return fmt.Errorf("create dict failed: %v", err)
					}
				}
			}

			difyDict := &Dict{
				Name:     "Dify配置",
				Code:     constant.DictTypeCodeDify,
				Value:    "Dify配置",
				ParentID: 0,
				Sort:     0,
			}
			if err := tx.Model(&Dict{}).Where(&Dict{
				Code: difyDict.Code,
			}).Attrs(difyDict).FirstOrCreate(difyDict).Error; err != nil {
				return fmt.Errorf("create dict failed: %v", err)
			}

			{

				if err := tx.Where(&Dict{
					Code:     constant.DictCodeDifyBaseUrl,
					ParentID: difyDict.ID,
				}).Attrs(&Dict{
					Name:  "DIFY基础URL",
					Value: "http://localhost:80/v1",
					Sort:  0,
				}).FirstOrCreate(&Dict{}).Error; err != nil {
					return fmt.Errorf("create dict failed: %v", err)
				}
				if err := tx.Where(&Dict{
					Code:     constant.DictCodeDifyToken,
					ParentID: difyDict.ID,
				}).Attrs(&Dict{
					Name: "DIFY密钥",
					Sort: 1,
				}).FirstOrCreate(&Dict{}).Error; err != nil {
					return fmt.Errorf("create dict failed: %v", err)
				}
			}

			// 内嵌智能体数据初始化
			{
				commonFlowAgent := &Agent{}
				if err := tx.Where(&Agent{
					Code: InnerAgentCodeCommonChatFlow,
					Type: 1,
				}).Attrs(&Agent{
					Name: InnerAgentNameCommonChatFlow,
				}).FirstOrCreate(commonFlowAgent).Error; err != nil {
					return fmt.Errorf("create agent failed: %v", err)
				}

				// 增加字典数据-默认智能体ID
				if err := tx.Model(&Dict{}).Where(&Dict{
					Code: constant.DictCodeDifyDefaultAgentID,
				}).Assign(&Dict{
					Name:     "默认智能体ID",
					Value:    fmt.Sprintf("%d", commonFlowAgent.ID),
					ParentID: difyDict.ID,
				}).FirstOrCreate(&Dict{}).Error; err != nil {
					return fmt.Errorf("update dict failed: %v", err)
				}

				// 增加本系统的通用流程智能体
				if err := tx.Where(map[string]any{
					"agent_id":       commonFlowAgent.ID,
					"application_id": 0,
				}).FirstOrCreate(&ApplicationAgent{}).Error; err != nil {
					return fmt.Errorf("create application agent failed: %v", err)
				}
			}
		}

		return nil
	})
}
