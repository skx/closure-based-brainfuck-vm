// package main contains a simple brainfuck interpreter which compiles
// a program into a series of closures which can then be iterated
// over in turn.
//
// The intent behind this implementation is to show how a "real"
// interpreter might work, built without the traditional approach
// of compiling to, and walking, a series of bytecode values.
//
// Bytecode interpreters are faster than AST-walking interpreters
// however they need to have a lot of internal duplication:
//
// * Some part of the code emits bytecode for each operation.
//
// * Some part of the code "does stuff" for each bytecode opcode.
//
// Keeping those two things in sync is annoying overhead, and also
// the traditional "switch-based" interpretation of bytecode opcodes
// is slow.
//
// The idea here is that we walk over a series of function-pointers
// and that is going to be fast.  We can do that because we generated
// a series of closures containing each "opcode thing".
package main

import (
	"errors"
	"fmt"
	"os"
)

var (
	// ErrExit is a faux error we append to the end of all our
	// input programs, and can thus be used to detect the end
	// of a program.
	ErrExit = errors.New("EXIT")
)

// VM contains the virtual machine, and it mostly exists to hold
// the state of our brainfuck program.
//
// The state is both our compiled closures, and some notes for
// interpretation (i.e. helpers for the loop-counters & etc).
type VM struct {

	// brainfuck stuff

	// ip is the instruction pointer into the brainfuck
	// program which is to be executed.
	ip int

	// ptr is the brainfuck programs index offset.
	ptr int

	// memory is the memory-space the brainfuck program uses
	memory [30000]int

	// loops is used to lookup look bounds
	loops map[int]int

	// stdout holds output we should write to the console.
	// save it away and show it all at once to speedup!
	stdout string

	// driver

	// program contains the set of closures that we can
	// execute one by one, to run the actual user program
	program []vmFunc

	// err records any error received when running the program.
	err error
}

// New is the VM constructor which takes our program as input
// and compiles it into a series of closures.  Some basic problems
// are detected and returned here.
func New(bf string) (*VM, error) {

	// Create empty VM
	v := VM{}

	v.loops = make(map[int]int)
	loopStack := []int{}

	// Ensure we got a program
	if len(bf) < 1 {
		return nil, errors.New("empty program is invalid")
	}

	// Index and bounds
	i := 0
	max := len(bf)

	// Walk each character
	for i < max {

		// Handle each known character
		c := bf[i]

		switch c {
		case '+':

			// Record our starting position
			begin := i

			// Loop forward to see how many times the character
			// is repeated.
			for i < max {

				// If it isn't the same character
				// we're done
				if bf[i] != c {
					break
				}

				// Otherwise keep advancing forward
				i++
			}

			// Return the token and the times it was
			// seen in adjacent positions
			count := i - begin

			i -= 1
			v.program = append(v.program, makeIncCell(count))
		case '-':
			// Record our starting position
			begin := i

			// Loop forward to see how many times the character
			// is repeated.
			for i < max {

				// If it isn't the same character
				// we're done
				if bf[i] != c {
					break
				}

				// Otherwise keep advancing forward
				i++
			}

			// Return the token and the times it was
			// seen in adjacent positions
			count := i - begin

			i -= 1
			v.program = append(v.program, makeDecCell(count))
		case '<':
			// Record our starting position
			begin := i

			// Loop forward to see how many times the character
			// is repeated.
			for i < max {

				// If it isn't the same character
				// we're done
				if bf[i] != c {
					break
				}

				// Otherwise keep advancing forward
				i++
			}

			// Return the token and the times it was
			// seen in adjacent positions
			count := i - begin

			i -= 1
			v.program = append(v.program, makeDecPtr(count))
		case '>':
			// Record our starting position
			begin := i

			// Loop forward to see how many times the character
			// is repeated.
			for i < max {

				// If it isn't the same character
				// we're done
				if bf[i] != c {
					break
				}

				// Otherwise keep advancing forward
				i++
			}

			// Return the token and the times it was
			// seen in adjacent positions
			count := i - begin

			i -= 1
			v.program = append(v.program, makeIncPtr(count))
		case ',':
			v.program = append(v.program, makeRead())
		case '.':
			v.program = append(v.program, makeWrite())
		case '[':
			// loop open
			loopStack = append(loopStack, len(v.program))

			v.program = append(v.program, makeLoopOpen())

		case ']':
			// Pop position of last JumpIfZero ("[") instruction off stack
			openInstruction := loopStack[len(loopStack)-1]
			loopStack = loopStack[:len(loopStack)-1]

			// loop points to the end
			v.loops[openInstruction] = len(v.program)

			// end points to start
			v.loops[len(v.program)] = openInstruction

			// Now add the instruction
			v.program = append(v.program, makeLoopClose())
		default:
			// Invalid character.
			// ignored.
		}
		i++
	}

	// Add a fake "exit" trap to the end of our program
	v.program = append(v.program, makeExit())

	return &v, nil
}

