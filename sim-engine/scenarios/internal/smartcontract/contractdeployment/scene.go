// 模块：sim-engine/scenarios/internal/smartcontract/contractdeployment
// 文件职责：SC-04 合约部署场景的完整实现（CREATE / CREATE2）。
//
// SSOT 依据：06.md §4.6.4 / §3.2 / §6.2 / §6.3 / §8.3；AGENTS.md §0 + §6。
//
// 算法实现：自实现 EVM 合约部署的完整流程（零外部依赖；Keccak-256 复用 keccak256hash 兄弟包）：
//
//   1. CREATE 地址推导：
//        addr = keccak256(rlp([sender_address, sender_nonce]))[12:]
//      · 自实现极简 RLP 编码（足够覆盖部署用例：地址 20B + nonce uint）。
//      · 部署后 sender_nonce += 1（与 EVM 一致）。
//
//   2. CREATE2 地址推导（EIP-1014）：
//        addr = keccak256(0xff ++ sender_address ++ salt(32B) ++ keccak256(init_code))[12:]
//      · 不依赖 nonce，地址确定可预测；同 (sender, salt, init_code) → 同地址。
//      · 已被部署的地址不能再次部署（codeSize > 0 → REVERT）。
//
//   3. 部署执行流程：
//        a) gasCheck                : 检查发起者 gas / value 余额
//        b) addressDerivation       : 计算目标地址（CREATE 或 CREATE2）
//        c) collisionCheck          : 检查目标地址是否已有 code
//        d) constructorExecution    : 执行 init_code 的 constructor 段，
//                                     初始化 storage（教学版用结构化字段）
//        e) runtimeStorage          : 把 runtime_code 写入目标账户
//        f) finalize                : 增加 sender nonce、转账 value
//      · 任意阶段失败 → REVERT，全部状态回滚。
//
//   4. 教学合约模板（init_code 用结构化字段表示，避免依赖完整 EVM 解析器）：
//        ContractTemplate {
//          name           string  // 合约逻辑标识（"Counter" / "Token" / "Vault"）
//          ConstructorArg int64   // 初始化参数（counter.start / token.supply / vault.cap）
//          RuntimeSize    int     // 部署后 runtime code 字节数（用于 gas 估算）
//        }
//      · keccak256(init_code) 用模板的 deterministic 序列化作输入。
//
//   5. 攻击 / 异常分支：
//        · CREATE2 重放部署：相同 (sender, salt, init_code) 第二次部署 → REVERT
//        · constructor revert：构造函数显式失败 → 部署失败、地址保留为空
//        · gas 不足 / value 超额 → REVERT
//        · selfdestruct 后 CREATE2 重新占用同一地址（教学：演示 EIP-6780 之前的可行性）

package contractdeployment

import (
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"strings"

	fw "github.com/lenschain/sim-engine/framework"
	"github.com/lenschain/sim-engine/scenarios/internal/cryptography/keccak256hash"
)

const (
	sceneCode     = "contract-deployment"
	schemaVersion = "v1.0.0"
	algorithmType = "evm-create-create2"

	createTypeCREATE  = "CREATE"
	createTypeCREATE2 = "CREATE2"

	tplCounter = "Counter"
	tplToken   = "Token"
	tplVault   = "Vault"

	gasCreateBase    = 32000
	gasPerByte       = 200 // 每字节 runtime code 的 gas
	defaultDeployGas = 200000

	linkGroupContractSec = "contract-security-group"
	linkOwnerSubtree     = "contract.deploy"
)

// =====================================================================
// 数据结构
// =====================================================================

// contractTemplate 教学合约模板：init_code 的结构化表示。
type contractTemplate struct {
	Name                  string
	ConstructorArg        int64
	RuntimeSize           int
	ConstructorWillRevert bool // 教学开关：演示 constructor 失败
}

// initCodeBytes 把模板序列化成 init_code（教学版的"字节码"）。
func (t contractTemplate) initCodeBytes() []byte {
	out := []byte{}
	out = append(out, []byte(t.Name)...)
	out = append(out, '|')
	var buf [8]byte
	binary.BigEndian.PutUint64(buf[:], uint64(t.ConstructorArg))
	out = append(out, buf[:]...)
	binary.BigEndian.PutUint64(buf[:], uint64(t.RuntimeSize))
	out = append(out, buf[:]...)
	if t.ConstructorWillRevert {
		out = append(out, 0xFF)
	} else {
		out = append(out, 0x00)
	}
	return out
}

// runtimeCodeBytes 部署后存到链上的 runtime code（教学版直接是 RuntimeSize 长度的伪代码）。
func (t contractTemplate) runtimeCodeBytes() []byte {
	out := make([]byte, t.RuntimeSize)
	tag := []byte(t.Name)
	for i := 0; i < t.RuntimeSize; i++ {
		out[i] = tag[i%len(tag)]
	}
	return out
}

// deployedContract 已部署到链上的合约账户。
type deployedContract struct {
	Address      string // 0x 开头 20 字节 hex
	Creator      string // 部署者地址
	CreationType string // CREATE / CREATE2
	TemplateName string
	RuntimeCode  []byte
	Storage      map[string]int64
	Balance      int64
	NonceAtBirth uint64 // 仅 CREATE 有意义
	Salt         string // 仅 CREATE2 有意义（hex）
	InitCodeHash string // keccak256(init_code) hex
	Destroyed    bool
}

// account 发起者账户（EOA 或合约）。
type account struct {
	Address string
	Nonce   uint64
	Balance int64
}

// deployRecord 一次部署的 trace 记录。
type deployRecord struct {
	Tick         int
	CreationType string
	Creator      string
	TemplateName string
	Salt         string
	InitCodeHash string
	DerivedAddr  string
	GasInit      int
	GasUsed      int
	Value        int64
	Success      bool
	RevertReason string
	NonceBefore  uint64
	NonceAfter   uint64
}

type snapState struct {
	Accounts           map[string]*account
	Contracts          map[string]*deployedContract
	History            []deployRecord
	Tick               int
	CollisionBlocks    int // CREATE2 碰撞被拦截的次数
	ConstructorReverts int
	GasFailures        int
	LastError          string
}

func defaultSnapState() snapState {
	st := snapState{
		Accounts:  map[string]*account{},
		Contracts: map[string]*deployedContract{},
	}
	st.Accounts["0x1111111111111111111111111111111111111111"] = &account{
		Address: "0x1111111111111111111111111111111111111111", Nonce: 0, Balance: 1000000,
	}
	st.Accounts["0x2222222222222222222222222222222222222222"] = &account{
		Address: "0x2222222222222222222222222222222222222222", Nonce: 5, Balance: 500000,
	}
	return st
}

// =====================================================================
// 极简 RLP 编码（仅覆盖部署所需：地址字节串 + uint nonce）
// =====================================================================

