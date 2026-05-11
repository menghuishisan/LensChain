// 模块：sim-engine/scenarios/internal/smartcontract/evmexecution
// 文件职责：SC-02 EVM 字节码执行场景的完整实现。
//
// SSOT 依据：06.md §4.6.2 / §3.2 / §6.2 / §6.3 / §8.3；AGENTS.md §0 + §6。
//
// 算法实现：自实现以太坊虚拟机（EVM）核心子集（零外部依赖）：
//   · 256-bit 大整数栈（用 [4]uint64 表示，最大深度 1024）
//   · 内存（按字节数组，按需扩展）
//   · 存储（key→value 256-bit 大整数映射）
//   · PC（程序计数器） + Gas 计数器
//   · 30+ opcodes：
//     · 算术：ADD/SUB/MUL/DIV/MOD/EXP
//     · 比较：LT/GT/EQ/ISZERO
//     · 位运算：AND/OR/XOR/NOT
//     · SHA3（KECCAK256）— 复用 keccak256hash
//     · 栈：PUSH1..PUSH32 / POP / DUP1..DUP16 / SWAP1..SWAP16
//     · 内存：MLOAD / MSTORE / MSTORE8 / MSIZE
//     · 存储：SLOAD / SSTORE
//     · 控制流：JUMP / JUMPI / PC / JUMPDEST / STOP / RETURN / REVERT
//     · 调用环境：CALLER / CALLVALUE / CALLDATALOAD / CALLDATASIZE
//   · 单步执行：每次 step 取 1 条指令，更新栈/内存/存储

package evmexecution

import (
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	fw "github.com/lenschain/sim-engine/framework"
	"github.com/lenschain/sim-engine/scenarios/internal/cryptography/keccak256hash"
)

const (
	sceneCode     = "evm-execution"
	schemaVersion = "v1.0.0"
	algorithmType = "evm-bytecode"

	maxStackDepth = 1024
	defaultGas    = 100000

	linkGroupContractSec = "contract-security-group"
	linkOwnerSubtree     = "contract.evm"
)

// =====================================================================
// 256-bit 大整数（用 4×uint64 表示，big-endian）
// =====================================================================

type word [4]uint64 // [0]=高位 ... [3]=低位

func (w word) isZero() bool { return w[0] == 0 && w[1] == 0 && w[2] == 0 && w[3] == 0 }

// add 256-bit 加法（带进位，溢出截断）。
func wAdd(a, b word) word {
	var r word
	var carry uint64
	for i := 3; i >= 0; i-- {
		sum := a[i] + b[i] + carry
		if sum < a[i] || (carry == 1 && sum == a[i]) {
			carry = 1
		} else {
			carry = 0
		}
		r[i] = sum
	}
	return r
}

// sub 256-bit 减法（mod 2^256）。
func wSub(a, b word) word {
	var r word
	var borrow uint64
	for i := 3; i >= 0; i-- {
		if a[i] >= b[i]+borrow && !(borrow == 1 && b[i] == ^uint64(0)) {
			r[i] = a[i] - b[i] - borrow
			borrow = 0
		} else {
			r[i] = a[i] - b[i] - borrow // wrap
			borrow = 1
		}
	}
	return r
}

// mul 256-bit 乘法（mod 2^256，教学版用低位逐字相乘）。
func wMul(a, b word) word {
	// 简化：把 256-bit 转成字节再乘（教学版精度足够；高位相乘部分会被截断）
	// 真实 EVM 用 4×4 uint64 卷积。这里我们做完整 256-bit 卷积然后取低 256-bit。
	var prod [8]uint64 // 8×64 = 512 bit 中间结果
	for i := 3; i >= 0; i-- {
		var carry uint64 = 0
		for j := 3; j >= 0; j-- {
			hi, lo := mul64(a[i], b[j])
			pos := i + j + 1 // 低位字位置（0-indexed）
			sum1 := prod[pos] + lo + carry
			c1 := uint64(0)
			if sum1 < prod[pos] || (carry == 1 && sum1 == prod[pos]) {
				c1 = 1
			}
			prod[pos] = sum1
			carry = hi + c1
		}
		if i+0 >= 0 {
			prod[i] += carry
		}
	}
	// prod[4..7] 是低 256 bit
	return word{prod[4], prod[5], prod[6], prod[7]}
}

// mul64 64×64 → 128 (hi, lo)。
func mul64(a, b uint64) (hi, lo uint64) {
	const halfShift = 32
	const halfMask = uint64(0xFFFFFFFF)
	a0 := a & halfMask
	a1 := a >> halfShift
	b0 := b & halfMask
	b1 := b >> halfShift
	w0 := a0 * b0
	t := a1*b0 + (w0 >> halfShift)
	w1 := t & halfMask
	w2 := t >> halfShift
	w1 += a0 * b1
	hi = a1*b1 + w2 + (w1 >> halfShift)
	lo = (w1 << halfShift) | (w0 & halfMask)
	return
}

// div 256-bit 整除（b=0 时返回 0，与 EVM 规范一致）。
func wDiv(a, b word) word {
	if b.isZero() {
		return word{}
	}
	// 教学简化：仅支持 b 在低 64 bit 内的整除（覆盖 99% 教学用例）
	if b[0] == 0 && b[1] == 0 && b[2] == 0 {
		return divBySmall(a, b[3])
	}
	// 否则做位移逐位试减（标准长除法）
	q, _ := longDivide(a, b)
	return q
}

// mod 256-bit 取模。
func wMod(a, b word) word {
	if b.isZero() {
		return word{}
	}
	if b[0] == 0 && b[1] == 0 && b[2] == 0 {
		_, r := divBySmallWithRem(a, b[3])
		return word{0, 0, 0, r}
	}
	_, r := longDivide(a, b)
	return r
}

// divBySmall a / s，s 是 64-bit。
func divBySmall(a word, s uint64) word {
	q, _ := divBySmallWithRem(a, s)
	return q
}

