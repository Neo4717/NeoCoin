package vm

import (
	"testing"
)

func TestStackPushPop(t *testing.T) {
	stack := Stack{}

	stack.Push(42)
	stack.Push(100)

	val, err := stack.Pop()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != 100 {
		t.Errorf("expected 100, got %d", val)
	}

	val, err = stack.Pop()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != 42 {
		t.Errorf("expected 42, got %d", val)
	}
}

func TestStackPeek(t *testing.T) {
	stack := Stack{}
	stack.Push(42)
	stack.Push(100)

	val, err := stack.Peek()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != 100 {
		t.Errorf("expected 100, got %d", val)
	}

	val, err = stack.Pop()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != 100 {
		t.Errorf("expected 100, got %d", val)
	}
}

func TestStackUnderflow(t *testing.T) {
	stack := Stack{}

	_, err := stack.Pop()
	if err == nil {
		t.Error("expected error for stack underflow")
	}
}

func TestStackDup(t *testing.T) {
	stack := Stack{}
	stack.Push(42)
	stack.Push(100)

	err := stack.Dup()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	val, err := stack.Pop()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != 100 {
		t.Errorf("expected 100, got %d", val)
	}

	val, err = stack.Pop()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != 100 {
		t.Errorf("expected 100, got %d", val)
	}
}

func TestStackSwap(t *testing.T) {
	stack := Stack{}
	stack.Push(10)
	stack.Push(20)
	stack.Push(30)

	err := stack.Swap(1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	val, _ := stack.Pop()
	if val != 20 {
		t.Errorf("expected 20, got %d", val)
	}

	val, _ = stack.Pop()
	if val != 30 {
		t.Errorf("expected 30, got %d", val)
	}

	val, _ = stack.Pop()
	if val != 10 {
		t.Errorf("expected 10, got %d", val)
	}
}

func TestVM_PUSH(t *testing.T) {
	vm := NewVM(100)

	code := []byte{byte(OpPUSH), 42}
	result, err := vm.Execute(code)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Error("execution should succeed")
	}
	if result.ReturnVal != 42 {
		t.Errorf("expected return value 42, got %d", result.ReturnVal)
	}
}

func TestVM_ADD(t *testing.T) {
	vm := NewVM(100)

	code := []byte{
		byte(OpPUSH), 5,
		byte(OpPUSH), 3,
		byte(OpADD),
	}
	result, err := vm.Execute(code)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Error("execution should succeed")
	}
	if result.ReturnVal != 8 {
		t.Errorf("expected return value 8, got %d", result.ReturnVal)
	}
}

func TestVM_SUB(t *testing.T) {
	vm := NewVM(100)

	code := []byte{
		byte(OpPUSH), 10,
		byte(OpPUSH), 3,
		byte(OpSUB),
	}
	result, err := vm.Execute(code)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Error("execution should succeed")
	}
	if result.ReturnVal != 7 {
		t.Errorf("expected return value 7, got %d", result.ReturnVal)
	}
}

func TestVM_MUL(t *testing.T) {
	vm := NewVM(100)

	code := []byte{
		byte(OpPUSH), 4,
		byte(OpPUSH), 5,
		byte(OpMUL),
	}
	result, err := vm.Execute(code)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Error("execution should succeed")
	}
	if result.ReturnVal != 20 {
		t.Errorf("expected return value 20, got %d", result.ReturnVal)
	}
}

func TestVM_DIV(t *testing.T) {
	vm := NewVM(100)

	code := []byte{
		byte(OpPUSH), 20,
		byte(OpPUSH), 4,
		byte(OpDIV),
	}
	result, err := vm.Execute(code)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Error("execution should succeed")
	}
	if result.ReturnVal != 5 {
		t.Errorf("expected return value 5, got %d", result.ReturnVal)
	}
}

func TestVM_DIVByZero(t *testing.T) {
	vm := NewVM(100)

	code := []byte{
		byte(OpPUSH), 10,
		byte(OpPUSH), 0,
		byte(OpDIV),
	}
	_, err := vm.Execute(code)
	if err == nil {
		t.Error("expected error for division by zero")
	}
}

func TestVM_POP(t *testing.T) {
	vm := NewVM(100)

	code := []byte{
		byte(OpPUSH), 42,
		byte(OpPUSH), 100,
		byte(OpPOP),
	}
	result, err := vm.Execute(code)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Error("execution should succeed")
	}
	if result.ReturnVal != 42 {
		t.Errorf("expected return value 42, got %d", result.ReturnVal)
	}
}

