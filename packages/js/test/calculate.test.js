import assert from 'node:assert/strict';
import { createHash } from 'node:crypto';
import { existsSync, readdirSync, readFileSync } from 'node:fs';
import { test } from 'node:test';

import { wasmBase64, wasmSha256 } from '../src/calc-wasm.js';
import { CartError, calculate } from '../src/index.js';

const fixtures = new URL('../../../fixtures/', import.meta.url);
const readJSON = (path) => JSON.parse(readFileSync(new URL(path, fixtures), 'utf8'));
const sha256 = (bytes) => createHash('sha256').update(bytes).digest('hex');

test('checkout: cart.json + offers.json gives expected.json', async () => {
  const cart = { ...readJSON('cart.json'), ...readJSON('offers.json') };
  assert.deepEqual(await calculate(cart), readJSON('expected.json'));
});

for (const file of readdirSync(new URL('scenarios/', fixtures)).filter((f) => f.endsWith('.json')).sort()) {
  test(`scenario ${file}`, async () => {
    const scenario = readJSON(`scenarios/${file}`);
    const input = scenario.rawInput ?? scenario.input;
    if (scenario.expectedError) {
      await assert.rejects(calculate(input), (err) => {
        assert.ok(err instanceof CartError);
        assert.deepEqual({ code: err.code, message: err.message, path: err.path }, { path: undefined, ...scenario.expectedError });
        return true;
      });
    } else {
      assert.deepEqual(await calculate(input), scenario.expected);
    }
  });
}

test('accepts the cart as JSON text', async () => {
  const text = JSON.stringify({ ...readJSON('cart.json'), ...readJSON('offers.json') });
  assert.deepEqual(await calculate(text), readJSON('expected.json'));
});

test('floats drift, cents do not', async () => {
  const cart = { cartItems: [{ sku: 'BAG', price: 0.1, quantity: 3, stock: 10 }] };
  assert.notEqual(0.1 * 3, 0.3);
  assert.equal((await calculate(cart)).summary.totalCents, 30);
});

test('CartError points at the offending field', async () => {
  const cart = { cartItems: [{ sku: 'A', price: '49,90', quantity: 1, stock: 1 }] };
  await assert.rejects(calculate(cart), {
    name: 'CartError',
    code: 'INVALID_INPUT',
    path: 'cartItems[0].price',
    message: 'cartItems[0].price: invalid amount: "49,90"',
  });
});

test('values that are not JSON are rejected before reaching the engine', async () => {
  await assert.rejects(calculate({ cartItems: [], total: 10n }), { name: 'CartError', code: 'INVALID_INPUT' });
});

test('concurrent calls share one engine safely', async () => {
  const cart = { ...readJSON('cart.json'), ...readJSON('offers.json') };
  const results = await Promise.all(Array.from({ length: 200 }, () => calculate(cart)));
  for (const result of results) {
    assert.deepEqual(result, readJSON('expected.json'));
  }
});

test('the embedded engine is the calc.wasm shipped in the package', () => {
  assert.equal(sha256(Buffer.from(wasmBase64, 'base64')), wasmSha256);
  assert.equal(sha256(readFileSync(new URL('../calc.wasm', import.meta.url))), wasmSha256);
});

// The Python package checks the same, so both packages carry the very same engine.
const built = new URL('../../../dist/calc.wasm', import.meta.url);
test('the packaged calc.wasm is the one built from core', { skip: !existsSync(built) && 'dist/calc.wasm not built (make build-go)' }, () => {
  assert.equal(sha256(readFileSync(built)), wasmSha256);
});

test('calc.wasm frees memory between calls (built with -gc=conservative)', async () => {
  const { instance } = await WebAssembly.instantiate(Buffer.from(wasmBase64, 'base64'));
  const { memory, alloc, dealloc, calculate: calc, _initialize } = instance.exports;
  _initialize();
  const input = new TextEncoder().encode(JSON.stringify({ ...readJSON('cart.json'), ...readJSON('offers.json') }));
  for (let i = 0; i < 10_000; i++) {
    const ptr = alloc(input.length);
    new Uint8Array(memory.buffer, ptr, input.length).set(input);
    const packed = calc(ptr, input.length);
    dealloc(ptr);
    dealloc(Number(BigInt.asUintN(32, packed >> 32n)));
  }
  // Without a collecting GC, 10k calls take hundreds of MB.
  assert.ok(memory.buffer.byteLength <= 4 * 1024 * 1024, `memory grew to ${memory.buffer.byteLength} bytes`);
});
