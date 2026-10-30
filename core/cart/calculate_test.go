package cart

import (
	"errors"
	"reflect"
	"testing"
)

const (
	eanCoffee = "7891000100103"
	upcA      = "036000291452"  // UPC-A
	upcAAs13  = "0036000291452" // the same product written as EAN-13
	otherEAN  = "4006381333931"
	unlimited = 1000
)

func calc(t *testing.T, c Cart) Result {
	t.Helper()
	res, err := Calculate(c)
	if err != nil {
		t.Fatalf("Calculate: %v", err)
	}
	return res
}

func codes(issues []Issue) []string {
	out := []string{}
	for _, is := range issues {
		out = append(out, is.Code)
	}
	return out
}

func TestTotalsAreExactCents(t *testing.T) {
	res := calc(t, Cart{Items: []Item{
		{SKU: "BAG", Price: 10, Quantity: 3, Stock: unlimited},  // 0.1 * 3 = 0.30000000000000004 in float64
		{SKU: "PEN", Price: 110, Quantity: 3, Stock: unlimited}, // 1.1 * 3 = 3.3000000000000003 in float64
		{SKU: "CANDY", Price: 20, Quantity: 1, Stock: unlimited},
	}})
	want := Summary{ItemCount: 3, UnitCount: 7, SubtotalCents: 380, DiscountCents: 0, TotalCents: 380}
	if res.Summary != want {
		t.Errorf("summary = %+v; want %+v", res.Summary, want)
	}
	if got := res.Items[0].TotalCents; got != 30 {
		t.Errorf("R$ 0.10 × 3 = %d cents; want 30", got)
	}
	if !res.Valid || len(res.Issues) != 0 {
		t.Errorf("valid = %v, issues = %v; want a valid cart without issues", res.Valid, res.Issues)
	}
}

func TestStock(t *testing.T) {
	res := calc(t, Cart{Items: []Item{
		{SKU: "OK", Price: 1000, Quantity: 2, Stock: 2},
		{SKU: "ADJ", Price: 1000, Quantity: 5, Stock: 3},
		{SKU: "OUT", Price: 1000, Quantity: 1, Stock: 0},
	}})
	tests := []struct {
		status    string
		quantity  int64
		total     Cents
		issueCode []string
	}{
		{StatusOK, 2, 2000, []string{}},
		{StatusAdjusted, 3, 3000, []string{IssueQuantityAdjusted}},
		{StatusUnavailable, 0, 0, []string{IssueOutOfStock}},
	}
	for i, tt := range tests {
		it := res.Items[i]
		if it.Status != tt.status || it.Quantity != tt.quantity || it.TotalCents != tt.total || !reflect.DeepEqual(codes(it.Issues), tt.issueCode) {
			t.Errorf("item %s: status=%s quantity=%d total=%d issues=%v; want %s %d %d %v",
				it.SKU, it.Status, it.Quantity, it.TotalCents, codes(it.Issues), tt.status, tt.quantity, tt.total, tt.issueCode)
		}
	}
	if res.Items[1].RequestedQuantity != 5 {
		t.Errorf("requestedQuantity = %d; want 5", res.Items[1].RequestedQuantity)
	}
	if want := (Summary{ItemCount: 2, UnitCount: 5, SubtotalCents: 5000, TotalCents: 5000}); res.Summary != want {
		t.Errorf("summary = %+v; want %+v", res.Summary, want)
	}
	if res.Valid {
		t.Error("a cart with adjusted items must not be valid")
	}
}

func TestDuplicateSKUsAreMergedBeforeCheckingStock(t *testing.T) {
	// Each line fits in the stock on its own; together they do not.
	res := calc(t, Cart{Items: []Item{
		{SKU: "A", Name: "first", Price: 500, Quantity: 3, Stock: 5},
		{SKU: "B", Price: 100, Quantity: 1, Stock: 5},
		{SKU: "A", Name: "second", Price: 500, Quantity: 3, Stock: 5},
	}})
	if len(res.Items) != 2 {
		t.Fatalf("got %d items; want the two A lines merged", len(res.Items))
	}
	a := res.Items[0]
	if a.Name != "first" || a.RequestedQuantity != 6 || a.Quantity != 5 || a.Status != StatusAdjusted {
		t.Errorf("merged item = %+v", a)
	}
	if want := []string{IssueDuplicateSKUMerged, IssueQuantityAdjusted}; !reflect.DeepEqual(codes(a.Issues), want) {
		t.Errorf("issues = %v; want %v", codes(a.Issues), want)
	}
}

