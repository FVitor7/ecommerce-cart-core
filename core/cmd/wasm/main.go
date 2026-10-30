//go:build tinygo.wasm

// Command wasm builds calc.wasm, the cart engine as a WebAssembly module with
// no imports, so the very same binary runs under wasmtime (Python), Node.js
// and browsers. Build it with:
//
//	tinygo build -target=wasm-unknown -gc=conservative -panic=trap -no-debug -o calc.wasm ./cmd/wasm
//
// The wasm-unknown target defaults to a GC that never frees memory, which
// would make a long-lived instance grow forever; -gc=conservative avoids that.
//
// ABI. After instantiating, the host calls _initialize() once. Then, per call:
//
//	alloc(size i32) -> ptr i32
//	    Reserves size bytes for the host to write the input JSON into.
//	calculate(ptr i32, size i32) -> i64
//	    Runs cart.CalculateJSON on those bytes and returns the response JSON
//	    packed as (outPtr << 32 | outLen). Read it as an unsigned 64-bit value.
//	dealloc(ptr i32)
//	    Releases a buffer returned by alloc or calculate. The host releases
//	    both the input and the response.
package main

import (
	"unsafe"

	"ecommerce-cart-core/core/cart"
)

// buffers keeps the memory shared with the host alive until it is released.
var buffers = map[uint32][]byte{}

//go:wasmexport alloc
func alloc(size uint32) uint32 {
	return keep(make([]byte, size))
}

//go:wasmexport dealloc
func dealloc(ptr uint32) {
	delete(buffers, ptr)
}

//go:wasmexport calculate
func calculate(ptr, size uint32) uint64 {
	input, ok := buffers[ptr]
	if !ok || uint32(len(input)) < size {
		out := keep([]byte(`{"error":{"code":"INTERNAL_ERROR","message":"invalid input buffer"}}`))
		return uint64(out)<<32 | uint64(len(buffers[out]))
	}
	output := cart.CalculateJSON(input[:size])
	return uint64(keep(output))<<32 | uint64(len(output))
}

func keep(b []byte) uint32 {
	if cap(b) == 0 {
		b = make([]byte, 0, 1) // a zero-size slice has no address of its own
	}
	ptr := uint32(uintptr(unsafe.Pointer(unsafe.SliceData(b))))
	buffers[ptr] = b
	return ptr
}

func main() {}
