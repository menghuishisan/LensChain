package catalog

import (
	"fmt"
	"slices"
	"strings"

	"github.com/lenschain/sim-engine/scenarios/internal/framework"
)

// Registry 管理全部内置场景定义。
type Registry struct {
	definitions map[string]framework.Definition
}

// NewRegistry 创建并装载平台内置的 43 个场景定义。
func NewRegistry() *Registry {
	definitions := make([]framework.Definition, 0, 43)
	definitions = append(definitions, nodeNetworkDefinitions()...)
	definitions = append(definitions, consensusDefinitions()...)
	definitions = append(definitions, cryptographyDefinitions()...)
	definitions = append(definitions, dataStructureDefinitions()...)
	definitions = append(definitions, transactionDefinitions()...)
	definitions = append(definitions, smartContractDefinitions()...)
	definitions = append(definitions, attackSecurityDefinitions()...)
	definitions = append(definitions, economicDefinitions()...)

	index := make(map[string]framework.Definition, len(definitions))
	for _, definition := range definitions {
		index[definition.Code] = definition
	}
	return &Registry{definitions: index}
}

// Get 根据场景编码读取定义。
func (r *Registry) Get(code string) (framework.Definition, error) {
	definition, ok := r.definitions[strings.TrimSpace(code)]
	if !ok {
		return framework.Definition{}, fmt.Errorf("未注册内置场景: %s", code)
	}
	return definition, nil
}

// List 返回全部内置场景定义。
func (r *Registry) List() []framework.Definition {
	result := make([]framework.Definition, 0, len(r.definitions))
	for _, definition := range r.definitions {
		result = append(result, definition)
	}
	slices.SortFunc(result, func(left framework.Definition, right framework.Definition) int {
		return strings.Compare(left.Code, right.Code)
	})
	return result
}