func TestDuplicateSKUsMustAgree(t *testing.T) {
	base := Item{SKU: "A", EAN: eanCoffee, Price: 500, Quantity: 1, Stock: 5}
	tests := map[string]Item{
		"cartItems[1].price": {SKU: "A", EAN: eanCoffee, Price: 400, Quantity: 1, Stock: 5},
		"cartItems[1].stock": {SKU: "A", EAN: eanCoffee, Price: 500, Quantity: 1, Stock: 4},
		"cartItems[1].ean":   {SKU: "A", Price: 500, Quantity: 1, Stock: 5},
	}
	for path, dup := range tests {
		_, err := Calculate(Cart{Items: []Item{base, dup}})
		assertInvalid(t, err, path)
	}
}

func TestOffers(t *testing.T) {
	item := Item{SKU: "COFFEE", EAN: eanCoffee, Price: 1890, Quantity: 2, Stock: unlimited}
	upc := Item{SKU: "UPC", EAN: upcA, Price: 1000, Quantity: 1, Stock: unlimited}
	tests := []struct {
		name      string
		item      Item
		offers    []Offer
		final     Cents
		issueCode []string
	}{
		{"matched by sku", item, []Offer{{SKU: "COFFEE", Price: 1890, DiscountedPrice: 1590}}, 1590, []string{}},
		{"matched by ean", item, []Offer{{EAN: eanCoffee, Price: 1890, DiscountedPrice: 1590}}, 1590, []string{}},
		{"matched by sku and ean", item, []Offer{{SKU: "COFFEE", EAN: eanCoffee, Price: 1890, DiscountedPrice: 1590}}, 1590, []string{}},
		{"ean written as GTIN-14", item, []Offer{{EAN: "0" + eanCoffee, Price: 1890, DiscountedPrice: 1590}}, 1590, []string{}},
		{"UPC-A item, EAN-13 offer", upc, []Offer{{EAN: upcAAs13, Price: 1000, DiscountedPrice: 900}}, 900, []string{}},
		{"other sku", item, []Offer{{SKU: "TEA", Price: 1890, DiscountedPrice: 1590}}, 1890, []string{}},
		{"sku matches, ean does not", item, []Offer{{SKU: "COFFEE", EAN: otherEAN, Price: 1890, DiscountedPrice: 1590}}, 1890, []string{}},
		{"stale price", item, []Offer{{SKU: "COFFEE", Price: 1990, DiscountedPrice: 1590}}, 1890, []string{IssueOfferPriceMismatch}},
		{"no discount", item, []Offer{{SKU: "COFFEE", Price: 1890, DiscountedPrice: 1890}}, 1890, []string{IssueOfferWithoutDiscount}},
		{"price increase", item, []Offer{{SKU: "COFFEE", Price: 1890, DiscountedPrice: 1990}}, 1890, []string{IssueOfferWithoutDiscount}},
		{"lowest price wins", item, []Offer{
			{SKU: "COFFEE", Price: 1890, DiscountedPrice: 1690},
			{EAN: eanCoffee, Price: 1890, DiscountedPrice: 1490},
			{SKU: "COFFEE", Price: 1890, DiscountedPrice: 1590},
		}, 1490, []string{}},
		{"rejections are not reported when an offer applies", item, []Offer{
			{SKU: "COFFEE", Price: 1990, DiscountedPrice: 1290},
			{SKU: "COFFEE", Price: 1890, DiscountedPrice: 1590},
		}, 1590, []string{}},
		{"every rejection is reported", item, []Offer{
			{SKU: "COFFEE", Price: 1990, DiscountedPrice: 1290},
			{SKU: "COFFEE", Price: 1890, DiscountedPrice: 1890},
		}, 1890, []string{IssueOfferPriceMismatch, IssueOfferWithoutDiscount}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := calc(t, Cart{Items: []Item{tt.item}, Offers: tt.offers})
			it := res.Items[0]
			if it.FinalUnitPriceCents != tt.final || it.OfferApplied != (tt.final < tt.item.Price) {
				t.Errorf("finalUnitPrice = %d, offerApplied = %v; want %d", it.FinalUnitPriceCents, it.OfferApplied, tt.final)
			}
			if !reflect.DeepEqual(codes(it.Issues), tt.issueCode) {
				t.Errorf("issues = %v; want %v", codes(it.Issues), tt.issueCode)
			}
			subtotal, total := it.UnitPriceCents*it.Quantity, it.FinalUnitPriceCents*it.Quantity
			if it.SubtotalCents != subtotal || it.TotalCents != total || it.DiscountCents != subtotal-total {
				t.Errorf("amounts = %d - %d = %d; want %d - %d = %d",
					it.SubtotalCents, it.DiscountCents, it.TotalCents, subtotal, subtotal-total, total)
			}
		})
	}
}

