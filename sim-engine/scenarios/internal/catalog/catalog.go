// 模块：sim-engine/scenarios/catalog
// 文件职责：43 内置场景注册表。聚合各场景包的 Definition()，提供查询与一致性校验。
// 协议依据：docs/modules/04-实验环境/06-可视化仿真引擎设计.md §四 / §3.5 / §8.2。
//
// 职责约束（详 sim-engine/AGENTS.md §六）：
//   - 本目录仅做"元信息聚合"，不实现任何场景算法；
//   - 元信息必须从场景包 Definition() 读取，不在此处硬编码；
//   - 每个 8 类目对应一个 *_definitions.go 文件，仅 import 该类目下场景子包并收集 Definition；
//   - 注册时执行 schema 校验：场景 code 唯一、category 9 选 1、time/data 模式合法、声明 LinkGroup 时必须实现联动调用。

package catalog

import (
	"fmt"
	"slices"
	"strings"

	"github.com/lenschain/sim-engine/framework"
)

// Registry 管理全部内置场景定义。
type Registry struct {
	definitions map[string]framework.Definition
}

// NewRegistry 创建并装载平台内置的 43 个场景定义。
//
// 单例化由调用方控制（launcher 通常每次启动构建一次即可）。
func NewRegistry() *Registry {
	all := collectAll()
	index := make(map[string]framework.Definition, len(all))
	for _, def := range all {
		validateDefinition(def)
		if _, exists := index[def.Code]; exists {
			panic(fmt.Sprintf("场景 code 重复注册: %s", def.Code))
		}
		index[def.Code] = def
	}
	return &Registry{definitions: index}
}

// Get 根据场景编码读取定义。
func (r *Registry) Get(code string) (framework.Definition, error) {
	def, ok := r.definitions[strings.TrimSpace(code)]
	if !ok {
		return framework.Definition{}, fmt.Errorf("未注册内置场景: %s", code)
	}
	return def, nil
}

// List 返回全部内置场景定义（按 code 升序）。
func (r *Registry) List() []framework.Definition {
	result := make([]framework.Definition, 0, len(r.definitions))
	for _, def := range r.definitions {
		result = append(result, def)
	}
	slices.SortFunc(result, func(left framework.Definition, right framework.Definition) int {
		return strings.Compare(left.Code, right.Code)
	})
	return result
}

// Codes 返回所有已注册场景 code（升序）。
func (r *Registry) Codes() []string {
	codes := make([]string, 0, len(r.definitions))
	for code := range r.definitions {
		codes = append(codes, code)
	}
	slices.Sort(codes)
	return codes
}

// validateDefinition 强制检查场景 Definition 的完整性与协议合规性。
//
// 校验项（与 06.md / sim-engine/AGENTS.md §十 协议一致性规范对齐）：
//   - 必填元信息 / 钩子非空；
//   - Category 必须在 9 选 1 范围内；
//   - TimeControlMode / DataSourceMode 合法；
//   - 声明了 SupportedLinkGroups 时，每项 code 不能为空。
func validateDefinition(def framework.Definition) {
	if strings.TrimSpace(def.Code) == "" {
		panic("场景 code 不能为空")
	}
	if strings.TrimSpace(def.Name) == "" {
		panic(fmt.Sprintf("场景 %s name 不能为空", def.Code))
	}
	if strings.TrimSpace(def.AlgorithmType) == "" {
		panic(fmt.Sprintf("场景 %s algorithm_type 不能为空", def.Code))
	}
	if strings.TrimSpace(def.Version) == "" {
		panic(fmt.Sprintf("场景 %s version 不能为空", def.Code))
	}
	switch def.Category {
	case framework.CategoryNodeNetwork, framework.CategoryConsensus, framework.CategoryCryptography,
		framework.CategoryDataStructure, framework.CategoryTransaction, framework.CategorySmartContract,
		framework.CategoryAttackSecurity, framework.CategoryEconomic, framework.CategoryGeneric:
	default:
		panic(fmt.Sprintf("场景 %s category 非法: %s", def.Code, def.Category))
	}
	switch def.TimeControlMode {
	case framework.TimeControlProcess, framework.TimeControlReactive, framework.TimeControlContinuous:
	default:
		panic(fmt.Sprintf("场景 %s time_control_mode 非法: %s", def.Code, def.TimeControlMode))
	}
	switch def.DataSourceMode {
	case framework.DataSourceSimulation, framework.DataSourceCollection, framework.DataSourceDual:
	default:
		panic(fmt.Sprintf("场景 %s data_source_mode 非法: %s", def.Code, def.DataSourceMode))
	}
	if def.Init == nil {
		panic(fmt.Sprintf("场景 %s 缺少 Init", def.Code))
	}
	if def.Step == nil {
		panic(fmt.Sprintf("场景 %s 缺少 Step", def.Code))
	}
	if def.HandleAction == nil {
		panic(fmt.Sprintf("场景 %s 缺少 HandleAction", def.Code))
	}
	if def.Interaction == nil {
		panic(fmt.Sprintf("场景 %s 缺少 Interaction provider", def.Code))
	}
	if def.DefaultState == nil {
		panic(fmt.Sprintf("场景 %s 缺少 DefaultState", def.Code))
	}
	for _, group := range def.SupportedLinkGroups {
		if strings.TrimSpace(group) == "" {
			panic(fmt.Sprintf("场景 %s 联动组 code 不能为空", def.Code))
		}
	}
}