// rlpEncodeBytes RLP 编码字节串。
func rlpEncodeBytes(b []byte) []byte {
	if len(b) == 1 && b[0] < 0x80 {
		return []byte{b[0]}
	}
	if len(b) <= 55 {
		return append([]byte{0x80 + byte(len(b))}, b...)
	}
	// 长字节串
	lenBytes := encodeBigEndianMinimal(uint64(len(b)))
	out := append([]byte{0xb7 + byte(len(lenBytes))}, lenBytes...)
	return append(out, b...)
}

// rlpEncodeUint64 RLP 编码非负整数。
func rlpEncodeUint64(n uint64) []byte {
	if n == 0 {
		return []byte{0x80}
	}
	return rlpEncodeBytes(encodeBigEndianMinimal(n))
}

// rlpEncodeList RLP 编码列表（拼接好的项编码）。
func rlpEncodeList(items ...[]byte) []byte {
	body := []byte{}
	for _, it := range items {
		body = append(body, it...)
	}
	if len(body) <= 55 {
		return append([]byte{0xc0 + byte(len(body))}, body...)
	}
	lenBytes := encodeBigEndianMinimal(uint64(len(body)))
	out := append([]byte{0xf7 + byte(len(lenBytes))}, lenBytes...)
	return append(out, body...)
}

// encodeBigEndianMinimal 最小化大端编码（去除前导 0）。
func encodeBigEndianMinimal(n uint64) []byte {
	if n == 0 {
		return []byte{}
	}
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, n)
	for i := 0; i < 8; i++ {
		if buf[i] != 0 {
			return buf[i:]
		}
	}
	return buf[7:]
}

// =====================================================================
// 地址推导
// =====================================================================

// addrToBytes 把 0x 前缀的 hex 地址转为 20 字节。
func addrToBytes(addr string) []byte {
	s := strings.TrimPrefix(addr, "0x")
	b, _ := hex.DecodeString(s)
	if len(b) > 20 {
		return b[len(b)-20:]
	}
	if len(b) < 20 {
		out := make([]byte, 20)
		copy(out[20-len(b):], b)
		return out
	}
	return b
}

// deriveCREATE 推导 CREATE 地址。
//
//	addr = keccak256(rlp([sender_addr, sender_nonce]))[12:]
func deriveCREATE(senderAddr string, senderNonce uint64) (string, []byte) {
	senderBytes := addrToBytes(senderAddr)
	rlp := rlpEncodeList(rlpEncodeBytes(senderBytes), rlpEncodeUint64(senderNonce))
	digest := keccak256hash.Sum256(rlp)
	addr := digest[12:]
	return "0x" + hex.EncodeToString(addr), rlp
}

// deriveCREATE2 推导 CREATE2 地址。
//
//	addr = keccak256(0xff ++ sender_addr ++ salt32B ++ keccak256(init_code))[12:]
func deriveCREATE2(senderAddr, saltHex string, initCode []byte) (string, [32]byte, []byte) {
	senderBytes := addrToBytes(senderAddr)
	salt := decodeSalt(saltHex)
	initHash := keccak256hash.Sum256(initCode)
	preimage := []byte{0xff}
	preimage = append(preimage, senderBytes...)
	preimage = append(preimage, salt[:]...)
	preimage = append(preimage, initHash[:]...)
	digest := keccak256hash.Sum256(preimage)
	return "0x" + hex.EncodeToString(digest[12:]), initHash, preimage
}

func decodeSalt(s string) [32]byte {
	s = strings.TrimPrefix(s, "0x")
	var salt [32]byte
	b, _ := hex.DecodeString(s)
	if len(b) > 32 {
		b = b[len(b)-32:]
	}
	copy(salt[32-len(b):], b)
	return salt
}

// =====================================================================
// 部署核心
// =====================================================================

// deploy 执行一次合约部署，返回 deployRecord 与可能的错误。
// 失败时所有状态回滚（账户 nonce / 余额 / 合约表）。
func (st *snapState) deploy(creator, ctype string, tpl contractTemplate, saltHex string, value int64, gas int) deployRecord {
	rec := deployRecord{
		Tick: st.Tick, CreationType: ctype, Creator: creator,
		TemplateName: tpl.Name, Salt: saltHex,
		GasInit: gas, Value: value,
	}
	creatorAcc, ok := st.Accounts[creator]
	if !ok {
		rec.Success = false
		rec.RevertReason = "部署者账户不存在"
		st.History = append(st.History, rec)
		st.GasFailures++
		return rec
	}
	rec.NonceBefore = creatorAcc.Nonce
	rec.NonceAfter = creatorAcc.Nonce

	// (a) gas / 余额检查
	estGas := gasCreateBase + tpl.RuntimeSize*gasPerByte
	if gas < estGas {
		rec.Success = false
		rec.RevertReason = fmt.Sprintf("gas 不足: %d < %d", gas, estGas)
		st.History = append(st.History, rec)
		st.GasFailures++
		return rec
	}
	if creatorAcc.Balance < value {
		rec.Success = false
		rec.RevertReason = fmt.Sprintf("余额不足: %d < %d", creatorAcc.Balance, value)
		st.History = append(st.History, rec)
		st.GasFailures++
		return rec
	}

	// (b) init_code 与 hash
	initCode := tpl.initCodeBytes()
	initHash := keccak256hash.Sum256(initCode)
	rec.InitCodeHash = "0x" + hex.EncodeToString(initHash[:])

	// (b2) 地址推导
	var derived string
	switch ctype {
	case createTypeCREATE:
		derived, _ = deriveCREATE(creator, creatorAcc.Nonce)
	case createTypeCREATE2:
		derived, _, _ = deriveCREATE2(creator, saltHex, initCode)
	default:
		rec.Success = false
		rec.RevertReason = "未知 create 类型: " + ctype
		st.History = append(st.History, rec)
		return rec
	}
	rec.DerivedAddr = derived

	// (c) collision check
	if existing, ok := st.Contracts[derived]; ok && !existing.Destroyed {
		rec.Success = false
		rec.RevertReason = "地址已被占用（CREATE2 collision）"
		st.History = append(st.History, rec)
		st.CollisionBlocks++
		return rec
	}

	// (d) constructor execution
	if tpl.ConstructorWillRevert {
		rec.Success = false
		rec.RevertReason = "constructor 显式 REVERT"
		rec.GasUsed = gasCreateBase / 2 // constructor 已消耗一半
		st.History = append(st.History, rec)
		st.ConstructorReverts++
		return rec
	}
	storage := constructorInit(tpl)

	// (e) runtime storage
	newC := &deployedContract{
		Address:      derived,
		Creator:      creator,
		CreationType: ctype,
		TemplateName: tpl.Name,
		RuntimeCode:  tpl.runtimeCodeBytes(),
		Storage:      storage,
		Balance:      value,
		NonceAtBirth: creatorAcc.Nonce,
		Salt:         saltHex,
		InitCodeHash: rec.InitCodeHash,
	}
	st.Contracts[derived] = newC

	// (f) finalize
	creatorAcc.Balance -= value
	if ctype == createTypeCREATE {
		creatorAcc.Nonce++
	}
	rec.NonceAfter = creatorAcc.Nonce
	rec.Success = true
	rec.GasUsed = estGas
	st.History = append(st.History, rec)
	if len(st.History) > 32 {
		st.History = st.History[len(st.History)-32:]
	}
	return rec
}

