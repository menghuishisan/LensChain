package catalog

import (
	"github.com/lenschain/sim-engine/scenarios/internal/cryptography/ecdsasign"
	"github.com/lenschain/sim-engine/scenarios/internal/cryptography/keccak256hash"
	"github.com/lenschain/sim-engine/scenarios/internal/cryptography/merkletree"
	"github.com/lenschain/sim-engine/scenarios/internal/cryptography/rsaencrypt"
	"github.com/lenschain/sim-engine/scenarios/internal/cryptography/sha256hash"
	"github.com/lenschain/sim-engine/scenarios/internal/cryptography/zkpbasic"
	"github.com/lenschain/sim-engine/scenarios/internal/framework"
)

// cryptographyDefinitions 返回密码学领域的 6 个场景定义。
func cryptographyDefinitions() []framework.Definition {
	return []framework.Definition{
		buildDefinition(SceneTemplate{
			Code:            "sha256-hash",
			Name:            "SHA-256 哈希过程",
			Description:     "展示消息分块、填充和 64 轮压缩函数的状态演化。",
			CategoryCode:    "cryptography",
			AlgorithmType:   "sha256-hash",
			Version:         "v1.0.0",
			TimeControlMode: "reactive",
			DataSourceMode:  "simulation",
			LinkGroups:      []string{"crypto-verify-group"},
			Profile:         StepProfile{Stages: []string{"消息分块", "消息填充", "压缩轮函数", "哈希输出"}, TotalTicks: 64, StepDuration: 900},
			BaseNodeLabels:  []string{"Input", "Schedule", "Round", "Digest"},
			BaseNodeRole:    "hash",
			Actions: []ActionSpec{
				{ActionCode: "mutate_input", Label: "修改输入", Description: "修改原始输入以观察雪崩效应。", Trigger: "form_submit", FieldKey: "input", FieldLabel: "输入内容", FieldType: "string", DefaultValue: "abc"},
			},
			DefaultState:  sha256hash.DefaultState,
			InitHandler:   sha256hash.Init,
			StepHandler:   sha256hash.Step,
			ActionHandler: sha256hash.HandleAction,
			SyncHandler:   sha256hash.SyncSharedState,
			RenderBuilder: sha256hash.BuildRenderState,
		}),
		buildDefinition(SceneTemplate{
			Code:            "keccak256-hash",
			Name:            "Keccak-256 哈希过程",
			Description:     "展示海绵结构吸收、置换和挤压输出。",
			CategoryCode:    "cryptography",
			AlgorithmType:   "keccak256-hash",
			Version:         "v1.0.0",
			TimeControlMode: "reactive",
			DataSourceMode:  "simulation",
			Profile:         StepProfile{Stages: []string{"吸收", "Theta/Rho/Pi", "Chi/Iota", "挤压输出"}, TotalTicks: 24, StepDuration: 900},
			BaseNodeLabels:  []string{"Lane-0", "Lane-1", "Lane-2", "Lane-3", "Lane-4"},
			BaseNodeRole:    "lane",
			Actions: []ActionSpec{
				{ActionCode: "toggle_lane", Label: "切换 Lane", Description: "扰动状态矩阵中的某一列。", Trigger: "click", FieldKey: "lane", FieldLabel: "Lane 索引", FieldType: "number", DefaultValue: "0"},
			},
			DefaultState:  keccak256hash.DefaultState,
			InitHandler:   keccak256hash.Init,
			StepHandler:   keccak256hash.Step,
			ActionHandler: keccak256hash.HandleAction,
			RenderBuilder: keccak256hash.BuildRenderState,
		}),
		buildDefinition(SceneTemplate{
			Code:            "ecdsa-sign",
			Name:            "ECDSA 签名验签",
			Description:     "展示密钥对、随机数生成、签名和验签流程。",
			CategoryCode:    "cryptography",
			AlgorithmType:   "ecdsa-sign",
			Version:         "v1.0.0",
			TimeControlMode: "reactive",
			DataSourceMode:  "simulation",
			LinkGroups:      []string{"crypto-verify-group"},
			Profile:         StepProfile{Stages: []string{"生成随机数 k", "点乘运算", "签名输出", "验签验证"}, TotalTicks: 12, StepDuration: 1000},
			BaseNodeLabels:  []string{"PrivateKey", "PublicKey", "Message", "Signature"},
			BaseNodeRole:    "crypto",
			Actions: []ActionSpec{
				{ActionCode: "sign_message", Label: "重新签名", Description: "使用当前私钥对新消息签名。", Trigger: "form_submit", FieldKey: "message", FieldLabel: "消息内容", FieldType: "string", DefaultValue: "hello"},
			},
			DefaultState:  ecdsasign.DefaultState,
			InitHandler:   ecdsasign.Init,
			StepHandler:   ecdsasign.Step,
			ActionHandler: ecdsasign.HandleAction,
			SyncHandler:   ecdsasign.SyncSharedState,
			RenderBuilder: ecdsasign.BuildRenderState,
		}),
		buildDefinition(SceneTemplate{
			Code:            "rsa-encrypt",
			Name:            "RSA 加密解密",
			Description:     "展示大数模幂运算和加密、解密、验签关系。",
			CategoryCode:    "cryptography",
			AlgorithmType:   "rsa-encrypt",
			Version:         "v1.0.0",
			TimeControlMode: "reactive",
			DataSourceMode:  "simulation",
			Profile:         StepProfile{Stages: []string{"选择质数", "计算模数", "加密", "解密"}, TotalTicks: 10, StepDuration: 1000},
			BaseNodeLabels:  []string{"Prime-p", "Prime-q", "Cipher", "Plaintext"},
			BaseNodeRole:    "rsa",
			Actions: []ActionSpec{
				{ActionCode: "encrypt_plaintext", Label: "重新加密", Description: "输入新的明文并执行加密。", Trigger: "form_submit", FieldKey: "plaintext", FieldLabel: "明文", FieldType: "string", DefaultValue: "42"},
			},
			DefaultState:  rsaencrypt.DefaultState,
			InitHandler:   rsaencrypt.Init,
			StepHandler:   rsaencrypt.Step,
			ActionHandler: rsaencrypt.HandleAction,
			RenderBuilder: rsaencrypt.BuildRenderState,
		}),
		buildDefinition(SceneTemplate{
			Code:            "merkle-tree",
			Name:            "Merkle 树构建验证",
			Description:     "展示叶子哈希、逐层合并和验证路径。",
			CategoryCode:    "cryptography",
			AlgorithmType:   "merkle-tree",
			Version:         "v1.0.0",
			TimeControlMode: "reactive",
			DataSourceMode:  "simulation",
			LinkGroups:      []string{"crypto-verify-group", "blockchain-integrity-group"},
			Profile:         StepProfile{Stages: []string{"叶子哈希", "两两合并", "根节点输出", "路径验证"}, TotalTicks: 16, StepDuration: 900},
			BaseNodeLabels:  []string{"Leaf-1", "Leaf-2", "Leaf-3", "Leaf-4"},
			BaseNodeRole:    "leaf",
			Actions: []ActionSpec{
				{ActionCode: "tamper_leaf", Label: "篡改叶子", Description: "修改叶子数据观察验证路径变化。", Trigger: "click", FieldKey: "leaf", FieldLabel: "叶子索引", FieldType: "number", DefaultValue: "0"},
			},
			DefaultState:  merkletree.DefaultState,
			InitHandler:   merkletree.Init,
			StepHandler:   merkletree.Step,
			ActionHandler: merkletree.HandleAction,
			SyncHandler:   merkletree.SyncSharedState,
			RenderBuilder: merkletree.BuildRenderState,
		}),
		buildDefinition(SceneTemplate{
			Code:            "zkp-basic",
			Name:            "零知识证明原理",
			Description:     "展示承诺、挑战、响应三步交互。",
			CategoryCode:    "cryptography",
			AlgorithmType:   "zkp-basic",
			Version:         "v1.0.0",
			TimeControlMode: "process",
			DataSourceMode:  "simulation",
			Profile:         StepProfile{Stages: []string{"承诺", "挑战", "响应", "验证通过"}, TotalTicks: 12, StepDuration: 1400},
			BaseNodeLabels:  []string{"Prover", "Verifier"},
			BaseNodeRole:    "zkp",
			Actions: []ActionSpec{
				{ActionCode: "change_secret", Label: "更换秘密", Description: "切换证明者的秘密值。", Trigger: "form_submit", FieldKey: "secret", FieldLabel: "秘密值", FieldType: "string", DefaultValue: "s1"},
			},
			DefaultState:  zkpbasic.DefaultState,
			InitHandler:   zkpbasic.Init,
			StepHandler:   zkpbasic.Step,
			ActionHandler: zkpbasic.HandleAction,
			RenderBuilder: zkpbasic.BuildRenderState,
		}),
	}
}
