// Package cart holds the e-commerce cart business rules: it revalidates a cart
// against stock and special offers and computes its totals in integer cents.
//
// It is the single source of truth for those rules. cmd/wasm compiles it with
// TinyGo into calc.wasm, which the Python and npm packages embed, so every
// project computes the same totals, cent for cent. Go services can import this
// package directly.
package cart

import "fmt"

// Cart is the input of Calculate.
type Cart struct {
	Items  []Item
	Offers []Offer
}

// Item is a cart line as sent by the caller, with catalog data (price, stock)
// for the product.
type Item struct {
	SKU      string // required; identifies the line
	EAN      string // optional; EAN-8, UPC-A, EAN-13 or GTIN-14
	Name     string
	Price    Cents // regular unit price
	Quantity int64 // requested quantity, at least 1
	Stock    int64 // units available
}

// Offer is a special price for a product: it was Price, it is now
// DiscountedPrice. It identifies the product by SKU and/or EAN; every
// identifier it provides must match the cart item. It applies only while Price
// is still the item's current price, so a stale offer never sells below what
// was intended.
type Offer struct {
	SKU             string
	EAN             string
	Price           Cents // the regular price the offer was created for
	DiscountedPrice Cents // the offer price
}

// Result is the revalidated cart.
type Result struct {
	// Valid reports whether the cart can be checked out exactly as submitted:
	// every item is fully available and there is something to buy.
	Valid   bool         `json:"valid"`
	Issues  []Issue      `json:"issues"`
	Summary Summary      `json:"summary"`
	Items   []ItemResult `json:"items"`
}

// Summary holds the cart totals. TotalCents = SubtotalCents − DiscountCents.
type Summary struct {
	ItemCount     int64 `json:"itemCount"` // distinct products being bought
	UnitCount     int64 `json:"unitCount"` // units being bought
	SubtotalCents Cents `json:"subtotalCents"`
	DiscountCents Cents `json:"discountCents"`
	TotalCents    Cents `json:"totalCents"`
}

// ItemResult is a revalidated cart line. Lines with the same SKU are merged,
// so there is one ItemResult per distinct SKU, in order of first appearance.
type ItemResult struct {
	SKU                 string  `json:"sku"`
	EAN                 string  `json:"ean"`
	Name                string  `json:"name"`
	Status              string  `json:"status"`
	RequestedQuantity   int64   `json:"requestedQuantity"`
	Quantity            int64   `json:"quantity"` // what will be charged: min(requested, stock)
	UnitPriceCents      Cents   `json:"unitPriceCents"`
	FinalUnitPriceCents Cents   `json:"finalUnitPriceCents"` // after the offer, if any
	SubtotalCents       Cents   `json:"subtotalCents"`       // UnitPriceCents × Quantity
	DiscountCents       Cents   `json:"discountCents"`
	TotalCents          Cents   `json:"totalCents"` // FinalUnitPriceCents × Quantity
	OfferApplied        bool    `json:"offerApplied"`
	Issues              []Issue `json:"issues"`
}

// Item statuses.
const (
	StatusOK          = "ok"          // available at the requested quantity
	StatusAdjusted    = "adjusted"    // quantity reduced to the available stock
	StatusUnavailable = "unavailable" // out of stock, not charged
)

// Issue explains something revalidation found. Code is stable and meant for
// programs; Message is ready to show to the customer.
type Issue struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// Issue codes.
const (
	IssueDuplicateSKUMerged   = "DUPLICATE_SKU_MERGED"
	IssueQuantityAdjusted     = "QUANTITY_ADJUSTED"
	IssueOutOfStock           = "OUT_OF_STOCK"
	IssueOfferPriceMismatch   = "OFFER_PRICE_MISMATCH"
	IssueOfferWithoutDiscount = "OFFER_WITHOUT_DISCOUNT"
	IssueEmptyCart            = "EMPTY_CART" // cart level
)

func newIssue(code, format string, args ...any) Issue {
	return Issue{Code: code, Message: fmt.Sprintf(format, args...)}
}

// Error is returned when the input cannot be calculated at all. Path points
// at the offending field, e.g. "cartItems[2].price", and Message includes it.
type Error struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Path    string `json:"path,omitempty"`
}

func (e *Error) Error() string { return e.Message }

// Error codes.
const (
	ErrInvalidJSON  = "INVALID_JSON"  // the input is not valid JSON
	ErrInvalidInput = "INVALID_INPUT" // valid JSON, but not a valid cart
)

func invalidInput(path, format string, args ...any) *Error {
	msg := fmt.Sprintf(format, args...)
	if path != "" {
		msg = path + ": " + msg
	}
	return &Error{Code: ErrInvalidInput, Message: msg, Path: path}
}