// constructorInit 教学版构造函数：根据模板初始化 storage。
func constructorInit(tpl contractTemplate) map[string]int64 {
	storage := map[string]int64{}
	switch tpl.Name {
	case tplCounter:
		storage["count"] = tpl.ConstructorArg
	case tplToken:
		storage["totalSupply"] = tpl.ConstructorArg
		storage["balance:owner"] = tpl.ConstructorArg
	case tplVault:
		storage["cap"] = tpl.ConstructorArg
		storage["deposited"] = 0
	default:
		storage["arg"] = tpl.ConstructorArg
	}
	return storage
}

// selfdestruct 标记合约销毁（教学版：保留地址但 RuntimeCode 清空、Destroyed=true，
// 以演示 EIP-6780 之前 CREATE2 可重新占用同一地址）。
func (st *snapState) selfdestruct(addr, beneficiary string) error {
	c, ok := st.Contracts[addr]
	if !ok {
		return fmt.Errorf("合约不存在: %s", addr)
	}
	if c.Destroyed {
		return errors.New("已销毁")
	}
	if benef, ok := st.Accounts[beneficiary]; ok {
		benef.Balance += c.Balance
	}
	c.Balance = 0
	c.RuntimeCode = nil
	c.Destroyed = true
	return nil
}

// =====================================================================
// 持久化
// =====================================================================

func loadState(s *fw.SceneState) snapState {
	if s == nil || s.Data == nil {
		return defaultSnapState()
	}
	d := s.Data
	st := snapState{
		Accounts:           map[string]*account{},
		Contracts:          map[string]*deployedContract{},
		Tick:               fw.MapInt(d, "tick", 0),
		CollisionBlocks:    fw.MapInt(d, "collision_blocks", 0),
		ConstructorReverts: fw.MapInt(d, "ctor_reverts", 0),
		GasFailures:        fw.MapInt(d, "gas_failures", 0),
		LastError:          fw.MapStr(d, "last_error", ""),
	}
	if accAny, ok := d["accounts"].(map[string]any); ok {
		for addr, aAny := range accAny {
			if am, ok := aAny.(map[string]any); ok {
				st.Accounts[addr] = &account{
					Address: addr,
					Nonce:   uint64(fw.MapInt(am, "nonce", 0)),
					Balance: int64(fw.MapInt(am, "balance", 0)),
				}
			}
		}
	}
	if len(st.Accounts) == 0 {
		return defaultSnapState()
	}
	if csAny, ok := d["contracts"].(map[string]any); ok {
		for addr, cAny := range csAny {
			if cm, ok := cAny.(map[string]any); ok {
				c := &deployedContract{
					Address:      addr,
					Creator:      fw.MapStr(cm, "creator", ""),
					CreationType: fw.MapStr(cm, "ctype", ""),
					TemplateName: fw.MapStr(cm, "tpl", ""),
					Balance:      int64(fw.MapInt(cm, "balance", 0)),
					NonceAtBirth: uint64(fw.MapInt(cm, "nonce_at_birth", 0)),
					Salt:         fw.MapStr(cm, "salt", ""),
					InitCodeHash: fw.MapStr(cm, "init_hash", ""),
					Destroyed:    fw.MapBool(cm, "destroyed", false),
					Storage:      map[string]int64{},
				}
				if rcHex, ok := cm["runtime"].(string); ok {
					if b, err := hex.DecodeString(rcHex); err == nil {
						c.RuntimeCode = b
					}
				}
				if stoAny, ok := cm["storage"].(map[string]any); ok {
					for k, v := range stoAny {
						c.Storage[k] = int64(intFromAny(v))
					}
				}
				st.Contracts[addr] = c
			}
		}
	}
	if hAny, ok := d["history"].([]any); ok {
		for _, rAny := range hAny {
			if rm, ok := rAny.(map[string]any); ok {
				st.History = append(st.History, deployRecord{
					Tick: fw.MapInt(rm, "tick", 0), CreationType: fw.MapStr(rm, "ctype", ""),
					Creator: fw.MapStr(rm, "creator", ""), TemplateName: fw.MapStr(rm, "tpl", ""),
					Salt: fw.MapStr(rm, "salt", ""), InitCodeHash: fw.MapStr(rm, "init_hash", ""),
					DerivedAddr: fw.MapStr(rm, "addr", ""),
					GasInit:     fw.MapInt(rm, "gas_init", 0), GasUsed: fw.MapInt(rm, "gas_used", 0),
					Value:        int64(fw.MapInt(rm, "value", 0)),
					Success:      fw.MapBool(rm, "success", false),
					RevertReason: fw.MapStr(rm, "reason", ""),
					NonceBefore:  uint64(fw.MapInt(rm, "nonce_before", 0)),
					NonceAfter:   uint64(fw.MapInt(rm, "nonce_after", 0)),
				})
			}
		}
	}
	return st
}

func saveState(s *fw.SceneState, st snapState) {
	if s.Data == nil {
		s.Data = map[string]any{}
	}
	s.Data["tick"] = st.Tick
	s.Data["collision_blocks"] = st.CollisionBlocks
	s.Data["ctor_reverts"] = st.ConstructorReverts
	s.Data["gas_failures"] = st.GasFailures
	s.Data["last_error"] = st.LastError
	accAny := map[string]any{}
	for addr, a := range st.Accounts {
		accAny[addr] = map[string]any{"nonce": int(a.Nonce), "balance": int(a.Balance)}
	}
	s.Data["accounts"] = accAny
	csAny := map[string]any{}
	for addr, c := range st.Contracts {
		stoAny := map[string]any{}
		for k, v := range c.Storage {
			stoAny[k] = int(v)
		}
		csAny[addr] = map[string]any{
			"creator": c.Creator, "ctype": c.CreationType, "tpl": c.TemplateName,
			"balance": int(c.Balance), "nonce_at_birth": int(c.NonceAtBirth),
			"salt": c.Salt, "init_hash": c.InitCodeHash, "destroyed": c.Destroyed,
			"runtime": hex.EncodeToString(c.RuntimeCode), "storage": stoAny,
		}
	}
	s.Data["contracts"] = csAny
	hAny := make([]any, len(st.History))
	for i, r := range st.History {
		hAny[i] = map[string]any{
			"tick": r.Tick, "ctype": r.CreationType, "creator": r.Creator,
			"tpl": r.TemplateName, "salt": r.Salt, "init_hash": r.InitCodeHash,
			"addr":     r.DerivedAddr,
			"gas_init": r.GasInit, "gas_used": r.GasUsed,
			"value": int(r.Value), "success": r.Success, "reason": r.RevertReason,
			"nonce_before": int(r.NonceBefore), "nonce_after": int(r.NonceAfter),
		}
	}
	s.Data["history"] = hAny
}

