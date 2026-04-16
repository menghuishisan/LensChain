// adapter.go
// SimEngine Collector 生态适配器注册表
// 负责维护文档要求的内置链生态适配器标准编码，避免在多处重复写生态校验逻辑。

package collector

import "strings"

// Adapter 描述一种链生态采集适配器。
type Adapter struct {
	Code          string
	Name          string
	CollectMethod string
	DataTypes     []string
}

// AdapterRegistry 管理内置和扩展采集适配器。
type AdapterRegistry struct {
	adapters map[string]Adapter
}

// NewAdapterRegistry 创建采集适配器注册表。
func NewAdapterRegistry() *AdapterRegistry {
	return &AdapterRegistry{
		adapters: make(map[string]Adapter),
	}
}

// Register 注册一个采集适配器标准编码。
func (r *AdapterRegistry) Register(adapter Adapter) {
	code := normalizeAdapterKey(adapter.Code)
	if code == "" {
		return
	}
	adapter.Code = code
	r.adapters[code] = adapter
}

// Get 根据标准编码读取适配器。
func (r *AdapterRegistry) Get(code string) (Adapter, bool) {
	adapter, ok := r.adapters[normalizeAdapterKey(code)]
	return adapter, ok
}

// Resolve 根据标准编码解析适配器。
func (r *AdapterRegistry) Resolve(input string) (Adapter, bool) {
	return r.Get(input)
}

var builtInAdapters = []Adapter{
	{
		Code:          "ethereum",
		Name:          "Ethereum (Geth)",
		CollectMethod: "JSON-RPC + WebSocket",
		DataTypes:     []string{"block", "transaction", "peer", "contract_event"},
	},
	{
		Code:          "fabric",
		Name:          "Hyperledger Fabric",
		CollectMethod: "gRPC EventHub",
		DataTypes:     []string{"block", "transaction", "chaincode_event", "channel_state"},
	},
	{
		Code:          "chainmaker",
		Name:          "ChainMaker",
		CollectMethod: "SDK subscribe",
		DataTypes:     []string{"block", "transaction", "contract_event"},
	},
	{
		Code:          "fisco",
		Name:          "FISCO BCOS",
		CollectMethod: "JSON-RPC + AMOP",
		DataTypes:     []string{"block", "transaction", "consensus", "contract_event"},
	},
}

var defaultAdapterRegistry = func() *AdapterRegistry {
	registry := NewAdapterRegistry()
	RegisterBuiltInAdapters(registry)
	return registry
}()

// RegisterBuiltInAdapters 将文档要求的四类生态适配器注册到指定注册表。
func RegisterBuiltInAdapters(registry *AdapterRegistry) {
	if registry == nil {
		return
	}
	for _, adapter := range builtInAdapters {
		registry.Register(adapter)
	}
}

// ResolveAdapterCode 校验并返回标准适配器编码。
func ResolveAdapterCode(input string) string {
	adapter, ok := defaultAdapterRegistry.Resolve(input)
	if !ok {
		return ""
	}
	return adapter.Code
}

// normalizeAdapterKey 统一适配器编码大小写和空白。
func normalizeAdapterKey(input string) string {
	return strings.ToLower(strings.TrimSpace(input))
}