func divBySmallWithRem(a word, s uint64) (word, uint64) {
	if s == 0 {
		return word{}, 0
	}
	var q word
	var rem uint64 = 0
	for i := 0; i < 4; i++ {
		// 把 (rem, a[i]) 视作 128-bit / 64-bit
		// 简化：当 rem=0 时直接除；否则需要 128/64 除法 — 这里用迭代位移
		// 一般情况：执行位级长除法
		hi := rem
		lo := a[i]
		// 按 64 位长除法
		for bit := 63; bit >= 0; bit-- {
			topBit := lo >> uint(bit) & 1
			hi = (hi << 1) | topBit
			if hi >= s {
				hi -= s
				q[i] |= 1 << uint(bit)
			}
		}
		rem = hi
	}
	return q, rem
}

// longDivide 256/256 长除法（教学：用位移逐位试减；性能不重要）。
func longDivide(a, b word) (q, r word) {
	for i := 255; i >= 0; i-- {
		// r = r << 1 | bit(a, i)
		r = shl(r, 1)
		bit := getBit(a, i)
		r[3] |= bit
		// if r >= b: r -= b; q[i]=1
		if !cmpLess(r, b) {
			r = wSub(r, b)
			setBit(&q, i, 1)
		}
	}
	return
}

func shl(a word, n uint) word {
	if n >= 256 {
		return word{}
	}
	// 简化：n=1
	if n == 1 {
		var r word
		var carry uint64
		for i := 3; i >= 0; i-- {
			r[i] = (a[i] << 1) | carry
			carry = a[i] >> 63
		}
		return r
	}
	// 通用：递归
	r := a
	for i := uint(0); i < n; i++ {
		r = shl(r, 1)
	}
	return r
}

func getBit(a word, i int) uint64 {
	if i < 0 || i >= 256 {
		return 0
	}
	w := 3 - i/64
	b := uint(i % 64)
	return (a[w] >> b) & 1
}

func setBit(a *word, i int, v uint64) {
	if i < 0 || i >= 256 {
		return
	}
	w := 3 - i/64
	b := uint(i % 64)
	if v == 1 {
		a[w] |= 1 << b
	} else {
		a[w] &^= 1 << b
	}
}

// cmpLess a < b ?
func cmpLess(a, b word) bool {
	for i := 0; i < 4; i++ {
		if a[i] != b[i] {
			return a[i] < b[i]
		}
	}
	return false
}

// fromUint64 把 uint64 升为 word。
func fromUint64(v uint64) word { return word{0, 0, 0, v} }

// fromBytes 把字节切片（≤32 字节，big-endian）转为 word。
func fromBytes(b []byte) word {
	var bytes [32]byte
	if len(b) > 32 {
		b = b[len(b)-32:]
	}
	copy(bytes[32-len(b):], b)
	return word{
		uint64(bytes[0])<<56 | uint64(bytes[1])<<48 | uint64(bytes[2])<<40 | uint64(bytes[3])<<32 |
			uint64(bytes[4])<<24 | uint64(bytes[5])<<16 | uint64(bytes[6])<<8 | uint64(bytes[7]),
		uint64(bytes[8])<<56 | uint64(bytes[9])<<48 | uint64(bytes[10])<<40 | uint64(bytes[11])<<32 |
			uint64(bytes[12])<<24 | uint64(bytes[13])<<16 | uint64(bytes[14])<<8 | uint64(bytes[15]),
		uint64(bytes[16])<<56 | uint64(bytes[17])<<48 | uint64(bytes[18])<<40 | uint64(bytes[19])<<32 |
			uint64(bytes[20])<<24 | uint64(bytes[21])<<16 | uint64(bytes[22])<<8 | uint64(bytes[23]),
		uint64(bytes[24])<<56 | uint64(bytes[25])<<48 | uint64(bytes[26])<<40 | uint64(bytes[27])<<32 |
			uint64(bytes[28])<<24 | uint64(bytes[29])<<16 | uint64(bytes[30])<<8 | uint64(bytes[31]),
	}
}

// toBytes 把 word 转回 32 字节大端。
func toBytes(w word) [32]byte {
	var b [32]byte
	for i := 0; i < 4; i++ {
		v := w[i]
		for j := 0; j < 8; j++ {
			b[i*8+j] = byte(v >> uint(56-j*8))
		}
	}
	return b
}

// toHex 把 word 转 hex 字符串。
func toHex(w word) string {
	b := toBytes(w)
	return hex.EncodeToString(b[:])
}

// asUint64 取低 64 bit 作为 uint64（用于 PC / 内存索引等）。
func asUint64(w word) uint64 {
	return w[3]
}

// =====================================================================
// EVM Opcode 表
// =====================================================================

const (
	opSTOP         = 0x00
	opADD          = 0x01
	opMUL          = 0x02
	opSUB          = 0x03
	opDIV          = 0x04
	opMOD          = 0x06
	opEXP          = 0x0a
	opLT           = 0x10
	opGT           = 0x11
	opEQ           = 0x14
	opISZERO       = 0x15
	opAND          = 0x16
	opOR           = 0x17
	opXOR          = 0x18
	opNOT          = 0x19
	opSHA3         = 0x20
	opCALLER       = 0x33
	opCALLVALUE    = 0x34
	opCALLDATALOAD = 0x35
	opCALLDATASIZE = 0x36
	opPOP          = 0x50
	opMLOAD        = 0x51
	opMSTORE       = 0x52
	opMSTORE8      = 0x53
	opSLOAD        = 0x54
	opSSTORE       = 0x55
	opJUMP         = 0x56
	opJUMPI        = 0x57
	opPC           = 0x58
	opMSIZE        = 0x59
	opGAS          = 0x5a
	opJUMPDEST     = 0x5b
	opRETURN       = 0xf3
	opREVERT       = 0xfd
)