// =====================================================================
// 场景定义
// =====================================================================

func Definition() fw.Definition {
	return fw.Definition{
		Code: sceneCode, Name: "合约部署（CREATE / CREATE2）",
		Description:         "演示合约部署：CREATE 地址推导（rlp[sender,nonce]）/ CREATE2 地址推导（0xff++sender++salt++keccak(init_code)）+ constructor 执行 + 部署 collision / gas 不足 / constructor REVERT 等异常",
		Category:            fw.CategorySmartContract,
		AlgorithmType:       algorithmType,
		Version:             schemaVersion,
		TimeControlMode:     fw.TimeControlReactive,
		DataSourceMode:      fw.DataSourceSimulation,
		SupportedLinkGroups: []string{linkGroupContractSec},

		ExtensionLevel:     fw.ExtensionL1,
		LinkGroupVersion:   "v0.5.0",
		SupportsMultiActor: false,
		OwnedFieldPaths: []string{
			"contract.deploy.last_address",
			"contract.deploy.collision_blocks",
		},

		DefaultParams: func() map[string]any { return map[string]any{} },
		DefaultState:  defaultStateFw,
		Interaction:   interactionDefinition,
		Init:                initScene,
		Step:                stepScene,
		HandleAction:        handleAction,
	}
}

func defaultStateFw() fw.SceneState {
	return fw.SceneState{SceneCode: sceneCode, Tick: 0, Phase: "ready", Data: map[string]any{}}
}

func interactionDefinition() fw.InteractionDefinition {
	return fw.InteractionDefinition{
		SceneCode: sceneCode, SchemaVersion: schemaVersion,
		Actions: []fw.ActionDef{
			{
				ActionCode: "deploy_create", Label: "CREATE 部署",
				Description: "用 sender + nonce 推导地址；部署后 nonce++",
				Category:    fw.ActionPrimary, Trigger: fw.TriggerSubmit,
				Roles: []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				Fields: []fw.FieldDef{
					{Name: "creator", Type: fw.FieldString, Label: "creator", Required: true,
						Default: "0x1111111111111111111111111111111111111111"},
					{Name: "template", Type: fw.FieldEnum, Label: "模板", Required: true, Default: tplCounter,
						Options: []any{tplCounter, tplToken, tplVault}},
					{Name: "ctor_arg", Type: fw.FieldNumber, Label: "constructor 参数", Required: true, Default: 100, Step: 1},
					{Name: "runtime_size", Type: fw.FieldNumber, Label: "runtime 字节数", Required: true, Default: 200, Min: 0, Max: 24576, Step: 50},
					{Name: "value", Type: fw.FieldNumber, Label: "value", Required: false, Default: 0, Min: 0, Step: 100},
					{Name: "gas", Type: fw.FieldNumber, Label: "gas", Required: true, Default: defaultDeployGas, Min: 1000, Step: 1000},
					{Name: "ctor_revert", Type: fw.FieldBoolean, Label: "构造函数 REVERT？", Required: false, Default: false},
				},
				WritesOwnedFields: []string{"contract.deploy.last_address"},
				LinkOwnerFields:   []string{"contract.deploy.last_address"},
			},
			{
				ActionCode: "deploy_create2", Label: "CREATE2 部署",
				Description: "用 0xff ++ sender ++ salt ++ keccak(init_code) 推导地址；不依赖 nonce",
				Category:    fw.ActionPrimary, Trigger: fw.TriggerSubmit,
				Roles: []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				Fields: []fw.FieldDef{
					{Name: "creator", Type: fw.FieldString, Label: "creator", Required: true,
						Default: "0x1111111111111111111111111111111111111111"},
					{Name: "salt", Type: fw.FieldString, Label: "salt (hex)", Required: true,
						Default: "0x0000000000000000000000000000000000000000000000000000000000000001"},
					{Name: "template", Type: fw.FieldEnum, Label: "模板", Required: true, Default: tplCounter,
						Options: []any{tplCounter, tplToken, tplVault}},
					{Name: "ctor_arg", Type: fw.FieldNumber, Label: "constructor 参数", Required: true, Default: 100, Step: 1},
					{Name: "runtime_size", Type: fw.FieldNumber, Label: "runtime 字节数", Required: true, Default: 200, Min: 0, Max: 24576, Step: 50},
					{Name: "value", Type: fw.FieldNumber, Label: "value", Required: false, Default: 0, Min: 0, Step: 100},
					{Name: "gas", Type: fw.FieldNumber, Label: "gas", Required: true, Default: defaultDeployGas, Min: 1000, Step: 1000},
					{Name: "ctor_revert", Type: fw.FieldBoolean, Label: "构造函数 REVERT？", Required: false, Default: false},
				},
				WritesOwnedFields: []string{"contract.deploy.last_address"},
				LinkOwnerFields:   []string{"contract.deploy.last_address"},
			},
			{
				ActionCode: "predict_address", Label: "预测地址（不部署）",
				Description: "仅推导 CREATE / CREATE2 地址，验证可预测性",
				Category:    fw.ActionPrimary, Trigger: fw.TriggerSubmit,
				Roles: []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				Fields: []fw.FieldDef{
					{Name: "creator", Type: fw.FieldString, Label: "creator", Required: true,
						Default: "0x1111111111111111111111111111111111111111"},
					{Name: "ctype", Type: fw.FieldEnum, Label: "推导类型", Required: true, Default: createTypeCREATE,
						Options: []any{createTypeCREATE, createTypeCREATE2}},
					{Name: "nonce", Type: fw.FieldNumber, Label: "nonce (CREATE 用)", Required: false, Default: 0, Min: 0, Step: 1},
					{Name: "salt", Type: fw.FieldString, Label: "salt (CREATE2 用)", Required: false, Default: "0x01"},
					{Name: "template", Type: fw.FieldEnum, Label: "模板", Required: true, Default: tplCounter,
						Options: []any{tplCounter, tplToken, tplVault}},
					{Name: "ctor_arg", Type: fw.FieldNumber, Label: "constructor 参数", Required: true, Default: 100, Step: 1},
					{Name: "runtime_size", Type: fw.FieldNumber, Label: "runtime 字节数", Required: true, Default: 200, Min: 0, Step: 50},
				},
			},
			{
				ActionCode: "selfdestruct", Label: "selfdestruct 销毁合约",
				Description: "销毁后腾出地址；演示 CREATE2 可重新占用",
				Category:    fw.ActionAttackInject, Trigger: fw.TriggerSubmit,
				Roles: []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				Fields: []fw.FieldDef{
					{Name: "address", Type: fw.FieldString, Label: "目标合约", Required: true, Default: ""},
					{Name: "beneficiary", Type: fw.FieldString, Label: "余额接收者", Required: true,
						Default: "0x1111111111111111111111111111111111111111"},
				},
			},
			{
				ActionCode: "demo_replay_create2", Label: "演示：CREATE2 重放部署",
				Description: "对同一 (sender, salt, init_code) 部署两次，第二次应被 collision 拦截",
				Category:    fw.ActionAttackInject, Trigger: fw.TriggerSubmit,
				Roles:           []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				LinkOwnerFields: []string{"contract.deploy.collision_blocks"},
			},
			{
				ActionCode: "reset", Label: "重置",
				Category: fw.ActionPrimary, Trigger: fw.TriggerImmediate,
				Roles: []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
			},
			{
				ActionCode:    "teacher_force_revert",
				Label:         "教师强制回滚",
				Description:   "仅教师可用，强制回滚用于教学演示",
				Category:      fw.ActionAttackInject,
				Trigger:       fw.TriggerSubmit,
				Roles:         []fw.UserRole{fw.RoleTeacher},
				InterveneType: fw.InterveneHint,
				Fields: []fw.FieldDef{
					{Name: "description", Type: fw.FieldString, Label: "描述", Required: false, Default: "教师强制回滚"},
				},
			},
			{
				ActionCode:    "compile_and_deploy",
				Label:         "编译并部署（真实链）",
				Description:   "调 remix-ide / solc 编译后部署到 geth",
				Category:      fw.ActionPrimary,
				Trigger:       fw.TriggerSubmit,
				Roles:         []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				HybridChannel: fw.HybridChannelContainer,
				ContainerCmd:  `curl -s -X POST -H "Content-Type: application/json" --data '{"jsonrpc":"2.0","method":"eth_sendTransaction","params":[{"from":"{{from}}","data":"{{bytecode}}","gas":"0x1000000"}],"id":1}' http://geth:8545`,
				Reversible:    false,
				Fields: []fw.FieldDef{
					{Name: "from", Type: fw.FieldString, Label: "from address", Required: true, Default: "0x0000000000000000000000000000000000000001"},
					{Name: "bytecode", Type: fw.FieldString, Label: "compiled bytecode (hex)", Required: true, Default: "0x6080604052"},
				},
			},
			fw.BroadcastHintAction(),
		},
	}
}

