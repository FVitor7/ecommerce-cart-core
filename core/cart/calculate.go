package cart

import (
	"fmt"
	"strings"
)

// Calculate revalidates the cart and computes its totals. The rules, in order:
//
//  1. Validation. Bad values (negative price, quantity below 1, invalid EAN,
//     an offer with neither SKU nor EAN...) make the whole calculation fail
//     with an *Error, as they mean the calling system has a bug.
//  2. Lines with the same SKU are merged, adding up their quantities, so stock
//     is checked against the total; they must agree on price, stock and EAN.
//  3. Stock: the quantity is capped at the available stock (status
//     "adjusted"); with no stock the item is "unavailable" and costs nothing.
//  4. Offers: an offer matches an item when every identifier it has (SKU,
//     EAN) is the item's. It is applied only when the regular price it was
//     created for is still the item's price and its offer price is lower.
//     Among applicable offers the lowest price wins, ties going to the first
//     one listed.
//  5. Totals are exact integer cents: subtotal = unit price × quantity,
//     total = final unit price × quantity, discount = subtotal − total.
//
// The result is Valid when every item is fully available and there is at
// least one unit to buy.
func Calculate(c Cart) (Result, error) {
	lines, err := mergeLines(c.Items)
	if err != nil {
		return Result{}, err
	}
	offers, err := checkOffers(c.Offers)
	if err != nil {
		return Result{}, err
	}

	res := Result{Valid: true, Issues: []Issue{}, Items: make([]ItemResult, 0, len(lines))}
	sum := &res.Summary
	for _, l := range lines {
		item, err := l.price(offers)
		if err != nil {
			return Result{}, err
		}
		res.Items = append(res.Items, item)
		if item.Status != StatusOK {
			res.Valid = false
		}
		if item.Quantity > 0 {
			sum.ItemCount++
		}
		var okUnits, okSubtotal bool
		sum.UnitCount, okUnits = addSafe(sum.UnitCount, item.Quantity)
		sum.SubtotalCents, okSubtotal = addSafe(sum.SubtotalCents, item.SubtotalCents)
		if !okUnits || !okSubtotal {
			return Result{}, invalidInput("cartItems", "the cart total exceeds the supported range")
		}
		// Bounded by the subtotal, so they cannot overflow.
		sum.DiscountCents += item.DiscountCents
		sum.TotalCents += item.TotalCents
	}
	if sum.UnitCount == 0 {
		res.Valid = false
		res.Issues = append(res.Issues, newIssue(IssueEmptyCart, "The cart has no items available for purchase."))
	}
	return res, nil
}

// line is a cart item after merging the lines that share its SKU.
type line struct {
	Item
	path  string // of the first line with this SKU
	gtin  string
	count int // cart lines merged into this one
}

func mergeLines(items []Item) ([]*line, error) {
	var lines []*line
	bySKU := make(map[string]*line, len(items))
	for i, it := range items {
		path := fmt.Sprintf("cartItems[%d]", i)
		gtin, err := checkItem(it, path)
		if err != nil {
			return nil, err
		}
		first, dup := bySKU[it.SKU]
		if !dup {
			l := &line{Item: it, path: path, gtin: gtin, count: 1}
			bySKU[it.SKU] = l
			lines = append(lines, l)
			continue
		}
		switch {
		case it.Price != first.Price:
			return nil, invalidInput(path+".price", "SKU %q is already in %s with a different price", it.SKU, first.path)
		case it.Stock != first.Stock:
			return nil, invalidInput(path+".stock", "SKU %q is already in %s with a different stock", it.SKU, first.path)
		case gtin != first.gtin:
			return nil, invalidInput(path+".ean", "SKU %q is already in %s with a different EAN", it.SKU, first.path)
		}
		var ok bool
		if first.Quantity, ok = addSafe(first.Quantity, it.Quantity); !ok {
			return nil, invalidInput(path+".quantity", "the combined quantity of SKU %q exceeds the supported range", it.SKU)
		}
		first.count++
	}
	return lines, nil
}

// checkItem validates an item and returns its normalized GTIN ("" if no EAN).
func checkItem(it Item, path string) (string, error) {
	if strings.TrimSpace(it.SKU) == "" {
		return "", invalidInput(path+".sku", "must not be empty")
	}
	gtin, err := checkEAN(it.EAN, path)
	if err != nil {
		return "", err
	}
	if err := checkMoney(it.Price, path+".price"); err != nil {
		return "", err
	}
	if it.Quantity < 1 {
		return "", invalidInput(path+".quantity", "must be at least 1")
	}
	if it.Quantity > MaxSafeInteger {
		return "", invalidInput(path+".quantity", "out of the supported range")
	}
	if it.Stock < 0 {
		return "", invalidInput(path+".stock", "must not be negative")
	}
	if it.Stock > MaxSafeInteger {
		return "", invalidInput(path+".stock", "out of the supported range")
	}
	return gtin, nil
}

