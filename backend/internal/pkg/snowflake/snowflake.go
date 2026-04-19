// snowflake.go
// 该文件封装平台统一使用的雪花 ID 生成器，为数据库主键和需要全局唯一编号的业务对象
// 提供稳定的 long integer ID。它的存在是为了让各模块不再自己生成主键或退回自增 ID。

package snowflake

import (
	"fmt"
	"strconv"

	"github.com/bwmarrin/snowflake"
)

// 全局雪花节点
var node *snowflake.Node

func init() {
	defaultNode, err := snowflake.NewNode(1)
	if err == nil {
		node = defaultNode
	}
}

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
	if node == nil {
		return 0
	}
	return node.Generate().Int64()
}

// GenerateString 生成雪花ID字符串
// 雪花ID超过 JavaScript 安全整数范围，JSON传输时必须使用字符串
func GenerateString() string {
	if node == nil {
		return ""
	}
	return node.Generate().String()
}

// ParseString 将字符串ID解析为int64
func ParseString(id string) (int64, error) {
	return strconv.ParseInt(id, 10, 64)
}
