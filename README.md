# simple vm

This repository contains a trivial virtual machine, to demonstrate the usage of an interpreter which is faster than using a naive bytecode-based approach.

Instead of compiling a program into a series of bytecode values, which are then interpreted by a giant switch-statement we can instead just compile a series of function-pointers - the function being the thing that does whatever is required of course.

Our interpreter is then very simple:

    ...
	for ip < len(code) {
		ip += code[ip](vm)
	}
    ..

The functions each return a value which controls where to execute next.  Most functions would return 1 to move to the next "instruction" but we could implement control-flow by bumping forwards/backwards as appropriate.



## Inspiration

This was inspired by the following article:

* [Faster interpreters in Go: Catching up with C++](https://planetscale.com/blog/faster-interpreters-in-go-catching-up-with-cpp)
  * [https://news.ycombinator.com/item?id=43595283](https://news.ycombinator.com/item?id=43595283)


The implementation here is simple enough to allow simple mathematical operations to be carried out, to prove the concept without getting bogged down in needless details.
