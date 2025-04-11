// package main contains a simple brainfuck interpreter which compiles
// a program into a series of closures which can then be iterated
// over in turn.
//
// The intent behind this implementation is to show how a reasonably
// fast and compact interpreter might work, despite being built without
// the traditional approach of compiling to bytecode and then walking
// that bytecode via a switch-operation.
//
// Bytecode interpreters are certainly faster than AST-walking interpreters
// however they have the downside that their implementation requires the
// generation and execution halves to be keep in sync:
//
// * Some part of the code emits bytecode for each operation.
//
// * Some part of the code "does stuff" for each bytecode opcode.
//
// Keeping those two things in sync is annoying overhead, and also
// the traditional "switch-based" interpretation of bytecode opcodes
// is slow, especially in golang.
//
// The idea implemented here is that we switch our core interpreter loop
// into merely executing a series of function-pointers over and over again
// until the program is complete.   These function-pointers refer to a series
// of generated closures which our "compiler" has prepared/created.
//
// As each individual closure is short, and almost exclusively branchless,
// we gain a speedup based on the absence of the cache-busting switch statement
// we would otherwise be hit by, and we _also_ gain from the lack of code
// duplication.
package main

import (
	"errors"
	"fmt"
	"os"
)

var (
	// ErrExit is a "fake error" we append to the end of all our
	// input programs.  These terminates execution cleanly.
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

	// loops is used to lookup look bounds - these is populated
	// in the constructor, New.
	loops map[int]int

	// stdout holds output we should write to the console.
	//
	// We do this because writing a single byte to STDOUT is inefficient
	// and by buffering until we get a complete line we get a little
	// speed-boost.
	stdout string

	// driver

	// program contains the set of closures that we can
	// execute one by one, to run the actual compiled brainfuck program.
	program []vmFunc

	// err records any error received when running the brainfuck program.
	err error
}

// New is the VM constructor which takes our program as input
// and compiles it into a series of closures.
func New(bf string) (*VM, error) {

	// Create empty VM
	v := VM{}

	// Setup a map for storing loop start/end pairs, along with
	// a stack we can use to populate those as we compile.
	v.loops = make(map[int]int)
	loopStack := []int{}

	// Ensure we got a program
	if len(bf) < 1 {
		return nil, errors.New("empty program is invalid")
	}

	// Index and bounds
	i := 0
	max := len(bf)

	// Should we buffer writes to STDOUT?
	buffer := true
	if os.Getenv("BUFFER_STDOUT") == "false" {
		buffer = false
	}

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

			i--
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

			i--
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

			i--
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

			i--
			v.program = append(v.program, makeIncPtr(count))
		case ',':
			v.program = append(v.program, makeRead())
		case '.':
			if buffer {
				v.program = append(v.program, makeWriteCached())
			} else {
				v.program = append(v.program, makeWrite())
			}
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
	len := len(vm.program)

	// For each operation.  Run it
	for vm.ip < len {

		// Call the closure.
		//
		// Here we assume that each opcode ends with
		// "vm.ip++", which lets us run forward.
		vm.program[vm.ip](vm)

		// Did we get an error?
		if vm.err != nil {

			// Show any pending output
			if vm.stdout != "" {
				fmt.Printf("%s\n", vm.stdout)
				vm.stdout = ""
			}

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

// vmFunc is the type-signature of our closures.
type vmFunc func(vm *VM)

// makeExit adds a closure which terminates execution.
func makeExit() vmFunc {
	return func(v *VM) {
		v.err = ErrExit
	}
}

// makeIncCell implements the brainfuck cell-increment operation.
func makeIncCell(n int) vmFunc {
	return func(v *VM) {
		v.memory[v.ptr] += n
		v.ip++
	}
}

// makeDecCell implements the brainfuck cell-decrement operation.
func makeDecCell(n int) vmFunc {
	return func(v *VM) {
		v.memory[v.ptr] -= n
		v.ip++
	}
}

// makeIncPtr implements the brainfuck ptr-increment operation.
func makeIncPtr(n int) vmFunc {
	return func(v *VM) {
		v.ptr += n
		v.ip++
	}
}

// makeDecPtr implements the brainfuck ptr-decrement operation.
func makeDecPtr(n int) vmFunc {
	return func(v *VM) {
		v.ptr -= n
		v.ip++
	}
}

// makeRead implements the brainfuck STDIN-reading operation.
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
		v.ip++
	}
}

// makeWrite implements the brainfuck STDOUT-writing operation, with no caching.
func makeWrite() vmFunc {
	return func(v *VM) {
		fmt.Printf("%c", v.memory[v.ptr])
		v.ip++
	}
}

// makeWriteCached implements the brainfuck STDOUT-writing operation.
// We cache output until we see a newline as a minor optimization.
func makeWriteCached() vmFunc {
	return func(v *VM) {
		// character to print
		c := v.memory[v.ptr]

		// newline?  show all pending output
		if c == '\n' {
			fmt.Printf("%s\n", v.stdout)
			v.stdout = ""
		} else {
			// otherwise save away
			v.stdout += string(rune(v.memory[v.ptr]))
		}
		v.ip++
	}
}

// makeLoopOpen implements the brainfuck loop opening operation.
func makeLoopOpen() vmFunc {
	return func(v *VM) {
		// early termination
		if v.memory[v.ptr] != 0x00 {
			v.ip++
			return
		}

		v.ip = v.loops[v.ip]
	}
}

// makeLoopClose implements the brainfuck loop closing operation.
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

// main is our entry-point and reads/launches a brainfuck program
// from an external file.
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
