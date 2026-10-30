import hashlib
import json
from concurrent.futures import ThreadPoolExecutor
from decimal import Decimal
from importlib import resources
from pathlib import Path

import pytest

from ecommerce_cart_core import CartError, calculate

ROOT = Path(__file__).resolve().parents[3]
FIXTURES = ROOT / "fixtures"


def load(name):
    return json.loads((FIXTURES / name).read_text(encoding="utf-8"))


def checkout_cart():
    return {**load("cart.json"), **load("offers.json")}


def test_checkout_fixture():
    assert calculate(checkout_cart()) == load("expected.json")


@pytest.mark.parametrize("path", sorted((FIXTURES / "scenarios").glob("*.json")), ids=lambda p: p.stem)
def test_scenario(path):
    scenario = json.loads(path.read_text(encoding="utf-8"))
    cart = scenario["rawInput"] if "rawInput" in scenario else scenario["input"]
    if "expectedError" in scenario:
        with pytest.raises(CartError) as info:
            calculate(cart)
        error = info.value
        assert {"code": error.code, "message": error.message, "path": error.path} == {
            "path": None,
            **scenario["expectedError"],
        }
    else:
        assert calculate(cart) == scenario["expected"]


def test_accepts_json_text_and_bytes():
    text = json.dumps(checkout_cart())
    assert calculate(text) == load("expected.json")
    assert calculate(text.encode()) == load("expected.json")


def test_decimal_prices_are_exact():
    cart = {"cartItems": [{"sku": "A", "price": Decimal("18.90"), "quantity": 3, "stock": 10}]}
    assert calculate(cart)["summary"]["totalCents"] == 5670


def test_floats_drift_cents_do_not():
    cart = {"cartItems": [{"sku": "BAG", "price": 0.1, "quantity": 3, "stock": 10}]}
    assert 0.1 * 3 != 0.3
    assert calculate(cart)["summary"]["totalCents"] == 30


def test_error_points_at_the_offending_field():
    cart = {"cartItems": [{"sku": "A", "price": "49,90", "quantity": 1, "stock": 1}]}
    with pytest.raises(CartError, match=r'cartItems\[0\]\.price: invalid amount: "49,90"') as info:
        calculate(cart)
    assert (info.value.code, info.value.path) == ("INVALID_INPUT", "cartItems[0].price")


@pytest.mark.parametrize("value", [float("nan"), {1, 2}])
def test_values_that_are_not_json_are_rejected(value):
    with pytest.raises(CartError) as info:
        calculate({"cartItems": [], "extra": value})
    assert info.value.code == "INVALID_INPUT"


def test_thread_safe():
    with ThreadPoolExecutor(max_workers=8) as pool:
        results = list(pool.map(lambda _: calculate(checkout_cart()), range(200)))
    assert all(result == load("expected.json") for result in results)


def test_packaged_calc_wasm_is_the_one_built_from_core():
    # The npm package checks the same, so both packages carry the very same engine.
    ours = resources.files("ecommerce_cart_core").joinpath("calc.wasm").read_bytes()
    built = ROOT / "dist" / "calc.wasm"
    if not built.exists():
        pytest.skip("dist/calc.wasm not built (make build-go)")
    assert hashlib.sha256(ours).hexdigest() == hashlib.sha256(built.read_bytes()).hexdigest()
