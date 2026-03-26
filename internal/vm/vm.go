package vm

import (
	"errors"
	"fmt"

	"github.com/Neo4717/NeoCoin/internal/blockchain"
)

type Opcode byte

const (
	OpInvalid Opcode = iota
	OpPUSH
	OpPOP
	OpADD
	OpSUB
	OpMUL
	OpDIV
	OpSTORE
	OpLOAD
	OpJUMP
	OpJUMPI
	OpEQ
	OpLT
	OpGT
	OpAND
	OpOR
	OpNOT
	OpRETURN
	OpCALL
	OpDUP
	OpSWAP
)

var GasCosts = map[Opcode]uint64{
	OpPUSH:   1,
	OpPOP:    1,
	OpADD:    2,
	OpSUB:    2,
	OpMUL:    3,
	OpDIV:    3,
	OpSTORE:  5,
	OpLOAD:   3,
	OpJUMP:   2,
	OpJUMPI:  3,
	OpEQ:     2,
	OpLT:     2,
	OpGT:     2,
	OpAND:    2,
	OpOR:     2,
	OpNOT:    1,
	OpRETURN: 1,
	OpCALL:   5,
	OpDUP:    1,
	OpSWAP:   1,
}

type Stack []int64

func (s *Stack) Push(v int64) {
	*s = append(*s, v)
}

func (s *Stack) Pop() (int64, error) {
	if len(*s) == 0 {
		return 0, errors.New("stack underflow")
	}
	v := (*s)[len(*s)-1]
	*s = (*s)[:len(*s)-1]
	return v, nil
}

func (s *Stack) Peek() (int64, error) {
	if len(*s) == 0 {
		return 0, errors.New("stack underflow")
	}
	return (*s)[len(*s)-1], nil
}

func (s *Stack) Swap(n int) error {
	if len(*s) < n+1 {
		return errors.New("stack underflow")
	}
	l := len(*s)
	(*s)[l-1], (*s)[l-1-n] = (*s)[l-1-n], (*s)[l-1]
	return nil
}

func (s *Stack) Dup() error {
	if len(*s) == 0 {
		return errors.New("stack underflow")
	}
	v := (*s)[len(*s)-1]
	*s = append(*s, v)
	return nil
}

type VM struct {
	stack           Stack
	memory          []byte
	pc              int
	gas             uint64
	gasLimit        uint64
	code            []byte
	contractStore   map[string]*Contract
	storage         map[string]map[string][]byte
	callStack       []int
	returnStack     Stack
	currentContract string
}

type Contract struct {
	Code     []byte
	Storage  map[string][]byte
	Owner    string
	Name     string
	GasLimit uint64
}

type ExecutionResult struct {
	Success   bool
	GasUsed   uint64
	ReturnVal int64
	Logs      []string
}

func NewVM(gasLimit uint64) *VM {
	return &VM{
		stack:         make(Stack, 0),
		memory:        make([]byte, 1024),
		pc:            0,
		gas:           gasLimit,
		gasLimit:      gasLimit,
		code:          nil,
		contractStore: make(map[string]*Contract),
		storage:       make(map[string]map[string][]byte),
		callStack:     make([]int, 0),
		returnStack:   make(Stack, 0),
	}
}

func (v *VM) Execute(code []byte) (*ExecutionResult, error) {
	v.code = code
	v.pc = 0
	v.stack = make(Stack, 0)
	if v.storage == nil {
		v.storage = make(map[string]map[string][]byte)
	}
	if v.currentContract == "" {
		v.currentContract = "default"
	}
	result := &ExecutionResult{
		Logs: make([]string, 0),
	}

	for {
		if v.pc >= len(v.code) {
			break
		}
		if v.gas < GasCosts[OpInvalid] {
			return result, errors.New("out of gas")
		}

		op := Opcode(v.code[v.pc])

		if op == OpInvalid || (op > OpCALL && op != OpDUP && op != OpSWAP) {
			return result, fmt.Errorf("invalid opcode: %d at position %d", op, v.pc)
		}

		cost := GasCosts[op]
		if v.gas < cost {
			return result, errors.New("out of gas")
		}
		v.gas -= cost
		result.GasUsed = v.gasLimit - v.gas

		err := v.executeOpcode(op)
		if err != nil {
			return result, err
		}
	}

	result.Success = true
	if len(v.stack) > 0 {
		result.ReturnVal = v.stack[len(v.stack)-1]
	}

	return result, nil
}

