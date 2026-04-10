// snowflake.go
// 雪花ID生成器
// 基于 bwmarrin/snowflake 封装，生成全局唯一的 BIGINT 主键
// 所有表的主键统一使用雪花ID，不使用自增

package snowflake

import (
	"fmt"
	"strconv"

	"github.com/bwmarrin/snowflake"
)

// 全局雪花节点
var node *snowflake.Node

// Init 初始化雪花ID生成器
// nodeID 为节点ID，集群部署时每个节点需要不同的ID（0-1023）
func Init(nodeID int64) error {
	var err error
	node, err = snowflake.NewNode(nodeID)
	if err != nil {
		return fmt.Errorf("初始化雪花ID生成器失败: %w", err)
	}
	return nil
}

// Generate 生成雪花ID（int64）
func Generate() int64 {
	return node.Generate().Int64()
}

// GenerateString 生成雪花ID字符串
// 雪花ID超过 JavaScript 安全整数范围，JSON传输时必须使用字符串
func GenerateString() string {
	return node.Generate().String()
}

// ParseString 将字符串ID解析为int64
func ParseString(id string) (int64, error) {
	return strconv.ParseInt(id, 10, 64)
}
