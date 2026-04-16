package app

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// InteractionContext 表示一次多人协作交互附带的操作者上下文。
type InteractionContext struct {
	ActorID string
	RoleKey string
}

// collaborationPolicy 保存会话级多人协作权限规则。
type collaborationPolicy struct {
	enabled bool
	members map[string]string
	roles   map[string]collaborationRolePolicy
}

// collaborationRolePolicy 定义某个角色在各场景下的权限集合。
type collaborationRolePolicy struct {
	Scenes map[string]scenePermissionPolicy `json:"scenes"`
}

// scenePermissionPolicy 定义单个场景下允许的动作与资源范围。
type scenePermissionPolicy struct {
	Actions     []string `json:"actions"`
	ResourceIDs []string `json:"resource_ids"`
}

// collaborationSessionConfig 对应会话配置中的多人协作配置入口。
type collaborationSessionConfig struct {
	Collaboration collaborationPolicyConfig `json:"collaboration"`
}

// collaborationPolicyConfig 描述会话中的成员映射和角色权限表。
type collaborationPolicyConfig struct {
	Members []collaborationMemberConfig        `json:"members"`
	Roles   map[string]collaborationRolePolicy `json:"roles"`
}

// collaborationMemberConfig 描述单个协作成员与其角色绑定关系。
type collaborationMemberConfig struct {
	ActorID string `json:"actor_id"`
	RoleKey string `json:"role_key"`
}

// parseCollaborationPolicy 从会话配置中提取多人协作权限规则。
func parseCollaborationPolicy(sessionConfigJSON []byte) (collaborationPolicy, error) {
	if len(sessionConfigJSON) == 0 {
		return collaborationPolicy{}, nil
	}

	var config collaborationSessionConfig
	if err := json.Unmarshal(sessionConfigJSON, &config); err != nil {
		return collaborationPolicy{}, err
	}

	policy := collaborationPolicy{
		members: make(map[string]string, len(config.Collaboration.Members)),
		roles:   make(map[string]collaborationRolePolicy, len(config.Collaboration.Roles)),
	}
	for _, member := range config.Collaboration.Members {
		actorID := strings.TrimSpace(member.ActorID)
		roleKey := strings.TrimSpace(member.RoleKey)
		if actorID == "" || roleKey == "" {
			return collaborationPolicy{}, errors.New("collaboration member actor_id and role_key are required")
		}
		policy.members[actorID] = roleKey
	}
	for roleKey, rolePolicy := range config.Collaboration.Roles {
		trimmedRoleKey := strings.TrimSpace(roleKey)
		if trimmedRoleKey == "" {
			return collaborationPolicy{}, errors.New("collaboration role key is required")
		}
		policy.roles[trimmedRoleKey] = rolePolicy
	}
	policy.enabled = len(policy.members) > 0 || len(policy.roles) > 0
	return policy, nil
}

// Validate 校验一次交互是否满足多人协作权限规则。
func (p collaborationPolicy) Validate(sceneCode string, actionCode string, paramsJSON []byte, ctx InteractionContext) error {
	if !p.enabled {
		return nil
	}

	actorID := strings.TrimSpace(ctx.ActorID)
	roleKey := strings.TrimSpace(ctx.RoleKey)
	if actorID == "" || roleKey == "" {
		return errors.New("actor_id and role_key are required for collaboration session")
	}

	expectedRoleKey, ok := p.members[actorID]
	if !ok {
		return fmt.Errorf("actor %s is not registered in collaboration session", actorID)
	}
	if expectedRoleKey != roleKey {
		return fmt.Errorf("actor %s role mismatch", actorID)
	}

	rolePolicy, ok := p.roles[roleKey]
	if !ok {
		return fmt.Errorf("role %s is not configured", roleKey)
	}

	scenePolicy, ok := rolePolicy.Scenes[sceneCode]
	if !ok {
		return fmt.Errorf("role %s has no access to scene %s", roleKey, sceneCode)
	}
	if !containsString(scenePolicy.Actions, actionCode) {
		return fmt.Errorf("role %s cannot execute action %s", roleKey, actionCode)
	}

	if len(scenePolicy.ResourceIDs) == 0 {
		return nil
	}

	resourceIDs, err := extractResourceIDs(paramsJSON)
	if err != nil {
		return err
	}
	for _, resourceID := range resourceIDs {
		if !containsString(scenePolicy.ResourceIDs, resourceID) {
			return fmt.Errorf("resource %s is out of role scope", resourceID)
		}
	}
	return nil
}

// extractResourceIDs 从交互参数中提取标准资源标识字段，并显式拒绝旧别名键。
func extractResourceIDs(paramsJSON []byte) ([]string, error) {
	if len(paramsJSON) == 0 {
		return nil, nil
	}

	var payload map[string]any
	if err := json.Unmarshal(paramsJSON, &payload); err != nil {
		return nil, nil
	}

	deprecatedKeys := []string{
		"node_id",
		"node_ids",
		"validator_id",
		"validator_ids",
		"delegate_id",
		"delegate_ids",
		"leader_id",
		"leader_ids",
		"target",
		"target_id",
		"target_ids",
		"target_node_id",
		"from_node_id",
		"to_node_id",
	}
	for _, key := range deprecatedKeys {
		if _, exists := payload[key]; exists {
			return nil, fmt.Errorf("deprecated resource parameter key %s is not allowed", key)
		}
	}

	result := make([]string, 0)
	if value, ok := payload["resource_id"].(string); ok && strings.TrimSpace(value) != "" {
		result = append(result, strings.TrimSpace(value))
	}
	items, ok := payload["resource_ids"].([]any)
	if ok {
		for _, item := range items {
			value, ok := item.(string)
			if ok && strings.TrimSpace(value) != "" {
				result = append(result, strings.TrimSpace(value))
			}
		}
	}
	return result, nil
}

// containsString 判断字符串切片中是否包含指定值。
func containsString(values []string, target string) bool {
	for _, value := range values {
		if strings.TrimSpace(value) == target {
			return true
		}
	}
	return false
}
