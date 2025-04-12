// package main contains a simple brainfuck interpreter which compiles
// a program into a series of closures which can then be iterated
// over in turn and executed.
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
	// input programs.  This will ensure the program terminates cleanly.
	ErrExit = errors.New("EXIT")
)

// VM contains the virtual machine, and it mostly exists to hold
// the state of our brainfuck program, the pointer, memory, etc.
type VM struct {

	// ip is the instruction pointer into the brainfuck
	// program which is to be executed.
	ip int

	// ptr is the brainfuck programs index offset.
	ptr int

	// memory is the memory-space the brainfuck program uses
	memory [30000]int

	// stdout holds output we should write to the console.
	//
	// We do this because writing a single byte to STDOUT is inefficient
	// and by buffering until we get a complete line we get a little
	// speed-boost.
	stdout string

	// program contains the set of closures that we can
	// execute one by one, to run the actual compiled brainfuck program.
	program []vmFunc
}

// vmFunc is the type-signature of our closures.
//
// Each closure will have access to the VM object, which means it can bump the
// ip pointer, to move to the next instruction, update the ptr, or memory, and
// do similar things.
type vmFunc func(vm *VM) error

// New is the VM constructor which takes our program as input
// and compiles it into a series of closures.
func New(bf string) (*VM, error) {

	// Ensure we got a program
	if len(bf) < 1 {
		return nil, errors.New("empty program is invalid")
	}

	// Create empty VM
	v := VM{}

	// Setup a stack we can use to match loops as we compile
	loopStack := []int{}

	// Should we buffer writes to STDOUT?
	buffer := true
	if os.Getenv("BUFFER_STDOUT") == "false" {
		buffer = false
	}

	// Index and bounds for walking the string of brainfuck source code
	i := 0
	max := len(bf)

	// inline function - designed to count how many consecutive times
	// we see the given character, c, repeated.  Returns the count and
	// the updated index variable for the program source.
	//
	// This is a bit horrid, but avoids repetition in the handlers.
	countRepeats := func(i int, c byte) (int, int) {
		// Record our starting position in the program source.
		begin := i

		// See if this character is repeated.
		for i < max {

			// Not a repeat?  Stop
			if bf[i] != c {
				break
			}

			// Otherwise keep advancing forward
			i++
		}

		// How many consecutive "+" did we see?
		count := i - begin

		// We'll end with an i++ so counter that
		i--

		return count, i
	}

	// Walk over the input program
	for i < max {

		// The character we're looking at right now.
		c := bf[i]

		// Handle each known character.
		switch c {
		case '+':
			// Count how many times "+" was repeated
			count := 0
			count, i = countRepeats(i, c)

			v.program = append(v.program, makeIncCell(count))
		case '-':
			// Count how many times "-" was repeated
			count := 0
			count, i = countRepeats(i, c)

			v.program = append(v.program, makeDecCell(count))
		case '<':
			// Count how many times "<" was repeated
			count := 0
			count, i = countRepeats(i, c)

			v.program = append(v.program, makeDecPtr(count))
		case '>':
			// Count how many times ">" was repeated
			count := 0
			count, i = countRepeats(i, c)

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

			// This will get replaced later, but we need to add _something_
			// to keep our offsets neat.
			v.program = append(v.program, nil)

		case ']':
			// So this is a loop-close, and we've got a stack which contains
			// the loop-start.
			//
			// Pop off the topmost value, which is our loop open.
			openInstruction := loopStack[len(loopStack)-1]
			loopStack = loopStack[:len(loopStack)-1]

			// We want the open-instruction to point to the position of the
			// close instruction we're just going to compile, so that's the
			// length of the program:
			v.program[openInstruction] = makeLoopOpen(len(v.program))

			// Now add the instruction itself, which will jump back to the
			// loop opening.
			v.program = append(v.program, makeLoopClose(openInstruction))
		default:
			// Invalid character.
			// ignored.
		}
		i++
	}

	// Finally add a fake "exit" trap to the end of our program
	v.program = append(v.program, makeExit())

	// Return the VM, we're now ready to be executed.
	return &v, nil
}

// RunProgram executes the program which was given in the constructor.
func (vm *VM) RunProgram() error {

	// Reset the state of the program each run.
	vm.ptr = 0
	vm.ip = 0

	// Hold errors from the closures
	var err error

	// For each operation.  Run it
	for {

		// Call the closure.
		//
		// Here we assume that each opcode ends with
		// "vm.ip++", which lets us run forward.
		err = vm.program[vm.ip](vm)

		// Did we get an error?
		if err != nil {

			// Show any pending output
			if vm.stdout != "" {
				fmt.Printf("%s\n", vm.stdout)
				vm.stdout = ""
			}

			// If it is the fake exit-program error
			// then we ignore it.
			if err == ErrExit {
				return nil
			}

			// otherwise return the error to the caller
			return err
		}
	}
}

// Okay here we write some helpers which create/return closures

// makeExit adds a closure which terminates execution.
func makeExit() vmFunc {
	return func(v *VM) error {
		return ErrExit
	}
}

// makeIncCell implements the brainfuck cell-increment operation.
func makeIncCell(n int) vmFunc {
	return func(v *VM) error {
		v.memory[v.ptr] += n
		v.ip++
		return nil
	}
}

// makeDecCell implements the brainfuck cell-decrement operation.
func makeDecCell(n int) vmFunc {
	return func(v *VM) error {
		v.memory[v.ptr] -= n
		v.ip++
		return nil
	}
}

// makeIncPtr implements the brainfuck ptr-increment operation.
func makeIncPtr(n int) vmFunc {
	return func(v *VM) error {
		v.ptr += n
		v.ip++
		return nil
	}
}

// makeDecPtr implements the brainfuck ptr-decrement operation.
func makeDecPtr(n int) vmFunc {
	return func(v *VM) error {
		v.ptr -= n
		v.ip++
		return nil
	}
}

// makeRead implements the brainfuck STDIN-reading operation.
func makeRead() vmFunc {
	return func(v *VM) error {
		buf := make([]byte, 1)
		l, err := os.Stdin.Read(buf)
		if err != nil {
			return err
		}
		if l != 1 {
			return fmt.Errorf("read %d bytes of input, not 1", l)
		}
		v.memory[v.ptr] = int(buf[0])
		v.ip++
		return nil
	}
}

// makeWrite implements the brainfuck STDOUT-writing operation, with no caching.
func makeWrite() vmFunc {
	return func(v *VM) error {
		fmt.Printf("%c", v.memory[v.ptr])
		v.ip++
		return nil
	}
}

// makeWriteCached implements the brainfuck STDOUT-writing operation.
// We cache output until we see a newline as a minor optimization.
func makeWriteCached() vmFunc {
	return func(v *VM) error {
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

		return nil
	}
}

// makeLoopOpen implements the brainfuck loop opening operation.
func makeLoopOpen(offset int) vmFunc {
	return func(v *VM) error {
		// early termination
		if v.memory[v.ptr] != 0x00 {
			v.ip++
			return nil
		}

		v.ip = offset
		return nil
	}
}

// makeLoopClose implements the brainfuck loop closing operation.
func makeLoopClose(offset int) vmFunc {
	return func(v *VM) error {

		// early termination
		if v.memory[v.ptr] == 0x00 {
			v.ip++
			return nil
		}

		v.ip = offset
		return nil
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