// offer is a validated Offer.
type offer struct {
	Offer
	gtin string
}

func checkOffers(offers []Offer) ([]offer, error) {
	checked := make([]offer, 0, len(offers))
	for i, o := range offers {
		path := fmt.Sprintf("offers[%d]", i)
		if o.SKU == "" && o.EAN == "" {
			return nil, invalidInput(path, "an offer must identify the product by sku or ean")
		}
		if o.SKU != "" && strings.TrimSpace(o.SKU) == "" {
			return nil, invalidInput(path+".sku", "must not be empty")
		}
		gtin, err := checkEAN(o.EAN, path)
		if err != nil {
			return nil, err
		}
		if err := checkMoney(o.Price, path+".price"); err != nil {
			return nil, err
		}
		if err := checkMoney(o.DiscountedPrice, path+".discountedPrice"); err != nil {
			return nil, err
		}
		checked = append(checked, offer{Offer: o, gtin: gtin})
	}
	return checked, nil
}

func checkEAN(ean, path string) (string, error) {
	if ean == "" {
		return "", nil
	}
	gtin, err := normalizeGTIN(ean)
	if err != nil {
		return "", invalidInput(path+".ean", "%s: %q", err, ean)
	}
	return gtin, nil
}

func checkMoney(c Cents, path string) error {
	if c < 0 {
		return invalidInput(path, "must not be negative")
	}
	if c > MaxSafeInteger {
		return invalidInput(path, "out of the supported range")
	}
	return nil
}

func (o *offer) matches(l *line) bool {
	return (o.SKU == "" || o.SKU == l.SKU) && (o.gtin == "" || o.gtin == l.gtin)
}

// price applies the stock and offer rules to a merged line.
func (l *line) price(offers []offer) (ItemResult, error) {
	r := ItemResult{
		SKU:                 l.SKU,
		EAN:                 l.EAN,
		Name:                l.Name,
		Status:              StatusOK,
		RequestedQuantity:   l.Quantity,
		Quantity:            min(l.Quantity, l.Stock),
		UnitPriceCents:      l.Price,
		FinalUnitPriceCents: l.Price,
		Issues:              []Issue{},
	}

	if l.count > 1 {
		r.Issues = append(r.Issues, newIssue(IssueDuplicateSKUMerged,
			"This product appeared in %d cart lines; their quantities were added up.", l.count))
	}

	switch {
	case r.Quantity == 0:
		r.Status = StatusUnavailable
		r.Issues = append(r.Issues, newIssue(IssueOutOfStock,
			"Out of stock; it will not be charged."))
	case r.Quantity < r.RequestedQuantity:
		r.Status = StatusAdjusted
		r.Issues = append(r.Issues, newIssue(IssueQuantityAdjusted,
			"Quantity reduced from %d to %d due to limited stock.", r.RequestedQuantity, r.Quantity))
	}

	best, rejected := l.bestOffer(offers)
	if best != nil {
		r.FinalUnitPriceCents = best.DiscountedPrice
		r.OfferApplied = true
	} else {
		r.Issues = append(r.Issues, rejected...)
	}

	subtotal, okSubtotal := mulSafe(r.UnitPriceCents, r.Quantity)
	total, okTotal := mulSafe(r.FinalUnitPriceCents, r.Quantity)
	if !okSubtotal || !okTotal {
		return ItemResult{}, invalidInput(l.path, "the item total exceeds the supported range")
	}
	r.SubtotalCents = subtotal
	r.TotalCents = total
	r.DiscountCents = subtotal - total
	return r, nil
}

// bestOffer returns the applicable offer with the lowest price or, when no
// offer applies, the reasons the matching ones were rejected.
func (l *line) bestOffer(offers []offer) (*offer, []Issue) {
	var best *offer
	var rejected []Issue
	for i := range offers {
		o := &offers[i]
		if !o.matches(l) {
			continue
		}
		switch {
		case o.DiscountedPrice >= o.Price:
			rejected = append(rejected, newIssue(IssueOfferWithoutDiscount,
				"Offer ignored: its price (%s) is not lower than the regular price (%s).",
				formatBRL(o.DiscountedPrice), formatBRL(o.Price)))
		case o.Price != l.Price:
			rejected = append(rejected, newIssue(IssueOfferPriceMismatch,
				"Offer ignored: it was created for a regular price of %s, but the product now costs %s.",
				formatBRL(o.Price), formatBRL(l.Price)))
		case best == nil || o.DiscountedPrice < best.DiscountedPrice:
			best = o
		}
	}
	if best != nil {
		return best, nil
	}
	return nil, rejected
}