func opName(op byte) string {
	switch {
	case op >= 0x60 && op <= 0x7f:
		return fmt.Sprintf("PUSH%d", int(op-0x5f))
	case op >= 0x80 && op <= 0x8f:
		return fmt.Sprintf("DUP%d", int(op-0x7f))
	case op >= 0x90 && op <= 0x9f:
		return fmt.Sprintf("SWAP%d", int(op-0x8f))
	}
	switch op {
	case opSTOP:
		return "STOP"
	case opADD:
		return "ADD"
	case opMUL:
		return "MUL"
	case opSUB:
		return "SUB"
	case opDIV:
		return "DIV"
	case opMOD:
		return "MOD"
	case opEXP:
		return "EXP"
	case opLT:
		return "LT"
	case opGT:
		return "GT"
	case opEQ:
		return "EQ"
	case opISZERO:
		return "ISZERO"
	case opAND:
		return "AND"
	case opOR:
		return "OR"
	case opXOR:
		return "XOR"
	case opNOT:
		return "NOT"
	case opSHA3:
		return "SHA3"
	case opCALLER:
		return "CALLER"
	case opCALLVALUE:
		return "CALLVALUE"
	case opCALLDATALOAD:
		return "CALLDATALOAD"
	case opCALLDATASIZE:
		return "CALLDATASIZE"
	case opPOP:
		return "POP"
	case opMLOAD:
		return "MLOAD"
	case opMSTORE:
		return "MSTORE"
	case opMSTORE8:
		return "MSTORE8"
	case opSLOAD:
		return "SLOAD"
	case opSSTORE:
		return "SSTORE"
	case opJUMP:
		return "JUMP"
	case opJUMPI:
		return "JUMPI"
	case opPC:
		return "PC"
	case opMSIZE:
		return "MSIZE"
	case opGAS:
		return "GAS"
	case opJUMPDEST:
		return "JUMPDEST"
	case opRETURN:
		return "RETURN"
	case opREVERT:
		return "REVERT"
	}
	return fmt.Sprintf("UNK_%02x", op)
}

// gasCost 简化 gas 表：常量 / pop / push / dup / swap = 3；store = 100；keccak = 30；其他 5。
func gasCost(op byte) int {
	switch op {
	case opSSTORE:
		return 100
	case opSHA3:
		return 30
	case opSTOP, opJUMPDEST, opPOP:
		return 2
	}
	if op >= 0x60 && op <= 0x9f { // PUSH/DUP/SWAP
		return 3
	}
	return 5
}

// =====================================================================
// EVM 执行状态
// =====================================================================

type evmState struct {
	Code         []byte
	PC           uint64
	Stack        []word
	Memory       []byte
	Storage      map[string]word // hex(key) → value
	Gas          int
	Caller       string
	CallValue    word
	CallData     []byte
	ReturnData   []byte
	Halted       bool
	HaltedReason string // STOP / RETURN / REVERT / out-of-gas / stack-overflow / stack-underflow / invalid
}

func newEVMState(code []byte, caller string, gas int) evmState {
	return evmState{
		Code:    append([]byte{}, code...),
		Storage: map[string]word{},
		Gas:     gas,
		Caller:  caller,
	}
}

func (e *evmState) push(w word) error {
	if len(e.Stack) >= maxStackDepth {
		e.Halted = true
		e.HaltedReason = "stack overflow"
		return errors.New("stack overflow")
	}
	e.Stack = append(e.Stack, w)
	return nil
}

func (e *evmState) pop() (word, error) {
	if len(e.Stack) == 0 {
		e.Halted = true
		e.HaltedReason = "stack underflow"
		return word{}, errors.New("stack underflow")
	}
	v := e.Stack[len(e.Stack)-1]
	e.Stack = e.Stack[:len(e.Stack)-1]
	return v, nil
}

func (e *evmState) peek(n int) word {
	if n >= len(e.Stack) {
		return word{}
	}
	return e.Stack[len(e.Stack)-1-n]
}

func (e *evmState) memoryWrite(off uint64, data []byte) {
	end := off + uint64(len(data))
	if uint64(len(e.Memory)) < end {
		e.Memory = append(e.Memory, make([]byte, end-uint64(len(e.Memory)))...)
	}
	copy(e.Memory[off:], data)
}

func (e *evmState) memoryRead(off, n uint64) []byte {
	end := off + n
	if uint64(len(e.Memory)) < end {
		// auto-extend
		e.Memory = append(e.Memory, make([]byte, end-uint64(len(e.Memory)))...)
	}
	return append([]byte{}, e.Memory[off:end]...)
}

