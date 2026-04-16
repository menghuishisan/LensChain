package framework

import "encoding/json"

// jsonUnmarshal 包装标准库反序列化，供运行时统一调用。
func jsonUnmarshal(raw []byte, target any) error {
	return json.Unmarshal(raw, target)
}