// =====================================================================
// 钩子
// =====================================================================

func initScene(state *fw.SceneState, in fw.InitInput) (fw.RenderEnvelope, error) {
	st := loadState(state)
	saveState(state, st)
	state.Phase = "ready"
	env := buildEnvelope(st, "init", "Deployment 初始化（2 个 EOA）", true)
	publishOwnerSubtree(&env, st)
	return env, nil
}

func stepScene(state *fw.SceneState, in fw.StepInput) (fw.StepOutput, error) {
	st := loadState(state)
	state.Tick = in.Tick
	return fw.StepOutput{Render: buildEnvelope(st, "tick", "", false)}, nil
}

func handleAction(state *fw.SceneState, in fw.ActionInput) (fw.ActionOutput, error) {
	_ = fw.EnsureActorBucket(state, in.ActorID)
	if out, ok := fw.HandleBroadcastHint(state, in); ok {
		return out, nil
	}

	st := loadState(state)
	out := fw.ActionOutput{Success: true}
	st.Tick++

	switch in.ActionCode {
	case "deploy_create":
		creator := fw.MapStr(in.Params, "creator", "0x1111111111111111111111111111111111111111")
		tpl := contractTemplate{
			Name:                  fw.MapStr(in.Params, "template", tplCounter),
			ConstructorArg:        int64(fw.MapInt(in.Params, "ctor_arg", 100)),
			RuntimeSize:           fw.MapInt(in.Params, "runtime_size", 200),
			ConstructorWillRevert: fw.MapBool(in.Params, "ctor_revert", false),
		}
		value := int64(fw.MapInt(in.Params, "value", 0))
		gas := fw.MapInt(in.Params, "gas", defaultDeployGas)
		rec := st.deploy(creator, createTypeCREATE, tpl, "", value, gas)
		saveState(state, st)
		summary := summarizeRecord(rec)
		out.Render = buildEnvelope(st, "deploy_create", summary, false)
		appendDeployMicroSteps(&out.Render, rec)
		out.SharedStateDiff = ownerDiff(st)
		return out, nil

	case "deploy_create2":
		creator := fw.MapStr(in.Params, "creator", "0x1111111111111111111111111111111111111111")
		salt := fw.MapStr(in.Params, "salt", "0x01")
		tpl := contractTemplate{
			Name:                  fw.MapStr(in.Params, "template", tplCounter),
			ConstructorArg:        int64(fw.MapInt(in.Params, "ctor_arg", 100)),
			RuntimeSize:           fw.MapInt(in.Params, "runtime_size", 200),
			ConstructorWillRevert: fw.MapBool(in.Params, "ctor_revert", false),
		}
		value := int64(fw.MapInt(in.Params, "value", 0))
		gas := fw.MapInt(in.Params, "gas", defaultDeployGas)
		rec := st.deploy(creator, createTypeCREATE2, tpl, salt, value, gas)
		saveState(state, st)
		summary := summarizeRecord(rec)
		out.Render = buildEnvelope(st, "deploy_create2", summary, false)
		appendDeployMicroSteps(&out.Render, rec)
		out.SharedStateDiff = ownerDiff(st)
		return out, nil

	case "predict_address":
		creator := fw.MapStr(in.Params, "creator", "0x1111111111111111111111111111111111111111")
		ctype := fw.MapStr(in.Params, "ctype", createTypeCREATE)
		tpl := contractTemplate{
			Name:           fw.MapStr(in.Params, "template", tplCounter),
			ConstructorArg: int64(fw.MapInt(in.Params, "ctor_arg", 100)),
			RuntimeSize:    fw.MapInt(in.Params, "runtime_size", 200),
		}
		var derived, rlpHex, preimageHex, initHashHex string
		if ctype == createTypeCREATE {
			nonce := uint64(fw.MapInt(in.Params, "nonce", 0))
			d, rlpBytes := deriveCREATE(creator, nonce)
			derived = d
			rlpHex = "0x" + hex.EncodeToString(rlpBytes)
		} else {
			salt := fw.MapStr(in.Params, "salt", "0x01")
			d, initHash, preimage := deriveCREATE2(creator, salt, tpl.initCodeBytes())
			derived = d
			initHashHex = "0x" + hex.EncodeToString(initHash[:])
			preimageHex = "0x" + hex.EncodeToString(preimage)
		}
		st.LastError = ""
		saveState(state, st)
		summary := fmt.Sprintf("预测地址 = %s", derived)
		env := buildEnvelope(st, "predict_address", summary, false)
		appendPredictMicroSteps(&env, ctype, derived, rlpHex, initHashHex, preimageHex)
		out.Render = env
		return out, nil

	case "selfdestruct":
		addr := fw.MapStr(in.Params, "address", "")
		benef := fw.MapStr(in.Params, "beneficiary", "0x1111111111111111111111111111111111111111")
		if err := st.selfdestruct(addr, benef); err != nil {
			saveState(state, st)
			out.Render = buildEnvelope(st, "selfdestruct", err.Error(), false)
			return fw.ActionOutput{Success: false, ErrorMessage: err.Error(), Render: out.Render}, nil
		}
		saveState(state, st)
		out.Render = buildEnvelope(st, "selfdestruct", "已销毁 "+addr, false)
		return out, nil

	case "demo_replay_create2":
		creator := "0x1111111111111111111111111111111111111111"
		salt := "0x000000000000000000000000000000000000000000000000000000000000abcd"
		tpl := contractTemplate{Name: tplCounter, ConstructorArg: 42, RuntimeSize: 200}
		// 第一次（应成功）
		rec1 := st.deploy(creator, createTypeCREATE2, tpl, salt, 0, defaultDeployGas)
		// 第二次（应被 collision 拦截）
		rec2 := st.deploy(creator, createTypeCREATE2, tpl, salt, 0, defaultDeployGas)
		saveState(state, st)
		summary := fmt.Sprintf("第一次: %s，第二次: %v %s",
			ifThen(rec1.Success, "✓ 部署 "+rec1.DerivedAddr, "✗ "+rec1.RevertReason),
			rec2.Success, rec2.RevertReason)
		out.Render = buildEnvelope(st, "demo_replay_create2", summary, false)
		appendReplayDemoMicroSteps(&out.Render, rec1, rec2)
		out.SharedStateDiff = ownerDiff(st)
		return out, nil

	case "teacher_force_revert":
		if in.UserRole != fw.RoleTeacher {
			return fw.ActionOutput{Success: false, ErrorMessage: "仅教师可调用"}, nil
		}
		desc, _ := in.Params["description"].(string)
		if desc == "" {
			desc = "教师强制回滚"
		}
		annot := fw.PrimAnnotation(fmt.Sprintf("teacher-revert-%d", state.Tick), "text", "teacher", 5000, map[string]any{"x": 0.5, "y": 0.1}, nil, desc)
		return fw.ActionOutput{
			Success: true,
			Render: fw.RenderEnvelope{
				Primitives:  []fw.Primitive{annot},
				ChangedKeys: []string{"teacher_action"},
			},
		}, nil

	case "reset":
		st = defaultSnapState()
		saveState(state, st)
		out.Render = buildEnvelope(st, "reset", "已重置", true)
		return out, nil
	}
	return fw.ActionOutput{Success: false, ErrorMessage: "未知 ActionCode"}, errors.New("unknown action")
}