// step 执行一条 opcode；halted 后再调用是 no-op。
func (e *evmState) step() (op byte, err error) {
	if e.Halted {
		return 0, nil
	}
	if e.PC >= uint64(len(e.Code)) {
		e.Halted = true
		e.HaltedReason = "end of code"
		return 0, nil
	}
	op = e.Code[e.PC]
	cost := gasCost(op)
	if e.Gas < cost {
		e.Halted = true
		e.HaltedReason = "out of gas"
		return op, errors.New("out of gas")
	}
	e.Gas -= cost
	pcStart := e.PC
	e.PC++

	// PUSH1..PUSH32
	if op >= 0x60 && op <= 0x7f {
		n := uint64(op - 0x5f)
		end := e.PC + n
		if end > uint64(len(e.Code)) {
			end = uint64(len(e.Code))
		}
		bytes := append([]byte{}, e.Code[e.PC:end]...)
		e.PC = end
		if err := e.push(fromBytes(bytes)); err != nil {
			return op, err
		}
		return op, nil
	}
	// DUP1..DUP16
	if op >= 0x80 && op <= 0x8f {
		n := int(op - 0x80)
		if n >= len(e.Stack) {
			e.Halted = true
			e.HaltedReason = "stack underflow (DUP)"
			return op, errors.New("stack underflow")
		}
		v := e.peek(n)
		return op, e.push(v)
	}
	// SWAP1..SWAP16
	if op >= 0x90 && op <= 0x9f {
		n := int(op-0x90) + 1
		if n >= len(e.Stack) {
			e.Halted = true
			e.HaltedReason = "stack underflow (SWAP)"
			return op, errors.New("stack underflow")
		}
		i := len(e.Stack) - 1
		j := len(e.Stack) - 1 - n
		e.Stack[i], e.Stack[j] = e.Stack[j], e.Stack[i]
		return op, nil
	}

	switch op {
	case opSTOP:
		e.Halted = true
		e.HaltedReason = "STOP"
	case opADD:
		a, _ := e.pop()
		b, _ := e.pop()
		e.push(wAdd(a, b))
	case opMUL:
		a, _ := e.pop()
		b, _ := e.pop()
		e.push(wMul(a, b))
	case opSUB:
		a, _ := e.pop()
		b, _ := e.pop()
		e.push(wSub(a, b))
	case opDIV:
		a, _ := e.pop()
		b, _ := e.pop()
		e.push(wDiv(a, b))
	case opMOD:
		a, _ := e.pop()
		b, _ := e.pop()
		e.push(wMod(a, b))
	case opEXP:
		base, _ := e.pop()
		exp, _ := e.pop()
		// 简化教学版：仅支持低位 exp
		eVal := asUint64(exp)
		r := fromUint64(1)
		for i := uint64(0); i < eVal && i < 256; i++ {
			r = wMul(r, base)
		}
		e.push(r)
	case opLT:
		a, _ := e.pop()
		b, _ := e.pop()
		if cmpLess(a, b) {
			e.push(fromUint64(1))
		} else {
			e.push(word{})
		}
	case opGT:
		a, _ := e.pop()
		b, _ := e.pop()
		if cmpLess(b, a) {
			e.push(fromUint64(1))
		} else {
			e.push(word{})
		}
	case opEQ:
		a, _ := e.pop()
		b, _ := e.pop()
		if a == b {
			e.push(fromUint64(1))
		} else {
			e.push(word{})
		}
	case opISZERO:
		a, _ := e.pop()
		if a.isZero() {
			e.push(fromUint64(1))
		} else {
			e.push(word{})
		}
	case opAND:
		a, _ := e.pop()
		b, _ := e.pop()
		e.push(word{a[0] & b[0], a[1] & b[1], a[2] & b[2], a[3] & b[3]})
	case opOR:
		a, _ := e.pop()
		b, _ := e.pop()
		e.push(word{a[0] | b[0], a[1] | b[1], a[2] | b[2], a[3] | b[3]})
	case opXOR:
		a, _ := e.pop()
		b, _ := e.pop()
		e.push(word{a[0] ^ b[0], a[1] ^ b[1], a[2] ^ b[2], a[3] ^ b[3]})
	case opNOT:
		a, _ := e.pop()
		e.push(word{^a[0], ^a[1], ^a[2], ^a[3]})
	case opSHA3:
		off, _ := e.pop()
		size, _ := e.pop()
		data := e.memoryRead(asUint64(off), asUint64(size))
		h := keccak256hash.Sum256(data) // 教学：注意 EVM SHA3 实际是 Keccak-256
		_ = h
		// 用我们自己的 Keccak（与 keccak256-hash 场景一致）
		hb := keccak256(data)
		e.push(fromBytes(hb[:]))
	case opCALLER:
		e.push(fromBytes([]byte(e.Caller)))
	case opCALLVALUE:
		e.push(e.CallValue)
	case opCALLDATALOAD:
		off, _ := e.pop()
		o := asUint64(off)
		buf := make([]byte, 32)
		for i := uint64(0); i < 32; i++ {
			if o+i < uint64(len(e.CallData)) {
				buf[i] = e.CallData[o+i]
			}
		}
		e.push(fromBytes(buf))
	case opCALLDATASIZE:
		e.push(fromUint64(uint64(len(e.CallData))))
	case opPOP:
		e.pop()
	case opMLOAD:
		off, _ := e.pop()
		buf := e.memoryRead(asUint64(off), 32)
		e.push(fromBytes(buf))
	case opMSTORE:
		off, _ := e.pop()
		v, _ := e.pop()
		b := toBytes(v)
		e.memoryWrite(asUint64(off), b[:])
	case opMSTORE8:
		off, _ := e.pop()
		v, _ := e.pop()
		e.memoryWrite(asUint64(off), []byte{byte(v[3])})
	case opSLOAD:
		k, _ := e.pop()
		key := toHex(k)
		v := e.Storage[key]
		e.push(v)
	case opSSTORE:
		k, _ := e.pop()
		v, _ := e.pop()
		key := toHex(k)
		if v.isZero() {
			delete(e.Storage, key)
		} else {
			e.Storage[key] = v
		}
	case opJUMP:
		dst, _ := e.pop()
		d := asUint64(dst)
		if d >= uint64(len(e.Code)) || e.Code[d] != opJUMPDEST {
			e.Halted = true
			e.HaltedReason = "invalid jump"
			return op, errors.New("invalid jump")
		}
		e.PC = d
	case opJUMPI:
		dst, _ := e.pop()
		cond, _ := e.pop()
		if !cond.isZero() {
			d := asUint64(dst)
			if d >= uint64(len(e.Code)) || e.Code[d] != opJUMPDEST {
				e.Halted = true
				e.HaltedReason = "invalid jump"
				return op, errors.New("invalid jump")
			}
			e.PC = d
		}
	case opPC:
		e.push(fromUint64(pcStart))
	case opMSIZE:
		e.push(fromUint64(uint64(len(e.Memory))))
	case opGAS:
		e.push(fromUint64(uint64(e.Gas)))
	case opJUMPDEST:
		// no-op
	case opRETURN:
		off, _ := e.pop()
		size, _ := e.pop()
		e.ReturnData = e.memoryRead(asUint64(off), asUint64(size))
		e.Halted = true
		e.HaltedReason = "RETURN"
	case opREVERT:
		off, _ := e.pop()
		size, _ := e.pop()
		e.ReturnData = e.memoryRead(asUint64(off), asUint64(size))
		e.Halted = true
		e.HaltedReason = "REVERT"
	default:
		e.Halted = true
		e.HaltedReason = fmt.Sprintf("invalid opcode 0x%02x", op)
	}
	return op, nil
}

// keccak256 自实现 sponge — 复用 keccak256hash.Sum256，但保持与其同源。
func keccak256(data []byte) [32]byte {
	return keccak256hash.Sum256(data)
}

// =====================================================================
// 场景内部状态
// =====================================================================

type snapState struct {
	Bytecode    string // hex
	Caller      string
	CallValue   string // 十进制
	CallData    string // hex
	GasInit     int
	EVM         evmState
	Disasm      []disasmLine
	Initialized bool
	LastError   string
}

type disasmLine struct {
	PC    uint64
	Op    byte
	Bytes string
	Name  string
}