func TestVM_STOREAndLOAD(t *testing.T) {
	vm := NewVM(100)

	code := []byte{
		byte(OpPUSH), 1,
		byte(OpPUSH), 42,
		byte(OpSTORE),
		byte(OpPUSH), 1,
		byte(OpLOAD),
	}
	result, err := vm.Execute(code)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Error("execution should succeed")
	}
	if result.ReturnVal != 42 {
		t.Errorf("expected return value 42, got %d", result.ReturnVal)
	}
}

func TestVM_JUMP(t *testing.T) {
	vm := NewVM(100)

	code := []byte{
		byte(OpPUSH), 5,
		byte(OpPUSH), 5,
		byte(OpJUMP),
		byte(OpPUSH), 9,
		byte(OpADD),
	}
	result, err := vm.Execute(code)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Error("execution should succeed")
	}
	if result.ReturnVal != 14 {
		t.Errorf("expected return value 14, got %d", result.ReturnVal)
	}
}

func TestVM_JUMPI(t *testing.T) {
	vm := NewVM(100)

	code := []byte{
		byte(OpPUSH), 9,
		byte(OpPUSH), 5,
		byte(OpPUSH), 9,
		byte(OpPUSH), 1,
		byte(OpJUMPI),
		byte(OpPUSH), 1,
		byte(OpPUSH), 2,
		byte(OpADD),
	}
	result, err := vm.Execute(code)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Error("execution should succeed")
	}
	if result.ReturnVal != 3 {
		t.Errorf("expected return value 3, got %d", result.ReturnVal)
	}
}

func TestVM_JUMPINotTaken(t *testing.T) {
	vm := NewVM(100)

	code := []byte{
		byte(OpPUSH), 5,
		byte(OpPUSH), 9,
		byte(OpPUSH), 0,
		byte(OpJUMPI),
		byte(OpPUSH), 9,
		byte(OpADD),
	}
	result, err := vm.Execute(code)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Error("execution should succeed")
	}
	if result.ReturnVal != 14 {
		t.Errorf("expected return value 14, got %d", result.ReturnVal)
	}
}

func TestVM_EQ(t *testing.T) {
	vm := NewVM(100)

	code := []byte{
		byte(OpPUSH), 5,
		byte(OpPUSH), 5,
		byte(OpEQ),
	}
	result, err := vm.Execute(code)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Error("execution should succeed")
	}
	if result.ReturnVal != 1 {
		t.Errorf("expected return value 1, got %d", result.ReturnVal)
	}

	vm2 := NewVM(100)
	code2 := []byte{
		byte(OpPUSH), 5,
		byte(OpPUSH), 3,
		byte(OpEQ),
	}
	result2, err := vm2.Execute(code2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result2.ReturnVal != 0 {
		t.Errorf("expected return value 0, got %d", result2.ReturnVal)
	}
}

func TestVM_LT(t *testing.T) {
	vm := NewVM(100)

	code := []byte{
		byte(OpPUSH), 3,
		byte(OpPUSH), 5,
		byte(OpLT),
	}
	result, err := vm.Execute(code)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Error("execution should succeed")
	}
	if result.ReturnVal != 1 {
		t.Errorf("expected return value 1, got %d", result.ReturnVal)
	}
}

func TestVM_GT(t *testing.T) {
	vm := NewVM(100)

	code := []byte{
		byte(OpPUSH), 5,
		byte(OpPUSH), 3,
		byte(OpGT),
	}
	result, err := vm.Execute(code)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Error("execution should succeed")
	}
	if result.ReturnVal != 1 {
		t.Errorf("expected return value 1, got %d", result.ReturnVal)
	}
}