func summarizeRecord(rec deployRecord) string {
	if rec.Success {
		return fmt.Sprintf("✓ %s 部署成功：%s（gas=%d，nonce %d→%d）",
			rec.CreationType, rec.DerivedAddr, rec.GasUsed, rec.NonceBefore, rec.NonceAfter)
	}
	return fmt.Sprintf("✗ %s 部署失败：%s", rec.CreationType, rec.RevertReason)
}

func ifThen(cond bool, a, b string) string {
	if cond {
		return a
	}
	return b
}

// =====================================================================
// RenderEnvelope 构造
// =====================================================================

func buildEnvelope(st snapState, reason, summary string, fullSnapshot bool) fw.RenderEnvelope {
	prims := make([]fw.Primitive, 0, 50)

	// 1) EOA 节点
	accAddrs := []string{}
	for a := range st.Accounts {
		accAddrs = append(accAddrs, a)
	}
	sort.Strings(accAddrs)
	accIDs := []string{}
	for _, a := range accAddrs {
		accIDs = append(accIDs, "acc-"+a[:10])
	}
	prims = append(prims, fw.PrimStack("eoa-stack", accIDs, "horizontal"))
	for i, a := range accAddrs {
		acc := st.Accounts[a]
		label := fmt.Sprintf("EOA\n%s\nnonce=%d\nbal=%d", a[:10]+"…", acc.Nonce, acc.Balance)
		prims = append(prims, fw.PrimNode(accIDs[i], label, "active", "eoa"))
	}

	// 2) 部署的合约节点
	cAddrs := []string{}
	for a := range st.Contracts {
		cAddrs = append(cAddrs, a)
	}
	sort.Strings(cAddrs)
	cIDs := []string{}
	for _, a := range cAddrs {
		cIDs = append(cIDs, "c-"+a[:10])
	}
	if len(cIDs) > 0 {
		prims = append(prims, fw.PrimStack("contract-stack", cIDs, "horizontal"))
	}
	for i, a := range cAddrs {
		c := st.Contracts[a]
		role := "contract-create"
		if c.CreationType == createTypeCREATE2 {
			role = "contract-create2"
		}
		status := "active"
		if c.Destroyed {
			status = "error"
			role = "contract-destroyed"
		}
		label := fmt.Sprintf("%s\n%s\n%s\nbal=%d\nrt=%dB",
			c.TemplateName, c.CreationType, a[:10]+"…", c.Balance, len(c.RuntimeCode))
		prims = append(prims, fw.PrimNode(cIDs[i], label, status, role))
	}

	// 3) 部署关系边（最近 5 条）
	startIdx := 0
	if len(st.History) > 5 {
		startIdx = len(st.History) - 5
	}
	for i, r := range st.History[startIdx:] {
		if !r.Success || r.DerivedAddr == "" {
			continue
		}
		creatorID := "acc-" + r.Creator[:10]
		contractID := "c-" + r.DerivedAddr[:10]
		// 只在两端都还存在时画
		if _, ok := st.Accounts[r.Creator]; !ok {
			continue
		}
		if _, ok := st.Contracts[r.DerivedAddr]; !ok {
			continue
		}
		anim := "flow"
		prims = append(prims, fw.PrimEdge(fmt.Sprintf("dep-edge-%d", i+startIdx), creatorID, contractID, "solid", anim))
	}

	// 4) 公式（CREATE / CREATE2）
	prims = append(prims, fw.PrimMathFormula("formula-create",
		`\text{CREATE}: \text{addr} = \mathrm{keccak256}(\mathrm{rlp}(\text{sender}, \text{nonce}))[12:]`, false))
	prims = append(prims, fw.PrimMathFormula("formula-create2",
		`\text{CREATE2}: \text{addr} = \mathrm{keccak256}(\mathtt{0xff} \,\Vert\, \text{sender} \,\Vert\, \text{salt}_{32} \,\Vert\, \mathrm{keccak256}(\text{init\_code}))[12:]`, false))

	// 5) 流水线：gasCheck → addrDerive → collisionCheck → constructor → finalize
	phases := []string{"gas check", "addr derive", "collision check", "constructor", "finalize"}
	phaseIDs := []string{"ph-gas", "ph-derive", "ph-collide", "ph-ctor", "ph-final"}
	prims = append(prims, fw.PrimStack("deploy-pipeline", phaseIDs, "horizontal"))
	curPhase := -1
	if len(st.History) > 0 {
		last := st.History[len(st.History)-1]
		switch {
		case strings.Contains(last.RevertReason, "gas") || strings.Contains(last.RevertReason, "余额"):
			curPhase = 0
		case strings.Contains(last.RevertReason, "未知 create"):
			curPhase = 1
		case strings.Contains(last.RevertReason, "collision") || strings.Contains(last.RevertReason, "已被占用"):
			curPhase = 2
		case strings.Contains(last.RevertReason, "constructor"):
			curPhase = 3
		case last.Success:
			curPhase = 4
		}
	}
	for i, p := range phases {
		role := p
		status := "normal"
		if i == curPhase {
			status = "active"
			if curPhase < 4 && len(st.History) > 0 && !st.History[len(st.History)-1].Success {
				status = "error"
			}
		}
		prims = append(prims, fw.PrimNode(phaseIDs[i], p, status, role))
	}
	for i := 0; i < 4; i++ {
		anim := ""
		if i < curPhase {
			anim = "flow"
		}
		prims = append(prims, fw.PrimEdge(fmt.Sprintf("pp-%d", i), phaseIDs[i], phaseIDs[i+1], "solid", anim))
	}

	// 6) 状态参数
	prims = append(prims, fw.PrimCodeBlock("cb-status",
		fmt.Sprintf("EOA = %d\n已部署合约 = %d (含销毁)\nCREATE2 collision 拦截 = %d\nconstructor REVERT = %d\ngas/balance 失败 = %d",
			len(st.Accounts), len(st.Contracts),
			st.CollisionBlocks, st.ConstructorReverts, st.GasFailures),
		"text", nil, 6))

	// 7) 账户表
	aLines := []string{"EOA                                          nonce  balance"}
	for _, addr := range accAddrs {
		acc := st.Accounts[addr]
		aLines = append(aLines, fmt.Sprintf("%s   %-5d  %d", addr, acc.Nonce, acc.Balance))
	}
	prims = append(prims, fw.PrimCodeBlock("cb-accounts", strings.Join(aLines, "\n"), "text", nil, 6))

	// 8) 合约表
	if len(cAddrs) > 0 {
		cLines := []string{"address (10/40)  type      template  balance  rt    salt(8)   nonce@birth  destroyed"}
		for _, addr := range cAddrs {
			c := st.Contracts[addr]
			salt := "-"
			if c.Salt != "" {
				saltShort := strings.TrimPrefix(c.Salt, "0x")
				if len(saltShort) > 8 {
					saltShort = saltShort[:8]
				}
				salt = saltShort
			}
			cLines = append(cLines, fmt.Sprintf("%s  %-9s  %-8s  %-7d  %-4d  %-8s  %-11d  %v",
				addr[:10]+"…", c.CreationType, c.TemplateName, c.Balance, len(c.RuntimeCode),
				salt, c.NonceAtBirth, c.Destroyed))
		}
		prims = append(prims, fw.PrimCodeBlock("cb-contracts", strings.Join(cLines, "\n"), "text", nil, 16))
	}

	// 9) 部署历史
	if len(st.History) > 0 {
		hLines := []string{"# tick ctype     creator        tpl       nonce#  salt(8)   addr           gas       value  ok  reason"}
		startIdx := 0
		if len(st.History) > 12 {
			startIdx = len(st.History) - 12
		}
		for i, r := range st.History[startIdx:] {
			ok := "✓"
			if !r.Success {
				ok = "✗"
			}
			salt := "-"
			if r.Salt != "" {
				ss := strings.TrimPrefix(r.Salt, "0x")
				if len(ss) > 8 {
					ss = ss[:8]
				}
				salt = ss
			}
			addrShort := "-"
			if r.DerivedAddr != "" {
				addrShort = r.DerivedAddr[:10] + "…"
			}
			hLines = append(hLines, fmt.Sprintf("%-2d %-4d %-9s %s %-8s %-6d  %-8s  %-13s  %-9d %-5d  %s   %s",
				i+startIdx, r.Tick, r.CreationType, r.Creator[:10]+"…", r.TemplateName,
				r.NonceBefore, salt, addrShort, r.GasUsed, r.Value, ok, r.RevertReason))
		}
		prims = append(prims, fw.PrimCodeBlock("cb-history", strings.Join(hLines, "\n"), "text", nil, 16))
	}

	// 10) 进度条（拦截统计）
	prims = append(prims, fw.PrimBar("bar-collision", float64(st.CollisionBlocks), 0, "success", "CREATE2 Collision Blocked"))
	prims = append(prims, fw.PrimBar("bar-ctor", float64(st.ConstructorReverts), 0, "warning", "Constructor REVERT"))
	prims = append(prims, fw.PrimBar("bar-gas", float64(st.GasFailures), 0, "danger", "Gas/Balance Failures"))

	// 11) 动效
	if len(st.History) > 0 {
		last := st.History[len(st.History)-1]
		if last.Success {
			id := "c-" + last.DerivedAddr[:10]
			prims = append(prims, fw.PrimBurst("burst-deploy", id, "success", int64(st.Tick), 700))
			prims = append(prims, fw.PrimGlow("glow-new", id, "success", 0.9))
		} else {
			prims = append(prims, fw.PrimShake("shake-fail", "deploy-pipeline", 0.4, 700))
			prims = append(prims, fw.PrimPulse("pulse-fail", "cb-history", "danger", 1500))
		}
	}

	// 12) 联动徽章
	prims = append(prims, fw.PrimLinkIndicator("link-sec", linkGroupContractSec, "idle", ""))

	if st.LastError != "" {
		prims = append(prims, fw.PrimErrorOverlay("err", "warning", "Deployment 错误", st.LastError, "scene", "请检查参数", true))
	}

	return fw.RenderEnvelope{
		Primitives:     prims,
		IsFullSnapshot: fullSnapshot,
		ChangedKeys:    []string{reason},
		Data:           buildSidePanelData(st, summary),
		ContainerData: []fw.ContainerMetric{
			{SourceContainer: "geth", MetricKey: "deploy.contract_count", Value: len(st.Contracts), TargetPrimitive: "cb-deploy", TargetParam: "count"},
		},
	}
}

