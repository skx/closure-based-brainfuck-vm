# A simple closure-based brainfuck interpreter

This repository contains a trivial interpreter for brainfuck, which is designed
to demonstrate the usage of an interpreter which is faster than using a naive
bytecode-based approach.

Traditionally an interpreter will start by creating an AST, then walking it.
When that works the next step is to update things to use the AST to generate
some bytecode which can then be executed - it might even be that moving straight
to bytecode makes sense as we know it is faster than the AST-walking approach.

This interpreter has no AST because brainfuck is such a simple language, but
rather than using the simple bytecode-based approach it instead uses a series
of closures to implement each "op".

This means the core of our loop is something like this:

    ...
	for ip < len(code) {
        code[ip](vm)
	}
    ..

In short the compilation stage just appends dynamic functions (i.e closures) to
a list, and the interpretation just involves walking that list of functions and
invoking each in turn - as the functions have a pointer to the VM-object they
can read/update the memory-cells etc.



## STDOUT Buffering

Brainfuck has only a pair of primitives for reading and writing to the console,
and these work on single bytes at a time.  By default we buffer writes to
STDOUT until we see a newline, which boosts the mandelbrot benchmark a decent
amount.

I think it's a legitimate optimization, but if you wish to disable it set the
environmental variable `BUFFER_STDOUT` to the literal string `false`.

You'll see that this has no runtime impact, the change here is in the
compilation phase where we use a different closure depending on the value of
the variable.



## Other Speed Notes

There are two things that this program is doing to be fast:

* Avoiding complex control-flow, by which I mean that regardless of the operation which is being carried out we don't have to do any switch-base lookup.  We literally jump around within a static array of function pointers.
* Avoiding doing too much work inside the closures.

As a case in point we calculate the offsets for the forward/backward jumps as we compile the program.  We could use that in our handlers to run something like:

     v.ip = v.loops[v.ip]

However that map-lookup is slow.  So instead we hardcode the offset in the closures for the loop open instructions and the same again for the loop close.  This hardcoding makes the program a little more complex as we have to rewrite closures however the results speak for themselves:

* Default: 35s
* Replacing the map looking for the loop-close only: 21s
* Replacing both map lookups: 17s



## Inspiration

This was inspired by the following article:

* [Faster interpreters in Go: Catching up with C++](https://planetscale.com/blog/faster-interpreters-in-go-catching-up-with-cpp)
  * [https://news.ycombinator.com/item?id=43595283](https://news.ycombinator.com/item?id=43595283)


It is hoped the implementation here is simple enough to understand, and still
demonstrate a decent speedup compared to the more obvious naive approaches.



## Links

If you like brainfuck you might want to see my brainfuck compiler challenge:

* [Brainfuck Compiler Challenge](https://github.com/skx/bfcc/)

If you want to see a traditional bytecode-based interpreter, written for learning:

* https://github.com/skx/go.vm

If you want a _real_ bytecode compiler written in go:

* https://github.com/skx/evalfilter

And for something different, but related:

* Writing a FORTH interpreter step by step, in golang
  * https://github.com/skx/foth
