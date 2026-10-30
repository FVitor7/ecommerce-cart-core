"""E-commerce cart calculation.

The cart rules are written once in Go and compiled to calc.wasm; the npm
package embeds the very same binary, so every project gets the same totals,
cent for cent.

    >>> from ecommerce_cart_core import calculate
    >>> cart = {"cartItems": [{"sku": "BAG", "price": 0.10, "quantity": 3, "stock": 500}]}
    >>> calculate(cart)["summary"]["totalCents"]
    30
"""

from __future__ import annotations

import json
from decimal import Decimal
from typing import Any, Mapping

from ._engine import Engine, EngineFailure
from ._types import CartItemResult, CartResult, CartSummary, Issue, IssueCode, ItemStatus

__all__ = [
    "calculate",
    "CartError",
    "CartResult",
    "CartItemResult",
    "CartSummary",
    "Issue",
    "IssueCode",
    "ItemStatus",
]
__version__ = "0.1.0"

_engine = Engine()


class CartError(Exception):
    """Raised when the cart cannot be calculated.

    code is INVALID_JSON, INVALID_INPUT or ENGINE_ERROR; path points at the
    offending field, e.g. "cartItems[2].price", when there is one.
    """

    def __init__(self, code: str, message: str, path: str | None = None) -> None:
        super().__init__(message)
        self.code = code
        self.message = message
        self.path = path


def calculate(cart: Mapping[str, Any] | str | bytes) -> CartResult:
    """Revalidates the cart against stock and offers and computes its totals
    in integer cents.

    cart is {"cartItems": [...], "offers": [...]}, or the same as JSON text.
    Prices may be int, float, str ("49.90") or Decimal; each is read exactly,
    never rounded.
    """
    if isinstance(cart, bytes):
        data = cart
    elif isinstance(cart, str):
        data = cart.encode()
    else:
        try:
            data = json.dumps(cart, ensure_ascii=False, allow_nan=False, default=_to_json).encode()
        except (TypeError, ValueError) as err:
            raise CartError("INVALID_INPUT", f"the cart cannot be converted to JSON: {err}") from err

    try:
        output = _engine.call(data)
    except EngineFailure as err:
        raise CartError("ENGINE_ERROR", f"the calculation engine failed: {err}") from err

    response = json.loads(output)
    if "error" in response:
        error = response["error"]
        raise CartError(error["code"], error["message"], error.get("path"))
    return response["result"]


def _to_json(value: Any) -> Any:
    if isinstance(value, Decimal):
        return str(value)  # a decimal string, which the engine reads exactly
    raise TypeError(f"Object of type {type(value).__name__} is not JSON serializable")
