# Alternative API generator for go-capnproto2

This is a drop in replacement for the capnpc-go code generator in zombiezen.com/go/capnproto2.

It generates an API that has functions that panic where the original would have returned an error.
This makes it less cumbersome to access fields in decoded messages.  The errors that were returned
were largely of 3 categories:
- programming errors
- allocation failures
- corrupt structures because of decoding corrupt messages

In all cases, it is more convenient to deal with them in a defer/recover.

	defer() func(){
		if err := recover() {
			// handle err
		}
	}()

## Getting started

Familiarize yourself with

- [godoc]: https://godoc.org/zombiezen.com/go/capnproto2
- [capnproto]: https://capnproto.org/

Install this code generator

	go get -u github.com/lvdlvd/capnpc-gopanic

Instead of 

	capnp compile -ogo ....

invoke

	capnp compile -ogo-panic ....

Read the godoc of your generated code to see what the api looks like.