func buildSidePanelData(st snapState, summary string) map[string]any {
	d := map[string]any{
		"eoa_count":           len(st.Accounts),
		"contract_count":      len(st.Contracts),
		"collision_blocks":    st.CollisionBlocks,
		"constructor_reverts": st.ConstructorReverts,
		"gas_failures":        st.GasFailures,
		"history_count":       len(st.History),
		"tick":                st.Tick,
	}
	if len(st.History) > 0 {
		last := st.History[len(st.History)-1]
		d["last_address"] = last.DerivedAddr
		d["last_success"] = last.Success
		d["last_ctype"] = last.CreationType
	}
	if summary != "" {
		d["summary"] = summary
	}
	return d
}

// =====================================================================
// MicroStep
// =====================================================================

func appendDeployMicroSteps(env *fw.RenderEnvelope, rec deployRecord) {
	steps := []fw.MicroStep{
		{ID: "d-1", Label: "gas / 余额检查", DurationMs: 400, HighlightIDs: []string{"ph-gas", "cb-accounts"}},
	}
	if !rec.Success && (strings.Contains(rec.RevertReason, "gas") || strings.Contains(rec.RevertReason, "余额")) {
		steps = append(steps, fw.MicroStep{ID: "d-fail", Label: "✗ " + rec.RevertReason, DurationMs: 500,
			HighlightIDs: []string{"bar-gas"}, FirePrimitives: []string{"shake-fail", "pulse-fail"}, IsLinkTrigger: true})
		env.MicroSteps = steps
		return
	}
	formulaID := "formula-create"
	if rec.CreationType == createTypeCREATE2 {
		formulaID = "formula-create2"
	}
	steps = append(steps, fw.MicroStep{ID: "d-2",
		Label:      fmt.Sprintf("地址推导 → %s", shortOr(rec.DerivedAddr)),
		DurationMs: 500, HighlightIDs: []string{"ph-derive", formulaID}})
	steps = append(steps, fw.MicroStep{ID: "d-3", Label: "collision check（账户已有 code 否？）",
		DurationMs: 400, HighlightIDs: []string{"ph-collide", "cb-contracts"}})
	if !rec.Success && strings.Contains(rec.RevertReason, "占用") {
		steps = append(steps, fw.MicroStep{ID: "d-collide", Label: "✗ 地址已被占用，REVERT",
			DurationMs: 500, HighlightIDs: []string{"bar-collision"},
			FirePrimitives: []string{"shake-fail", "pulse-fail"}, IsLinkTrigger: true})
		env.MicroSteps = steps
		return
	}
	steps = append(steps, fw.MicroStep{ID: "d-4", Label: "执行 constructor，初始化 storage",
		DurationMs: 500, HighlightIDs: []string{"ph-ctor"}})
	if !rec.Success && strings.Contains(rec.RevertReason, "constructor") {
		steps = append(steps, fw.MicroStep{ID: "d-ctor", Label: "✗ constructor REVERT",
			DurationMs: 500, HighlightIDs: []string{"bar-ctor"},
			FirePrimitives: []string{"shake-fail", "pulse-fail"}, IsLinkTrigger: true})
		env.MicroSteps = steps
		return
	}
	steps = append(steps, fw.MicroStep{ID: "d-5", Label: "写入 runtime code，nonce++",
		DurationMs: 500, HighlightIDs: []string{"ph-final", "cb-accounts", "cb-contracts"},
		FirePrimitives: []string{"glow-new", "burst-deploy"}, IsLinkTrigger: true})
	env.MicroSteps = steps
}

