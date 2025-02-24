package util

import snowflake "github.com/yockii/snowflake_ext"

var idGenerator *snowflake.Worker

// InitNode 初始化ID生成器
func InitNode(nodeID uint64) error {
	var err error
	idGenerator, err = snowflake.NewSnowflake(nodeID)
	if err != nil {
		return err
	}
	return nil
}

// NewID 生成新的ID
func NewID() uint64 {
	return idGenerator.NextId()
}