func TestOfferMessage(t *testing.T) {
	res := calc(t, Cart{
		Items:  []Item{{SKU: "DETERGENT", Price: 2190, Quantity: 1, Stock: 1}},
		Offers: []Offer{{SKU: "DETERGENT", Price: 2390, DiscountedPrice: 1990}},
	})
	want := "Offer ignored: it was created for a regular price of R$ 23.90, but the product now costs R$ 21.90."
	if got := res.Items[0].Issues[0].Message; got != want {
		t.Errorf("message = %q; want %q", got, want)
	}
}

func TestEmptyCart(t *testing.T) {
	for name, items := range map[string][]Item{
		"no items":          nil,
		"nothing available": {{SKU: "A", Price: 100, Quantity: 1, Stock: 0}},
	} {
		res := calc(t, Cart{Items: items})
		if res.Valid || !reflect.DeepEqual(codes(res.Issues), []string{IssueEmptyCart}) {
			t.Errorf("%s: valid = %v, issues = %v; want invalid with %s", name, res.Valid, codes(res.Issues), IssueEmptyCart)
		}
		if res.Items == nil {
			t.Errorf("%s: items is nil; want an empty list", name)
		}
	}
}

func TestValidation(t *testing.T) {
	item := func(change func(*Item)) Cart {
		it := Item{SKU: "A", EAN: eanCoffee, Price: 100, Quantity: 1, Stock: 1}
		change(&it)
		return Cart{Items: []Item{it}}
	}
	offer := func(o Offer) Cart {
		return Cart{Items: []Item{{SKU: "A", Price: 100, Quantity: 1, Stock: 1}}, Offers: []Offer{o}}
	}
	tests := map[string]Cart{
		"cartItems[0].sku":          item(func(it *Item) { it.SKU = " " }),
		"cartItems[0].ean":          item(func(it *Item) { it.EAN = "7891000100104" }),
		"cartItems[0].price":        item(func(it *Item) { it.Price = -1 }),
		"cartItems[0].quantity":     item(func(it *Item) { it.Quantity = 0 }),
		"cartItems[0].stock":        item(func(it *Item) { it.Stock = -1 }),
		"cartItems[0]":              item(func(it *Item) { it.Price, it.Quantity, it.Stock = MaxSafeInteger, 2, 2 }),
		"offers[0]":                 offer(Offer{Price: 100, DiscountedPrice: 90}),
		"offers[0].sku":             offer(Offer{SKU: "  ", Price: 100, DiscountedPrice: 90}),
		"offers[0].ean":             offer(Offer{EAN: "123", Price: 100, DiscountedPrice: 90}),
		"offers[0].price":           offer(Offer{SKU: "A", Price: -100, DiscountedPrice: 90}),
		"offers[0].discountedPrice": offer(Offer{SKU: "A", Price: 100, DiscountedPrice: -90}),
	}
	for path, c := range tests {
		_, err := Calculate(c)
		assertInvalid(t, err, path)
	}
}

func TestCartTotalOverflow(t *testing.T) {
	half := Cents(MaxSafeInteger/2 + 1)
	_, err := Calculate(Cart{Items: []Item{
		{SKU: "A", Price: half, Quantity: 1, Stock: 1},
		{SKU: "B", Price: half, Quantity: 1, Stock: 1},
	}})
	assertInvalid(t, err, "cartItems")
}

func assertInvalid(t *testing.T, err error, path string) {
	t.Helper()
	var e *Error
	if !errors.As(err, &e) || e.Code != ErrInvalidInput || e.Path != path {
		t.Errorf("error = %#v; want %s at %q", err, ErrInvalidInput, path)
	}
}