// RunProgram executes the program which was given in the constructor.
func (vm *VM) RunProgram() error {

	// Reset the state of the program each run.
	vm.ptr = 0
	vm.ip = 0

	// Execute the program.
	code := vm.program

	// For each operation.  Run it
	for vm.ip < len(code) {

		// Call the closure.
		//
		// Here we assume that each opcode ends with
		// "vm.ip++", which lets us run forward.
		code[vm.ip](vm)

		// Did we get an error?
		if vm.err != nil {

			// Show any pending output
			fmt.Printf("%s\n", vm.stdout)
			vm.stdout = ""

			// If it is the fake exit-program error
			// then we ignore it.
			if vm.err == ErrExit {
				return nil
			}

			// otherwise return the error to the caller
			return vm.err
		}
	}

	return nil
}

//
// Okay here we write some helpers which create/return closures
//

// vmFunc is the type-signature of a mutating primitive we would implement.
type vmFunc func(vm *VM)

// makeExit: brainfuck implementation
func makeExit() vmFunc {
	return func(v *VM) {
		v.err = ErrExit
	}
}

// makeIncCell: brainfuck implementation
func makeIncCell(n int) vmFunc {
	return func(v *VM) {
		v.memory[v.ptr] += n
		v.ip += 1
	}
}

// makeDecCell: brainfuck implementation
func makeDecCell(n int) vmFunc {
	return func(v *VM) {
		v.memory[v.ptr] -= n
		v.ip += 1
	}
}

// makeIncPtr: brainfuck implementation
func makeIncPtr(n int) vmFunc {
	return func(v *VM) {
		v.ptr += n
		v.ip += 1
	}
}

// makeDecPtr: brainfuck implementation
func makeDecPtr(n int) vmFunc {
	return func(v *VM) {
		v.ptr -= n
		v.ip += 1
	}
}

// makeRead: brainfuck implementation
func makeRead() vmFunc {
	return func(v *VM) {
		buf := make([]byte, 1)
		l, err := os.Stdin.Read(buf)
		if err != nil {
			v.err = err
			return
		}
		if l != 1 {
			v.err = fmt.Errorf("read %d bytes of input, not 1", l)
			return
		}
		v.memory[v.ptr] = int(buf[0])
		v.ip += 1
	}
}

// makeWrite: brainfuck implementation
func makeWrite() vmFunc {
	return func(v *VM) {
		// character to print
		c := v.memory[v.ptr]

		// newline?  show all pending output
		if c == '\n' {
			fmt.Printf("%s\n", v.stdout)
			v.stdout = ""
		} else {
			// otherwise save away
			v.stdout += string(v.memory[v.ptr])
		}
		v.ip += 1
	}
}

// makeLoopOpen: brainfuck implementation
func makeLoopOpen() vmFunc {
	return func(v *VM) {
		// early termination
		if v.memory[v.ptr] != 0x00 {
			v.ip += 1
			return
		}

		v.ip = v.loops[v.ip]
	}
}

// makeLoopClose: brainfuck implementation
func makeLoopClose() vmFunc {
	return func(v *VM) {

		// early termination
		if v.memory[v.ptr] == 0x00 {
			v.ip++
			return
		}

		v.ip = v.loops[v.ip]
	}
}

//
// Closure time is over now.
//

// main is our entry-point and creates/runs a program.
func main() {

	// No arguments?  Abort
	if len(os.Args) != 2 {
		fmt.Printf("Usage: simple-vm path/to/file.bf\n")
		return
	}

	// Read the file
	dat, err := os.ReadFile(os.Args[1])
	if err != nil {
		fmt.Printf("Error reading %s:%s\n", os.Args[1], err)
		return
	}

	// create an interpreter to run that program.
	v, err := New(string(dat))
	if err != nil {
		fmt.Printf("error compiling program: %s\n", err.Error())
		return
	}

	// now launch it
	err = v.RunProgram()
	if err != nil {
		fmt.Printf("error running program: %s\n", err)
	}
}
