# simple vm

This repository contains a trivial interpreter for brainfuck, which
is designed to demonstrate the usage of an interpreter which is faster than using a naive bytecode-based approach, and which instead uses
a series of closures to implement each "op".

Instead of compiling a program into a series of bytecode values, which are then interpreted by a giant switch-statement we compile each small
operation into a closure which can be implemented without the switch
overhead, like so:

    ...
	for ip < len(code) {
        code[ip](vm)
	}
    ..

The functions each do their thing, and bump the IP to move to the
next instruction.



## STDOUT Buffering

Brainfuck has only a pair of primitives for reading and writing to the console, and these work on single bytes at a time.  By default we buffer writes to STDOUT until we see a newline, which boosts the mandelbrot benchmark a decent amount.

I think it's a legitimate optimization, but if you wish to disable it set the environmental variable `BUFFER_STDOUT` to the literal string `false`.

You'll see that this has no runtime impact, the change here is in the compilation phase where we use a different closure depending on the value of the variable.



## Inspiration

This was inspired by the following article:

* [Faster interpreters in Go: Catching up with C++](https://planetscale.com/blog/faster-interpreters-in-go-catching-up-with-cpp)
  * [https://news.ycombinator.com/item?id=43595283](https://news.ycombinator.com/item?id=43595283)


The implementation here is simple enough to allow simple mathematical operations to be carried out, to prove the concept without getting bogged down in needless details.
