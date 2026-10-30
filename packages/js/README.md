# ecommerce-cart-core for JavaScript

E-commerce cart calculation. The rules are written once in Go and compiled to `calc.wasm`. This
package embeds that file, and the Python package ships the very same file, so Node.js, the browser
and Python get the same totals, cent for cent.

## Install

Build the package at the repository root with `make build` (it only needs Docker), then:

```sh
npm install /path/to/dist/js/ecommerce-cart-core-0.1.0.tgz
```

Node.js 18 or later; tested on 18 to 26. TypeScript types are included.

## Usage

```js
import { CartError, calculate } from 'ecommerce-cart-core';

const result = await calculate({
  cartItems: [
    { sku: 'COFFEE-500G', ean: '7891234000019', name: 'Ground Coffee 500g', price: 18.9, quantity: 2, stock: 40 },
  ],
  offers: [{ sku: 'COFFEE-500G', price: 18.9, discountedPrice: 15.9 }],
});
result.summary.totalCents; // 3180
```

- `calculate` takes the cart object or its JSON text and returns a `Promise` of the result.
- Prices may be numbers or decimal strings (`"18.90"`); each is read exactly, never rounded.
  Amounts come back as integer cents.
- Invalid input rejects with `CartError`, with `code` (e.g. `INVALID_INPUT`), `path` (e.g.
  `cartItems[0].price`) and `message`.
- It works without configuration in Node.js and in Next.js: Server Components, Route Handlers and
  Client Components. The `.wasm` is loaded on the first call, so the browser downloads it (about
  180 KB gzipped) only when `calculate` runs. The Edge Runtime is not supported.
- It is an ES module; CommonJS code can `require()` it on Node.js 20.19+ and 22.12+.

The input and output formats and the rules are in the [repository README](../../README.md).
