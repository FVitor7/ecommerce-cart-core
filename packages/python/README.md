# ecommerce-cart-core for Python

E-commerce cart calculation. The rules are written once in Go and compiled to `calc.wasm`. This
package runs that file with [wasmtime](https://pypi.org/project/wasmtime/), and the npm package
ships the very same file, so Python, Node.js and the browser get the same totals, cent for cent.

## Install

Build the package at the repository root with `make build` (it only needs Docker), then:

```sh
pip install /path/to/dist/python/ecommerce_cart_core-0.1.0-py3-none-any.whl
```

Python 3.9 or later; tested on 3.9 to 3.15.

## Usage

```python
from ecommerce_cart_core import CartError, calculate

result = calculate({
    "cartItems": [
        {"sku": "COFFEE-500G", "ean": "7891234000019", "name": "Ground Coffee 500g",
         "price": 18.90, "quantity": 2, "stock": 40},
    ],
    "offers": [
        {"sku": "COFFEE-500G", "price": 18.90, "discountedPrice": 15.90},
    ],
})
result["summary"]["totalCents"]  # 3180
```

- `calculate` takes a `dict` or the cart's JSON text (`str` or `bytes`) and returns a `dict`,
  typed as `CartResult`.
- Prices may be `int`, `float`, `str` (`"18.90"`) or `Decimal`; each is read exactly, never
  rounded. Amounts come back as integer cents.
- Invalid input raises `CartError`, with `code` (e.g. `INVALID_INPUT`), `path` (e.g.
  `cartItems[0].price`) and `message`.
- It is thread-safe. The engine is created on the first call, once per process, and again in
  forked processes.

The input and output formats and the rules are in the [repository README](../../README.md).
