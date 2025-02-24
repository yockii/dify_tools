package config

import (
	"fmt"
	"strings"
	"sync"

	"github.com/spf13/viper"
)

var (
	config *viper.Viper
	once   sync.Once
)

// Init 初始化配置
func Init(configFiles ...string) error {
	var err error
	once.Do(func() {
		config = viper.New()
		configFile := "config.yaml"
		if len(configFiles) > 0 {
			configFile = configFiles[0]
		}
		config.SetConfigFile(configFile)

		// 设置默认值
		setDefaults()

		// 读取配置文件
		if err = config.ReadInConfig(); err != nil {
			err = fmt.Errorf("read config file failed: %v", err)
			return
		}

		// 监听配置文件变化
		config.WatchConfig()
	})
	return err
}

// setDefaults 设置默认值
func setDefaults() {
	config.SetDefault("server.port", 8080)
	config.SetDefault("server.mode", "debug")

	config.SetDefault("jwt.secret", "your-jwt-secret-key")
	config.SetDefault("jwt.expire", 86400)

	config.SetDefault("database.host", "localhost")
	config.SetDefault("database.port", 5432)
	config.SetDefault("database.user", "postgres")
	config.SetDefault("database.password", "postgres")
	config.SetDefault("database.dbname", "dify_tools")
	config.SetDefault("database.max_idle_conns", 10)
	config.SetDefault("database.max_open_conns", 100)
	config.SetDefault("database.conn_max_lifetime", 3600)

	config.SetDefault("log.filename", "logs/app.log")
	config.SetDefault("log.max_size", 100)
	config.SetDefault("log.max_backups", 3)
	config.SetDefault("log.max_age", 28)
	config.SetDefault("log.compress", true)

	config.SetDefault("security.allowed_origins", "*")

	config.SetDefault("rate_limit.enabled", true)
	config.SetDefault("rate_limit.max_requests", 1000)
	config.SetDefault("rate_limit.duration", 3600)
}

// Get 获取配置值
func Get(key string) interface{} {
	return config.Get(key)
}

// GetString 获取字符串配置值
func GetString(key string) string {
	return config.GetString(key)
}

// GetInt 获取整数配置值
func GetInt(key string) int {
	return config.GetInt(key)
}

// GetInt64 获取64位整数配置值
func GetInt64(key string) int64 {
	return config.GetInt64(key)
}

// GetUint64 获取64位无符号整数配置值
func GetUint64(key string) uint64 {
	return config.GetUint64(key)
}

// GetFloat64 获取浮点数配置值
func GetFloat64(key string) float64 {
	return config.GetFloat64(key)
}

// GetBool 获取布尔配置值
func GetBool(key string) bool {
	return config.GetBool(key)
}

// GetStringSlice 获取字符串切片配置值
func GetStringSlice(key string) []string {
	return config.GetStringSlice(key)
}

// GetStringMapString 获取字符串映射配置值
func GetStringMapString(key string) map[string]string {
	return config.GetStringMapString(key)
}

// Set 设置配置值
func Set(key string, value interface{}) {
	config.Set(key, value)
}

// IsSet 检查配置值是否已设置
func IsSet(key string) bool {
	return config.IsSet(key)
}

// AllSettings 获取所有配置
func AllSettings() map[string]interface{} {
	return config.AllSettings()
}

// GetDSN 获取数据库连接字符串
func GetDSN() string {
	dbType := GetString("database.type")
	switch strings.ToLower(dbType) {
	case "postgres":
		return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
			GetString("database.host"),
			GetInt("database.port"),
			GetString("database.user"),
			GetString("database.password"),
			GetString("database.dbname"),
		)
	case "mysql":
		return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
			GetString("database.user"),
			GetString("database.password"),
			GetString("database.host"),
			GetInt("database.port"),
			GetString("database.dbname"),
		)
	default:
		return ""
	}
}

// GetJWTSecret 获取JWT密钥
func GetJWTSecret() []byte {
	return []byte(GetString("jwt.secret"))
}

// GetServerAddress 获取服务器地址
func GetServerAddress() string {
	return fmt.Sprintf(":%d", GetInt("server.port"))
}