func (v *VM) executeOpcode(op Opcode) error {
	switch op {
	case OpPUSH:
		v.pc++
		if v.pc >= len(v.code) {
			return errors.New("unexpected end of code during PUSH")
		}
		v.stack.Push(int64(v.code[v.pc]))
		v.pc++
		return nil

	case OpPOP:
		_, err := v.stack.Pop()
		if err != nil {
			return err
		}
		v.pc++
		return nil

	case OpADD:
		a, err := v.stack.Pop()
		if err != nil {
			return err
		}
		b, err := v.stack.Pop()
		if err != nil {
			return err
		}
		v.stack.Push(b + a)
		v.pc++
		return nil

	case OpSUB:
		a, err := v.stack.Pop()
		if err != nil {
			return err
		}
		b, err := v.stack.Pop()
		if err != nil {
			return err
		}
		v.stack.Push(b - a)
		v.pc++
		return nil

	case OpMUL:
		a, err := v.stack.Pop()
		if err != nil {
			return err
		}
		b, err := v.stack.Pop()
		if err != nil {
			return err
		}
		v.stack.Push(b * a)
		v.pc++
		return nil

	case OpDIV:
		a, err := v.stack.Pop()
		if err != nil {
			return err
		}
		if a == 0 {
			return errors.New("division by zero")
		}
		b, err := v.stack.Pop()
		if err != nil {
			return err
		}
		v.stack.Push(b / a)
		v.pc++
		return nil

	case OpSTORE:
		val, err := v.stack.Pop()
		if err != nil {
			return err
		}
		key, err := v.stack.Pop()
		if err != nil {
			return err
		}
		keyStr := fmt.Sprintf("%d", key)
		if v.storage[v.currentContract] == nil {
			v.storage[v.currentContract] = make(map[string][]byte)
		}
		v.storage[v.currentContract][keyStr] = []byte(fmt.Sprintf("%d", val))
		v.pc++
		return nil

	case OpLOAD:
		key, err := v.stack.Pop()
		if err != nil {
			return err
		}
		keyStr := fmt.Sprintf("%d", key)
		if v.storage[v.currentContract] == nil {
			v.stack.Push(0)
		} else {
			val := v.storage[v.currentContract][keyStr]
			if len(val) == 0 {
				v.stack.Push(0)
			} else {
				var valInt int64
				fmt.Sscanf(string(val), "%d", &valInt)
				v.stack.Push(valInt)
			}
		}
		v.pc++
		return nil

	case OpJUMP:
		target, err := v.stack.Pop()
		if err != nil {
			return err
		}
		if target < 0 || target >= int64(len(v.code)) {
			return fmt.Errorf("jump target out of bounds: %d", target)
		}
		v.pc = int(target)
		return nil

	case OpJUMPI:
		cond, err := v.stack.Pop()
		if err != nil {
			return err
		}
		target, err := v.stack.Pop()
		if err != nil {
			return err
		}
		if cond != 0 {
			if target < 0 || target >= int64(len(v.code)) {
				return fmt.Errorf("jump target out of bounds: %d", target)
			}
			v.pc = int(target)
		} else {
			v.pc++
		}
		return nil

	case OpEQ:
		a, err := v.stack.Pop()
		if err != nil {
			return err
		}
		b, err := v.stack.Pop()
		if err != nil {
			return err
		}
		if b == a {
			v.stack.Push(1)
		} else {
			v.stack.Push(0)
		}
		v.pc++
		return nil

	case OpLT:
		a, err := v.stack.Pop()
		if err != nil {
			return err
		}
		b, err := v.stack.Pop()
		if err != nil {
			return err
		}
		if b < a {
			v.stack.Push(1)
		} else {
			v.stack.Push(0)
		}
		v.pc++
		return nil

	case OpGT:
		a, err := v.stack.Pop()
		if err != nil {
			return err
		}
		b, err := v.stack.Pop()
		if err != nil {
			return err
		}
		if b > a {
			v.stack.Push(1)
		} else {
			v.stack.Push(0)
		}
		v.pc++
		return nil

	case OpAND:
		a, err := v.stack.Pop()
		if err != nil {
			return err
		}
		b, err := v.stack.Pop()
		if err != nil {
			return err
		}
		if b != 0 && a != 0 {
			v.stack.Push(1)
		} else {
			v.stack.Push(0)
		}
		v.pc++
		return nil

	case OpOR:
		a, err := v.stack.Pop()
		if err != nil {
			return err
		}
		b, err := v.stack.Pop()
		if err != nil {
			return err
		}
		if b != 0 || a != 0 {
			v.stack.Push(1)
		} else {
			v.stack.Push(0)
		}
		v.pc++
		return nil

	case OpNOT:
		a, err := v.stack.Pop()
		if err != nil {
			return err
		}
		if a == 0 {
			v.stack.Push(1)
		} else {
			v.stack.Push(0)
		}
		v.pc++
		return nil

	case OpCALL:
		addr, err := v.stack.Pop()
		if err != nil {
			return err
		}
		v.callStack = append(v.callStack, v.pc+1)
		v.pc = int(addr)
		return nil

	case OpRETURN:
		if len(v.callStack) > 0 {
			v.pc = v.callStack[len(v.callStack)-1]
			v.callStack = v.callStack[:len(v.callStack)-1]
		}
		return nil

	case OpDUP:
		if err := v.stack.Dup(); err != nil {
			return err
		}
		v.pc++
		return nil

	case OpSWAP:
		n, err := v.stack.Pop()
		if err != nil {
			return err
		}
		if err := v.stack.Swap(int(n)); err != nil {
			return err
		}
		v.pc++
		return nil

	default:
		return fmt.Errorf("unimplemented opcode: %v", op)
	}
}

