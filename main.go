// package main contains a simple "interpreter"
//
// This interpreter is the skeleton of a virtual machine which is built upon
// the design of using closures to populate function-pointers which are then
// executed in turn.
package main

import "fmt"

// VM contains the virtual machine, it mostly exists to host a stack and a
// programs which has been "compiled" into a series of function-pointers.
//
// The VM will execute a program by calling each function-pointer in turn,
// and those functions will be closures that mutate the state of the VM as
// side-effects.
type VM struct {
	// stack contains arguments which are passed to our minimal operations.
	stack []float64

	// program contains the program we're going to execute.
	program []vmFunc
}

// New is the VM constructor
func New(prog []vmFunc) *VM {
	return &VM{program: prog}
}

// RunProgram executes the program which was given in the constructor.
func (vm *VM) RunProgram() {

	// Reset the state of the stack each run.
	vm.stack = []float64{}

	// Execute the program.
	code := vm.program
	ip := 0

	// For each operation.  Run it
	for ip < len(code) {

		// Here the return value controls which operation
		// we execute next by changing the offset within
		// the code-array.
		//
		// We could allow loops by having the opcodes return
		// different values.  In this example we only move
		// forwards.
		ip += code[ip](vm)
	}
}

//
// Okay here we write some helpers which create/return closures
//

// vmFunc is the type-signature of a mutating primitive we would implement.
type vmFunc func(vm *VM) int

// newInt creates, and returns, a closure which adds the float value to the program stack.
func newInt(n float64) vmFunc {
	return func(v *VM) int {
		v.stack = append(v.stack, n)
		return 1
	}
}

// mathOp creates, and returns, a closure which adds a mathematical operation to the program.
//
// We're stack-based so we pop our arguments, run the operation, and push the result.
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

// printOp shows the value at the top of the stack.
func printOp() vmFunc {
	return func(v *VM) int {
		if len(v.stack) > 0 {
			fmt.Printf("%f\n", v.stack[len(v.stack)-1])
		} else {
			fmt.Printf("stack is empty\n")
		}
		return 1
	}
}

//
// Closure time is over now.
//

// main is our entry-point and creates/runs a program.
func main() {

	// Create a basic program.
	prog := []vmFunc{}
	prog = append(prog, newInt(3))
	prog = append(prog, newInt(7))
	prog = append(prog, mathOp("+")) // 3 + 7
	prog = append(prog, printOp())

	prog = append(prog, newInt(4.5)) // * 4.5
	prog = append(prog, mathOp("*"))
	prog = append(prog, printOp())

	// create an interpreter to run that program.
	v := New(prog)

	// now launch it
	v.RunProgram()
}