func appendPredictMicroSteps(env *fw.RenderEnvelope, ctype, derived, rlpHex, initHashHex, preimageHex string) {
	if ctype == createTypeCREATE {
		env.MicroSteps = []fw.MicroStep{
			{ID: "p-1", Label: "构造 RLP([sender, nonce])", DurationMs: 400, HighlightIDs: []string{"formula-create"}},
			{ID: "p-2", Label: "RLP = " + shortOr(rlpHex), DurationMs: 400, HighlightIDs: []string{"formula-create"}},
			{ID: "p-3", Label: "keccak256(rlp)[12:] = " + shortOr(derived), DurationMs: 500, HighlightIDs: []string{"cb-status"}, IsLinkTrigger: true},
		}
	} else {
		env.MicroSteps = []fw.MicroStep{
			{ID: "p-1", Label: "keccak256(init_code) = " + shortOr(initHashHex), DurationMs: 400, HighlightIDs: []string{"formula-create2"}},
			{ID: "p-2", Label: "preimage = 0xff ++ sender ++ salt ++ init_hash = " + shortOr(preimageHex), DurationMs: 500, HighlightIDs: []string{"formula-create2"}},
			{ID: "p-3", Label: "keccak256(preimage)[12:] = " + shortOr(derived), DurationMs: 500, HighlightIDs: []string{"cb-status"}, IsLinkTrigger: true},
		}
	}
}

func appendReplayDemoMicroSteps(env *fw.RenderEnvelope, rec1, rec2 deployRecord) {
	env.MicroSteps = []fw.MicroStep{
		{ID: "r-1", Label: "第一次 CREATE2 部署：" + shortOr(rec1.DerivedAddr), DurationMs: 500,
			HighlightIDs: []string{"ph-derive", "cb-contracts"}, FirePrimitives: []string{"glow-new"}},
		{ID: "r-2", Label: "再次部署相同 (sender, salt, init_code)", DurationMs: 400,
			HighlightIDs: []string{"ph-collide", "formula-create2"}},
		{ID: "r-3", Label: "✓ collision 拦截：" + rec2.RevertReason, DurationMs: 500,
			HighlightIDs:   []string{"bar-collision", "cb-history"},
			FirePrimitives: []string{"shake-fail", "pulse-fail"}, IsLinkTrigger: true},
	}
}

func shortOr(s string) string {
	if len(s) <= 14 {
		return s
	}
	return s[:10] + "…" + s[len(s)-4:]
}

// =====================================================================
// 联动
// =====================================================================

func deployLastAddr(st snapState) string {
	if len(st.History) == 0 {
		return ""
	}
	return st.History[len(st.History)-1].DerivedAddr
}

func publishOwnerSubtree(env *fw.RenderEnvelope, st snapState) {
	if env.Data == nil {
		env.Data = map[string]any{}
	}
	env.LinkTriggers = append(env.LinkTriggers, fw.LinkTrigger{
		ID:             "deploy-publish",
		SourceScene:    sceneCode,
		SourceAction:   "publish_deploy",
		LinkGroup:      linkGroupContractSec,
		ChangedFields:  []string{"contract.deploy.last_address"},
		Payload:        map[string]any{"last_address": deployLastAddr(st)},
		SourceAnchorID: "deploy-anchor",
		TargetAnchorID: "contract-sec-anchor",
	})
	env.ChangedKeys = append(env.ChangedKeys, "contract.deploy.last_address")
	env.Data["link_owner_subtree"] = linkOwnerSubtree
}

func ownerDiff(st snapState) map[string]any {
	d := map[string]any{
		"contract_count":      len(st.Contracts),
		"collision_blocks":    st.CollisionBlocks,
		"constructor_reverts": st.ConstructorReverts,
		"gas_failures":        st.GasFailures,
		"history_count":       len(st.History),
	}
	if len(st.History) > 0 {
		last := st.History[len(st.History)-1]
		d["last_address"] = last.DerivedAddr
		d["last_success"] = last.Success
		d["last_ctype"] = last.CreationType
	}
	return map[string]any{"contract": map[string]any{"deploy": d}}
}

// =====================================================================
// 工具
// =====================================================================

func intFromAny(v any) int {
	switch t := v.(type) {
	case float64:
		return int(t)
	case int:
		return t
	case int64:
		return int(t)
	}
	return 0
}