func (v *VM) Deploy(code []byte, owner string, name string, gasLimit uint64) (string, error) {
	contract := &Contract{
		Code:     code,
		Storage:  make(map[string][]byte),
		Owner:    owner,
		Name:     name,
		GasLimit: gasLimit,
	}

	addr := blockchain.GenerateAddress([]byte(name + owner))
	v.contractStore[addr] = contract

	return addr, nil
}

func (v *VM) Call(contractAddr string, gasLimit uint64) (*ExecutionResult, error) {
	contract, exists := v.contractStore[contractAddr]
	if !exists {
		return nil, errors.New("contract not found")
	}

	vm := NewVM(gasLimit)
	vm.contractStore = v.contractStore
	if v.storage[contractAddr] == nil {
		v.storage[contractAddr] = make(map[string][]byte)
	}
	for k, val := range contract.Storage {
		v.storage[contractAddr][k] = val
	}
	vm.storage = v.storage
	vm.gasLimit = gasLimit
	vm.gas = gasLimit

	result, err := vm.Execute(contract.Code)
	if err != nil {
		return result, err
	}

	for k, val := range v.storage[contractAddr] {
		contract.Storage[k] = val
	}

	return result, nil
}

func (v *VM) GetStorage(contractAddr string) (map[string][]byte, error) {
	contract, exists := v.contractStore[contractAddr]
	if !exists {
		return nil, errors.New("contract not found")
	}
	return contract.Storage, nil
}

func (v *VM) GetContract(addr string) (*Contract, error) {
	contract, exists := v.contractStore[addr]
	if !exists {
		return nil, errors.New("contract not found")
	}
	return contract, nil
}

func (v *VM) RemainingGas() uint64 {
	return v.gas
}

func (v *VM) GetStack() []int64 {
	return v.stack
}