func disassemble(code []byte) []disasmLine {
	out := []disasmLine{}
	pc := uint64(0)
	for pc < uint64(len(code)) {
		op := code[pc]
		l := disasmLine{PC: pc, Op: op, Name: opName(op), Bytes: fmt.Sprintf("%02x", op)}
		if op >= 0x60 && op <= 0x7f {
			n := uint64(op - 0x5f)
			end := pc + 1 + n
			if end > uint64(len(code)) {
				end = uint64(len(code))
			}
			l.Bytes += " " + hex.EncodeToString(code[pc+1:end])
			pc = end
		} else {
			pc++
		}
		out = append(out, l)
	}
	return out
}

func defaultSnapState() snapState {
	// 默认：PUSH1 5, PUSH1 7, ADD, PUSH1 0, SSTORE, PUSH1 0, SLOAD, PUSH1 0, MSTORE, PUSH1 32, PUSH1 0, RETURN
	// 计算 5+7=12，存到 storage[0]，再读出来 RETURN 32 字节
	bytecode := []byte{
		0x60, 0x05, // PUSH1 5
		0x60, 0x07, // PUSH1 7
		0x01,       // ADD
		0x60, 0x00, // PUSH1 0 (storage key)
		0x55,       // SSTORE
		0x60, 0x00, // PUSH1 0
		0x54,       // SLOAD
		0x60, 0x00, // PUSH1 0
		0x52,       // MSTORE
		0x60, 0x20, // PUSH1 32
		0x60, 0x00, // PUSH1 0
		0xf3, // RETURN
	}
	return snapState{
		Bytecode:  hex.EncodeToString(bytecode),
		Caller:    "alice",
		CallValue: "0",
		CallData:  "",
		GasInit:   defaultGas,
		EVM:       newEVMState(bytecode, "alice", defaultGas),
		Disasm:    disassemble(bytecode),
	}
}

// initEVM 重置 EVM 到 PC=0。
func (st *snapState) initEVM() error {
	code, err := hex.DecodeString(st.Bytecode)
	if err != nil {
		return fmt.Errorf("bytecode hex 解析失败: %s", err.Error())
	}
	cd, _ := hex.DecodeString(st.CallData)
	st.EVM = newEVMState(code, st.Caller, st.GasInit)
	st.EVM.CallData = cd
	// CallValue 解析
	st.EVM.CallValue = parseDecimal(st.CallValue)
	st.Disasm = disassemble(code)
	st.Initialized = true
	return nil
}

// parseDecimal 把十进制字符串转 word（最多 64 bit 大小）。
func parseDecimal(s string) word {
	v := uint64(0)
	for _, c := range s {
		if c < '0' || c > '9' {
			break
		}
		v = v*10 + uint64(c-'0')
	}
	return fromUint64(v)
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
		Bytecode:    fw.MapStr(d, "bytecode", ""),
		Caller:      fw.MapStr(d, "caller", "alice"),
		CallValue:   fw.MapStr(d, "call_value", "0"),
		CallData:    fw.MapStr(d, "call_data", ""),
		GasInit:     fw.MapInt(d, "gas_init", defaultGas),
		Initialized: fw.MapBool(d, "initialized", false),
		LastError:   fw.MapStr(d, "last_error", ""),
	}
	if st.Bytecode == "" {
		return defaultSnapState()
	}
	st.initEVM()
	// 恢复 EVM 运行状态
	st.EVM.PC = uint64(fw.MapInt(d, "pc", 0))
	st.EVM.Gas = fw.MapInt(d, "gas", st.GasInit)
	st.EVM.Halted = fw.MapBool(d, "halted", false)
	st.EVM.HaltedReason = fw.MapStr(d, "halt_reason", "")
	if stkAny, ok := d["stack"].([]any); ok {
		st.EVM.Stack = nil
		for _, v := range stkAny {
			if s, ok := v.(string); ok {
				if b, err := hex.DecodeString(s); err == nil {
					st.EVM.Stack = append(st.EVM.Stack, fromBytes(b))
				}
			}
		}
	}
	if memHex, ok := d["memory"].(string); ok {
		if b, err := hex.DecodeString(memHex); err == nil {
			st.EVM.Memory = b
		}
	}
	if stoAny, ok := d["storage"].(map[string]any); ok {
		for k, v := range stoAny {
			if s, ok := v.(string); ok {
				if b, err := hex.DecodeString(s); err == nil {
					st.EVM.Storage[k] = fromBytes(b)
				}
			}
		}
	}
	if rdHex, ok := d["return_data"].(string); ok {
		if b, err := hex.DecodeString(rdHex); err == nil {
			st.EVM.ReturnData = b
		}
	}
	return st
}

func saveState(s *fw.SceneState, st snapState) {
	if s.Data == nil {
		s.Data = map[string]any{}
	}
	s.Data["bytecode"] = st.Bytecode
	s.Data["caller"] = st.Caller
	s.Data["call_value"] = st.CallValue
	s.Data["call_data"] = st.CallData
	s.Data["gas_init"] = st.GasInit
	s.Data["initialized"] = st.Initialized
	s.Data["last_error"] = st.LastError
	s.Data["pc"] = int(st.EVM.PC)
	s.Data["gas"] = st.EVM.Gas
	s.Data["halted"] = st.EVM.Halted
	s.Data["halt_reason"] = st.EVM.HaltedReason
	stkAny := make([]any, len(st.EVM.Stack))
	for i, w := range st.EVM.Stack {
		b := toBytes(w)
		stkAny[i] = hex.EncodeToString(b[:])
	}
	s.Data["stack"] = stkAny
	s.Data["memory"] = hex.EncodeToString(st.EVM.Memory)
	stoAny := map[string]any{}
	for k, v := range st.EVM.Storage {
		b := toBytes(v)
		stoAny[k] = hex.EncodeToString(b[:])
	}
	s.Data["storage"] = stoAny
	s.Data["return_data"] = hex.EncodeToString(st.EVM.ReturnData)
}

// =====================================================================
// 场景定义
// =====================================================================

