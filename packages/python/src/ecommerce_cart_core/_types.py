"""Shapes of the result of calculate(). All amounts are integer cents."""

from __future__ import annotations

from typing import List, Literal, TypedDict

ItemStatus = Literal["ok", "adjusted", "unavailable"]

IssueCode = Literal[
    "DUPLICATE_SKU_MERGED",
    "QUANTITY_ADJUSTED",
    "OUT_OF_STOCK",
    "OFFER_PRICE_MISMATCH",
    "OFFER_WITHOUT_DISCOUNT",
    "EMPTY_CART",
]


class Issue(TypedDict):
    code: IssueCode
    message: str
    """Ready to show to the customer."""


class CartItemResult(TypedDict):
    sku: str
    ean: str
    name: str
    status: ItemStatus
    requestedQuantity: int
    quantity: int
    """What will be charged: min(requestedQuantity, stock)."""
    unitPriceCents: int
    finalUnitPriceCents: int
    """After the offer, if any."""
    subtotalCents: int
    discountCents: int
    totalCents: int
    offerApplied: bool
    issues: List[Issue]


class CartSummary(TypedDict):
    itemCount: int
    """Distinct products being bought."""
    unitCount: int
    """Units being bought."""
    subtotalCents: int
    discountCents: int
    totalCents: int


class CartResult(TypedDict):
    valid: bool
    """The cart can be checked out exactly as submitted: everything is available."""
    issues: List[Issue]
    """Cart-level issues. Item issues are in each item."""
    summary: CartSummary
    items: List[CartItemResult]
    """One entry per distinct SKU, in order of first appearance."""
