"""Runs calc.wasm under wasmtime."""

from __future__ import annotations

import os
import threading
from importlib import resources

import wasmtime

# Raised by wasmtime when the module traps or cannot be loaded. Trap does not
# derive from WasmtimeError, so both are listed.
EngineFailure = (wasmtime.Trap, wasmtime.WasmtimeError)


class Engine:
    """A calc.wasm instance shared by the whole process.

    A wasmtime Store must not be used by two threads at once, so calls are
    serialized; each one takes microseconds. The instance is created on first
    use, again in each forked child process (gunicorn, multiprocessing), and
    again after a trap, which could leave it inconsistent.
    """

    def __init__(self) -> None:
        self._lock = threading.Lock()
        self._pid: int | None = None
        self._instance: _Instance | None = None

    def call(self, data: bytes) -> bytes:
        with self._lock:
            if self._instance is None or self._pid != os.getpid():
                self._instance = _Instance()
                self._pid = os.getpid()
            try:
                return self._instance.call(data)
            except EngineFailure:
                self._instance = None
                raise


class _Instance:
    def __init__(self) -> None:
        wasm = resources.files(__package__).joinpath("calc.wasm").read_bytes()
        engine = wasmtime.Engine()
        self.store = wasmtime.Store(engine)
        instance = wasmtime.Instance(self.store, wasmtime.Module(engine, wasm), [])
        exports = instance.exports(self.store)
        self.memory = exports["memory"]
        self.alloc = exports["alloc"]
        self.dealloc = exports["dealloc"]
        self.calculate = exports["calculate"]
        exports["_initialize"](self.store)

    def call(self, data: bytes) -> bytes:
        store = self.store
        in_ptr = self.alloc(store, len(data))
        self.memory.write(store, data, in_ptr)
        # An i64 holding (out_ptr << 32 | out_len); wasmtime returns it signed.
        packed = self.calculate(store, in_ptr, len(data)) & 0xFFFF_FFFF_FFFF_FFFF
        self.dealloc(store, in_ptr)
        out_ptr, out_len = packed >> 32, packed & 0xFFFF_FFFF
        output = self.memory.read(store, out_ptr, out_ptr + out_len)
        self.dealloc(store, out_ptr)
        return bytes(output)