func TestVM_AND(t *testing.T) {
	vm := NewVM(100)

	code := []byte{
		byte(OpPUSH), 1,
		byte(OpPUSH), 1,
		byte(OpAND),
	}
	result, err := vm.Execute(code)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ReturnVal != 1 {
		t.Errorf("expected return value 1, got %d", result.ReturnVal)
	}

	vm2 := NewVM(100)
	code2 := []byte{
		byte(OpPUSH), 1,
		byte(OpPUSH), 0,
		byte(OpAND),
	}
	result2, err := vm2.Execute(code2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result2.ReturnVal != 0 {
		t.Errorf("expected return value 0, got %d", result2.ReturnVal)
	}
}

func TestVM_OR(t *testing.T) {
	vm := NewVM(100)

	code := []byte{
		byte(OpPUSH), 0,
		byte(OpPUSH), 1,
		byte(OpOR),
	}
	result, err := vm.Execute(code)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ReturnVal != 1 {
		t.Errorf("expected return value 1, got %d", result.ReturnVal)
	}

	vm2 := NewVM(100)
	code2 := []byte{
		byte(OpPUSH), 0,
		byte(OpPUSH), 0,
		byte(OpOR),
	}
	result2, err := vm2.Execute(code2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result2.ReturnVal != 0 {
		t.Errorf("expected return value 0, got %d", result2.ReturnVal)
	}
}

func TestVM_NOT(t *testing.T) {
	vm := NewVM(100)

	code := []byte{
		byte(OpPUSH), 0,
		byte(OpNOT),
	}
	result, err := vm.Execute(code)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ReturnVal != 1 {
		t.Errorf("expected return value 1, got %d", result.ReturnVal)
	}

	vm2 := NewVM(100)
	code2 := []byte{
		byte(OpPUSH), 1,
		byte(OpNOT),
	}
	result2, err := vm2.Execute(code2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result2.ReturnVal != 0 {
		t.Errorf("expected return value 0, got %d", result2.ReturnVal)
	}
}

func TestVM_DUP(t *testing.T) {
	vm := NewVM(100)

	code := []byte{
		byte(OpPUSH), 5,
		byte(OpDUP),
		byte(OpADD),
	}
	result, err := vm.Execute(code)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ReturnVal != 10 {
		t.Errorf("expected return value 10, got %d", result.ReturnVal)
	}
}

func TestVM_SWAP(t *testing.T) {
	vm := NewVM(100)

	code := []byte{
		byte(OpPUSH), 5,
		byte(OpPUSH), 10,
		byte(OpPUSH), 1,
		byte(OpSWAP),
		byte(OpADD),
	}
	result, err := vm.Execute(code)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ReturnVal != 15 {
		t.Errorf("expected return value 15, got %d", result.ReturnVal)
	}
}

func TestVM_GasConsumption(t *testing.T) {
	vm := NewVM(100)

	code := []byte{
		byte(OpPUSH), 5,
		byte(OpPUSH), 3,
		byte(OpADD),
		byte(OpPUSH), 2,
		byte(OpMUL),
	}
	result, err := vm.Execute(code)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedGas := GasCosts[OpPUSH]*2 + GasCosts[OpADD] + GasCosts[OpPUSH] + GasCosts[OpMUL]
	if result.GasUsed != expectedGas {
		t.Errorf("expected gas used %d, got %d", expectedGas, result.GasUsed)
	}

	if vm.RemainingGas() != vm.gasLimit-result.GasUsed {
		t.Error("remaining gas mismatch")
	}
}

func TestVM_OutOfGas(t *testing.T) {
	vm := NewVM(1)

	code := []byte{
		byte(OpPUSH), 5,
		byte(OpPUSH), 3,
		byte(OpADD),
	}
	_, err := vm.Execute(code)
	if err == nil {
		t.Error("expected out of gas error")
	}
}

func TestVM_DeployAndCall(t *testing.T) {
	vm := NewVM(1000)

	code := []byte{
		byte(OpPUSH), 5,
		byte(OpPUSH), 3,
		byte(OpADD),
	}

	addr, err := vm.Deploy(code, "owner1", "test_contract", 1000)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if addr == "" {
		t.Error("expected non-empty address")
	}

	contract, err := vm.GetContract(addr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if contract == nil {
		t.Error("expected contract")
	}

	result, err := vm.Call(addr, 500)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.GasUsed == 0 {
		t.Error("expected gas used")
	}

	storage, err := vm.GetStorage(addr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if storage == nil {
		t.Error("expected storage")
	}
}

func TestVM_ContractNotFound(t *testing.T) {
	vm := NewVM(100)

	_, err := vm.Call("invalid_address", 100)
	if err == nil {
		t.Error("expected error for contract not found")
	}
}

func TestVM_InvalidOpcode(t *testing.T) {
	vm := NewVM(100)

	code := []byte{byte(OpInvalid)}
	_, err := vm.Execute(code)
	if err == nil {
		t.Error("expected error for invalid opcode")
	}
}

func TestGasCosts(t *testing.T) {
	tests := []struct {
		op       Opcode
		expected uint64
	}{
		{OpPUSH, 1},
		{OpPOP, 1},
		{OpADD, 2},
		{OpSUB, 2},
		{OpMUL, 3},
		{OpDIV, 3},
		{OpSTORE, 5},
		{OpLOAD, 3},
		{OpJUMP, 2},
		{OpJUMPI, 3},
		{OpEQ, 2},
		{OpLT, 2},
		{OpGT, 2},
		{OpAND, 2},
		{OpOR, 2},
		{OpNOT, 1},
		{OpRETURN, 1},
		{OpCALL, 5},
		{OpDUP, 1},
		{OpSWAP, 1},
	}

	for _, test := range tests {
		cost, exists := GasCosts[test.op]
		if !exists {
			t.Errorf("gas cost not defined for opcode %v", test.op)
			continue
		}
		if cost != test.expected {
			t.Errorf("expected gas cost %d for %v, got %d", test.expected, test.op, cost)
		}
	}
}