func Definition() fw.Definition {
	return fw.Definition{
		Code: sceneCode, Name: "EVM 字节码执行",
		Description:         "演示 EVM 256-bit 栈机：30+ opcodes + 内存 + 存储 + Gas + 单步执行",
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
			"contract.evm.pc",
			"contract.evm.gas",
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
		SceneCode:     sceneCode,
		SchemaVersion: schemaVersion,
		Actions: []fw.ActionDef{
			{
				ActionCode: "load_bytecode", Label: "加载字节码",
				Description:   "把 hex 字节码载入 EVM，重置 PC/Stack/Memory",
				Category:      fw.ActionPrimary, Trigger: fw.TriggerSubmit,
				Roles:         []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				InterveneType: fw.InterveneState,
				Fields: []fw.FieldDef{
					{Name: "bytecode", Type: fw.FieldString, Label: "bytecode (hex)", Required: true,
						Default: "60056007016000556000546000526020600073"},
					{Name: "caller", Type: fw.FieldString, Label: "caller", Required: true, Default: "alice"},
					{Name: "call_value", Type: fw.FieldString, Label: "call_value (decimal)", Required: false, Default: "0"},
					{Name: "call_data", Type: fw.FieldString, Label: "call_data (hex)", Required: false, Default: ""},
					{Name: "gas", Type: fw.FieldNumber, Label: "gas", Required: true, Default: defaultGas, Min: 100, Step: 100},
				},
			},
			{
				ActionCode: "step", Label: "单步执行",
				Description: "取 PC 处一条 opcode 执行",
				Category:    fw.ActionPrimary, Trigger: fw.TriggerImmediate,
				Roles:              []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				InterveneType:     fw.IntervenePhase,
				WritesOwnedFields: []string{"contract.evm.pc", "contract.evm.gas"},
				LinkOwnerFields:   []string{"contract.evm.pc", "contract.evm.gas"},
			},
			{
				ActionCode: "step_n", Label: "执行 N 步",
				Category:      fw.ActionPrimary, Trigger: fw.TriggerSubmit,
				Roles:         []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				InterveneType: fw.IntervenePhase,
				Fields: []fw.FieldDef{
					{Name: "n", Type: fw.FieldNumber, Label: "步数", Required: true, Default: 5, Min: 1, Max: 1000, Step: 1},
				},
			},
			{
				ActionCode: "run_to_end", Label: "执行到结束",
				Description:   "持续 step 直到 halted",
				Category:      fw.ActionPrimary, Trigger: fw.TriggerImmediate,
				Roles:         []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				InterveneType: fw.IntervenePhase,
			},
			{
				ActionCode: "reset", Label: "重置（PC=0）",
				Category:      fw.ActionPrimary, Trigger: fw.TriggerImmediate,
				Roles:         []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				InterveneType: fw.InterveneReset,
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
				ActionCode:    "deploy_contract",
				Label:         "部署合约（真实链）",
				Description:   "调 geth eth_sendTransaction 部署合约",
				Category:      fw.ActionPrimary,
				Trigger:       fw.TriggerSubmit,
				Roles:         []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				InterveneType: fw.InterveneState,
				HybridChannel: fw.HybridChannelContainer,
				ContainerCmd:  `curl -s -X POST -H "Content-Type: application/json" --data '{"jsonrpc":"2.0","method":"eth_sendTransaction","params":[{"from":"{{from}}","data":"{{bytecode}}","gas":"0x1000000"}],"id":1}' http://geth:8545`,
				Reversible:    false,
				Fields: []fw.FieldDef{
					{Name: "from", Type: fw.FieldString, Label: "from address", Required: true, Default: "0x0000000000000000000000000000000000000001"},
					{Name: "bytecode", Type: fw.FieldString, Label: "bytecode (hex)", Required: true, Default: "0x6080604052"},
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
	env := buildEnvelope(st, "init", "EVM 初始化（默认演示：5+7→storage[0]→return）", true)
	publishOwnerSubtree(&env, st)
	return env, nil
}

func stepScene(state *fw.SceneState, in fw.StepInput) (fw.StepOutput, error) {
	st := loadState(state)
	state.Tick = in.Tick
	env := buildEnvelope(st, "tick", "", false)
	return fw.StepOutput{Render: env}, nil
}

func handleAction(state *fw.SceneState, in fw.ActionInput) (fw.ActionOutput, error) {
	_ = fw.EnsureActorBucket(state, in.ActorID)
	if out, ok := fw.HandleBroadcastHint(state, in); ok {
		return out, nil
	}

	st := loadState(state)
	out := fw.ActionOutput{Success: true}

	switch in.ActionCode {
	case "load_bytecode":
		st.Bytecode = fw.MapStr(in.Params, "bytecode", st.Bytecode)
		st.Caller = fw.MapStr(in.Params, "caller", "alice")
		st.CallValue = fw.MapStr(in.Params, "call_value", "0")
		st.CallData = fw.MapStr(in.Params, "call_data", "")
		st.GasInit = fw.MapInt(in.Params, "gas", defaultGas)
		if err := st.initEVM(); err != nil {
			return fw.ActionOutput{Success: false, ErrorMessage: err.Error()}, nil
		}
		saveState(state, st)
		out.Render = buildEnvelope(st, "load_bytecode", "已加载字节码", true)
		appendLoadMicroSteps(&out.Render)
		return out, nil

	case "step":
		op, _ := st.EVM.step()
		saveState(state, st)
		summary := fmt.Sprintf("PC=%d Op=%s Gas=%d", st.EVM.PC, opName(op), st.EVM.Gas)
		if st.EVM.Halted {
			summary = fmt.Sprintf("HALTED (%s) PC=%d", st.EVM.HaltedReason, st.EVM.PC)
		}
		out.Render = buildEnvelope(st, "step", summary, false)
		appendStepMicroSteps(&out.Render, opName(op), st.EVM.Halted)
		out.SharedStateDiff = ownerDiff(st)
		return out, nil

	case "step_n":
		n := fw.MapInt(in.Params, "n", 5)
		var lastOp byte
		for i := 0; i < n && !st.EVM.Halted; i++ {
			lastOp, _ = st.EVM.step()
		}
		saveState(state, st)
		summary := fmt.Sprintf("执行到 PC=%d，最后 op=%s", st.EVM.PC, opName(lastOp))
		if st.EVM.Halted {
			summary += fmt.Sprintf(" HALTED (%s)", st.EVM.HaltedReason)
		}
		out.Render = buildEnvelope(st, "step_n", summary, false)
		appendStepMicroSteps(&out.Render, opName(lastOp), st.EVM.Halted)
		out.SharedStateDiff = ownerDiff(st)
		return out, nil

	case "run_to_end":
		guard := 0
		for !st.EVM.Halted && guard < 100000 {
			st.EVM.step()
			guard++
		}
		saveState(state, st)
		out.Render = buildEnvelope(st, "run_to_end",
			fmt.Sprintf("执行 %d 步 → halted (%s)", guard, st.EVM.HaltedReason), false)
		appendStepMicroSteps(&out.Render, "run", true)
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
		st.initEVM()
		saveState(state, st)
		out.Render = buildEnvelope(st, "reset", "重置 EVM (PC=0)", true)
		return out, nil
	}

	return fw.ActionOutput{Success: false, ErrorMessage: "未知 ActionCode: " + in.ActionCode}, errors.New("unknown action")
}

// =====================================================================
// RenderEnvelope 构造
// =====================================================================

func buildEnvelope(st snapState, reason, summary string, fullSnapshot bool) fw.RenderEnvelope {
	prims := make([]fw.Primitive, 0, 30)

	// 1) 流水线：fetch → decode → execute → halt
	phases := []string{"fetch", "decode", "execute", "halt"}
	phaseIDs := []string{"phase-fetch", "phase-decode", "phase-execute", "phase-halt"}
	prims = append(prims, fw.PrimStack("pipeline", phaseIDs, "horizontal"))
	curPhase := 0
	if st.EVM.Halted {
		curPhase = 3
	} else if st.EVM.PC > 0 {
		curPhase = 2
	} else {
		curPhase = 1
	}
	for i, p := range phases {
		role := p
		status := "normal"
		if i == curPhase {
			status = "active"
		}
		prims = append(prims, fw.PrimNode(phaseIDs[i], p, status, role))
	}
	for i := 0; i < 3; i++ {
		anim := ""
		if i < curPhase {
			anim = "flow"
		}
		prims = append(prims, fw.PrimEdge(fmt.Sprintf("ph-edge-%d", i), phaseIDs[i], phaseIDs[i+1], "solid", anim))
	}

	// 2) 反汇编代码块（高亮当前 PC）
	disLines := []string{"PC    Op            Bytes"}
	highlight := []int{}
	for i, l := range st.Disasm {
		mark := "  "
		if l.PC == st.EVM.PC && !st.EVM.Halted {
			mark = "▶ "
			highlight = append(highlight, i+1)
		}
		disLines = append(disLines, fmt.Sprintf("%s%-5d %-13s %s", mark, l.PC, l.Name, l.Bytes))
	}
	prims = append(prims, fw.PrimCodeBlock("cb-disasm", strings.Join(disLines, "\n"), "asm", highlight, 24))

	// 3) Stack 寄存器组（最多 16 项）
	depth := len(st.EVM.Stack)
	showDepth := depth
	if showDepth > 16 {
		showDepth = 16
	}
	stackLabels := make([]string, showDepth)
	stackValues := make([]string, showDepth)
	for i := 0; i < showDepth; i++ {
		// top of stack 在最后
		idx := depth - 1 - i
		stackLabels[i] = fmt.Sprintf("[%d]", i)
		stackValues[i] = "0x" + toHex(st.EVM.Stack[idx])[60:] // 取末尾 4 字节方便阅读
	}
	if showDepth > 0 {
		prims = append(prims, fw.PrimRegisterRow("stack-row", stackLabels, stackValues, 0))
	}

	// 4) Memory hex
	memHex := "(empty)"
	if len(st.EVM.Memory) > 0 {
		memHex = hexBlocks(st.EVM.Memory)
	}
	prims = append(prims, fw.PrimCodeBlock("cb-memory",
		fmt.Sprintf("Memory (size=%d):\n%s", len(st.EVM.Memory), memHex),
		"text", nil, 12))

	// 5) Storage 表
	stoLines := []string{"Storage：key → value"}
	keys := []string{}
	for k := range st.EVM.Storage {
		keys = append(keys, k)
	}
	for i, k := range keys {
		v := st.EVM.Storage[k]
		stoLines = append(stoLines, fmt.Sprintf("  %s → %s", k[60:], toHex(v)[60:]))
		if i >= 8 {
			stoLines = append(stoLines, "  …")
			break
		}
	}
	prims = append(prims, fw.PrimCodeBlock("cb-storage", strings.Join(stoLines, "\n"), "text", nil, 12))

	// 6) 状态参数
	prims = append(prims, fw.PrimCodeBlock("cb-status",
		fmt.Sprintf("PC = %d / %d\nGas = %d / %d\nStack 深度 = %d\nMemory size = %d\nStorage 项数 = %d\nHalted = %v %s\ncaller = %s\ncall_value = %s\ncall_data size = %d",
			st.EVM.PC, len(st.EVM.Code), st.EVM.Gas, st.GasInit,
			depth, len(st.EVM.Memory), len(st.EVM.Storage),
			st.EVM.Halted, st.EVM.HaltedReason,
			st.EVM.Caller, st.CallValue, len(st.EVM.CallData)),
		"text", nil, 12))

	// 7) Gas 进度条
	prims = append(prims, fw.PrimProgressBar("bar-gas", float64(st.GasInit-st.EVM.Gas), float64(st.GasInit),
		fmt.Sprintf("Gas used %d/%d", st.GasInit-st.EVM.Gas, st.GasInit)))
	prims = append(prims, fw.PrimProgressBar("bar-pc", float64(st.EVM.PC), float64(len(st.EVM.Code)),
		fmt.Sprintf("PC %d/%d", st.EVM.PC, len(st.EVM.Code))))

	// 8) Return data
	if len(st.EVM.ReturnData) > 0 {
		prims = append(prims, fw.PrimCodeBlock("cb-return",
			fmt.Sprintf("Return Data (size=%d):\n%s", len(st.EVM.ReturnData), hex.EncodeToString(st.EVM.ReturnData)),
			"text", nil, 6))
	}

	// 9) 公式
	prims = append(prims, fw.PrimMathFormula("formula-evm",
		`\text{step}: op = code[PC];\ PC \mathrel{+}{=} 1;\ \text{exec}(op,\text{stack},\text{mem},\text{store});\ \text{gas} \mathrel{-}{=} \text{cost}(op)`, false))

	// 10) 动效
	prims = append(prims, fw.PrimGlow("glow-pc", "phase-execute", "info", 0.8))
	if st.EVM.Halted {
		col := "success"
		if st.EVM.HaltedReason == "REVERT" || strings.Contains(st.EVM.HaltedReason, "stack") || strings.Contains(st.EVM.HaltedReason, "out of gas") {
			col = "danger"
		}
		prims = append(prims, fw.PrimGlow("glow-halt", "phase-halt", col, 0.9))
		prims = append(prims, fw.PrimBurst("burst-halt", "phase-halt", col, int64(st.EVM.PC), 700))
	} else {
		prims = append(prims, fw.PrimPulse("pulse-pc", "cb-disasm", "info", 1500))
	}

	// 11) 联动徽章
	prims = append(prims, fw.PrimLinkIndicator("link-sec", linkGroupContractSec, "idle", ""))

	if st.LastError != "" {
		prims = append(prims, fw.PrimErrorOverlay("err", "warning", "EVM 错误", st.LastError, "scene", "请检查参数", true))
	}

	return fw.RenderEnvelope{
		Primitives:     prims,
		IsFullSnapshot: fullSnapshot,
		ChangedKeys:    []string{reason},
		Data:           buildSidePanelData(st, summary),
		ContainerData: []fw.ContainerMetric{
			{SourceContainer: "geth", MetricKey: "evm.pc", Value: st.EVM.PC, TargetPrimitive: "cb-disasm", TargetParam: "highlight_line"},
		},
	}
}

func hexBlocks(b []byte) string {
	out := ""
	for i := 0; i < len(b); i += 16 {
		end := i + 16
		if end > len(b) {
			end = len(b)
		}
		if i > 0 {
			out += "\n"
		}
		out += fmt.Sprintf("%04x: %s", i, hex.EncodeToString(b[i:end]))
		if i >= 64 && i+16 < len(b) {
			out += "\n…"
			break
		}
	}
	return out
}

func buildSidePanelData(st snapState, summary string) map[string]any {
	d := map[string]any{
		"pc":            st.EVM.PC,
		"code_size":     len(st.EVM.Code),
		"gas_remaining": st.EVM.Gas,
		"gas_init":      st.GasInit,
		"stack_depth":   len(st.EVM.Stack),
		"memory_size":   len(st.EVM.Memory),
		"storage_keys":  len(st.EVM.Storage),
		"halted":        st.EVM.Halted,
		"halt_reason":   st.EVM.HaltedReason,
		"caller":        st.EVM.Caller,
		"return_size":   len(st.EVM.ReturnData),
	}
	if summary != "" {
		d["summary"] = summary
	}
	return d
}

// =====================================================================
// MicroStep
// =====================================================================

func appendLoadMicroSteps(env *fw.RenderEnvelope) {
	env.MicroSteps = []fw.MicroStep{
		{ID: "ld-1", Label: "解析字节码 hex", DurationMs: 400, HighlightIDs: []string{"cb-disasm"}},
		{ID: "ld-2", Label: "反汇编生成 PC ↔ Op 映射", DurationMs: 500, HighlightIDs: []string{"cb-disasm"}},
		{ID: "ld-3", Label: "重置 EVM 状态", DurationMs: 400, HighlightIDs: []string{"cb-status"}, IsLinkTrigger: true},
	}
}

func appendStepMicroSteps(env *fw.RenderEnvelope, op string, halted bool) {
	tail := "等待下一步"
	if halted {
		tail = "执行结束（halted）"
	}
	env.MicroSteps = []fw.MicroStep{
		{ID: "s-1", Label: "fetch: 取 code[PC]", DurationMs: 300, HighlightIDs: []string{"phase-fetch", "cb-disasm"}},
		{ID: "s-2", Label: "decode: " + op, DurationMs: 300, HighlightIDs: []string{"phase-decode"}, FirePrimitives: []string{"glow-pc"}},
		{ID: "s-3", Label: "execute: 更新 stack/mem/storage", DurationMs: 400, HighlightIDs: []string{"phase-execute", "stack-row", "cb-memory", "cb-storage"}, FirePrimitives: []string{"pulse-pc"}},
		{ID: "s-4", Label: tail, DurationMs: 300, HighlightIDs: []string{"phase-halt", "cb-status"}, FirePrimitives: []string{"burst-halt"}, IsLinkTrigger: true},
	}
}

// =====================================================================
// 联动
// =====================================================================

func publishOwnerSubtree(env *fw.RenderEnvelope, st snapState) {
	if env.Data == nil {
		env.Data = map[string]any{}
	}
	env.LinkTriggers = append(env.LinkTriggers, fw.LinkTrigger{
		ID:             "evm-publish",
		SourceScene:    sceneCode,
		SourceAction:   "publish_evm",
		LinkGroup:      linkGroupContractSec,
		ChangedFields:  []string{"contract.evm.pc", "contract.evm.gas"},
		Payload:        map[string]any{"pc": st.EVM.PC, "gas": st.EVM.Gas},
		SourceAnchorID: "evm-anchor",
		TargetAnchorID: "contract-sec-anchor",
	})
	env.ChangedKeys = append(env.ChangedKeys, "contract.evm.pc", "contract.evm.gas")
	env.Data["link_owner_subtree"] = linkOwnerSubtree
}

func ownerDiff(st snapState) map[string]any {
	return map[string]any{
		"contract": map[string]any{
			"evm": map[string]any{
				"pc":           st.EVM.PC,
				"gas":          st.EVM.Gas,
				"stack_depth":  len(st.EVM.Stack),
				"memory_size":  len(st.EVM.Memory),
				"storage_keys": len(st.EVM.Storage),
				"halted":       st.EVM.Halted,
				"halt_reason":  st.EVM.HaltedReason,
			},
		},
	}
}

// =====================================================================
// 工具
// =====================================================================

