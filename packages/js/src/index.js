// The cart rules are written once in Go and compiled to calc.wasm; the Python
// package embeds the very same binary, so every project gets the same totals.
//
// calc.wasm has no imports and is embedded as base64 (see calc-wasm.js), so
// this file runs unchanged in Node.js, Next.js Server Components, Route
// Handlers and Client Components, and plain browsers.

/** Thrown when the cart cannot be calculated. */
export class CartError extends Error {
  /**
   * @param {{ code: string, message: string, path?: string }} error
   * @param {ErrorOptions} [options]
   */
  constructor({ code, message, path }, options) {
    super(message, options);
    this.name = 'CartError';
    this.code = code;
    this.path = path;
  }
}

/**
 * Revalidates the cart against stock and offers and computes its totals in
 * integer cents.
 *
 * @param {import('../index.js').CartInput | string} cart
 *   `{ cartItems, offers? }`, or the same as JSON text.
 * @returns {Promise<import('../index.js').CartResult>}
 * @throws {CartError}
 */
export async function calculate(cart) {
  let json;
  try {
    json = typeof cart === 'string' ? cart : (JSON.stringify(cart) ?? '');
  } catch (err) {
    throw new CartError(
      { code: 'INVALID_INPUT', message: `the cart cannot be converted to JSON: ${err.message}` },
      { cause: err },
    );
  }

  const run = await loadEngine();
  let output;
  try {
    output = run(json);
  } catch (err) {
    // A trap can leave the instance inconsistent: the next call starts a new one.
    engine = undefined;
    throw new CartError(
      { code: 'ENGINE_ERROR', message: `the calculation engine failed: ${err.message}` },
      { cause: err },
    );
  }

  const response = JSON.parse(output);
  if (response.error) {
    throw new CartError(response.error);
  }
  return response.result;
}

/** @type {Promise<(json: string) => string> | undefined} */
let engine;

function loadEngine() {
  engine ??= instantiate().catch((err) => {
    engine = undefined;
    throw new CartError(
      { code: 'ENGINE_ERROR', message: `could not load the calculation engine: ${err.message}` },
      { cause: err },
    );
  });
  return engine;
}

async function instantiate() {
  // Imported on first use, so bundlers put the ~440 KB of base64 in a chunk of
  // its own instead of in the bundle of every page that imports this package.
  const { wasmBase64 } = await import('./calc-wasm.js');
  const { instance } = await WebAssembly.instantiate(decodeBase64(wasmBase64));
  const { memory, alloc, dealloc, calculate, _initialize } = instance.exports;
  _initialize();

  const encoder = new TextEncoder();
  const decoder = new TextDecoder();
  return (json) => {
    const input = encoder.encode(json);
    const inPtr = alloc(input.length);
    new Uint8Array(memory.buffer, inPtr, input.length).set(input);
    // An i64 holding (outPtr << 32 | outLen), as a signed BigInt.
    const packed = calculate(inPtr, input.length);
    dealloc(inPtr);
    const outPtr = Number(BigInt.asUintN(32, packed >> BigInt(32)));
    const outLen = Number(BigInt.asUintN(32, packed));
    // memory.buffer is read again: the module may have grown its memory.
    const output = decoder.decode(new Uint8Array(memory.buffer, outPtr, outLen));
    dealloc(outPtr);
    return output;
  };
}

function decodeBase64(base64) {
  const binary = atob(base64);
  const bytes = new Uint8Array(binary.length);
  for (let i = 0; i < binary.length; i++) {
    bytes[i] = binary.charCodeAt(i);
  }
  return bytes;
}
