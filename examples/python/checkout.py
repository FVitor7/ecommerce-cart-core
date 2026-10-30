"""Simulates a checkout: revalidates fixtures/cart.json with the offers in fixtures/offers.json.

Runs in the container of the Python package (make demo compares it with the Node.js output):

    docker compose run --rm python python /src/examples/python/checkout.py
"""

import json
from pathlib import Path

from ecommerce_cart_core import calculate

fixtures = Path(__file__).resolve().parents[2] / "fixtures"
cart = json.loads((fixtures / "cart.json").read_text(encoding="utf-8"))
offers = json.loads((fixtures / "offers.json").read_text(encoding="utf-8"))

result = calculate({**cart, **offers})
print(json.dumps(result, indent=2, ensure_ascii=False))
