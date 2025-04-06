// package main contains a simple "interpreter"
//
// This interpreter is the skeleton of a virtual machine which is built upon
// the design of using closures to populate function-pointers which are then
// executed in turn.
package main

import "fmt"

// Type VM contains the virtual machine, it mostly exists to host a stack and a
// set of programs.
type VM struct {
	// stack contains arguments which are passed to our minimal operations.
	// We support +, -, * & / operations only.
	stack []float64

	// program contains the program we're going to execute.
	// This is hardcoded.
	program []vmFunc
}

// vmFunc is the type-signature of a primitive we've implemented.
type vmFunc func(vm *VM) int

// New is the VM constructor
func New() *VM {
	return &VM{}
}

// newInt creates, and returns, a closure which adds the float value to the program stack.
func newInt(n float64) vmFunc {
	return func(v *VM) int {
		v.stack = append(v.stack, n)
		return 1
	}
}

// mathOp creates, and returns, a closure which adds a maths operation to the
// program.
func mathOp(op string) vmFunc {

	return func(v *VM) int {
		x := 0.0
		y := 0.0
		res := 0.0

		x, v.stack = v.stack[len(v.stack)-1], v.stack[:len(v.stack)-1]
		y, v.stack = v.stack[len(v.stack)-1], v.stack[:len(v.stack)-1]

		switch op {
		case "+":
			res = x + y
		case "-":
			res = x - y
		case "/":
			res = x / y
		case "*":
			res = x * y
		default:
			panic("unknown operation")
		}
		v.stack = append(v.stack, res)
		return 1
	}
}

// RunProgram creates a simple program and executes it.
func (vm *VM) RunProgram() {

	// The stack is empty remember.

	// Create a basic program.

	// 3
	vm.program = append(vm.program, newInt(3))

	// 42
	vm.program = append(vm.program, newInt(42))

	// +
	vm.program = append(vm.program, mathOp("+"))

	// show
	vm.program = append(vm.program,
		func(v *VM) int {
			x := 0.0

			x, v.stack = v.stack[len(v.stack)-1], v.stack[:len(v.stack)-1]

			fmt.Printf("RESULT:%f\n", x)
			return 1
		},
	)

	// Execute the program.
	code := vm.program
	ip := 0

	// For each operation.  Run it
	for ip < len(code) {

		// Here the re
		ip += code[ip](vm)
	}
}

// main is our entrypoint and creates/runs a program.
func main() {

	v := New()
	v.RunProgram()
}
