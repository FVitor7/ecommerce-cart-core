/**
 * Money input: reais as a JSON number (`49.9`) or a decimal string (`"49.90"`),
 * with at most 2 decimal places. It is read from its text, never as a float.
 */
export type MoneyInput = number | string;

export interface CartItemInput {
  /** Identifies the line. Lines with the same SKU are merged. */
  sku: string;
  /** EAN-8, UPC-A, EAN-13 or GTIN-14, validated by its check digit. */
  ean?: string | null;
  name?: string | null;
  /** Regular unit price. */
  price: MoneyInput;
  /** Requested quantity, at least 1. */
  quantity: number;
  /** Units available. */
  stock: number;
  /** Other fields are ignored. */
  [field: string]: unknown;
}

/**
 * A special price: the product was `price`, it is now `discountedPrice`. Every
 * identifier the offer provides (sku, ean) must match the item, and the offer
 * applies only while `price` is still the item's current price.
 */
export interface OfferInput {
  sku?: string | null;
  ean?: string | null;
  price: MoneyInput;
  discountedPrice: MoneyInput;
  [field: string]: unknown;
}

export interface CartInput {
  cartItems: CartItemInput[];
  offers?: OfferInput[] | null;
  [field: string]: unknown;
}

export type ItemStatus = 'ok' | 'adjusted' | 'unavailable';

export type IssueCode =
  | 'DUPLICATE_SKU_MERGED'
  | 'QUANTITY_ADJUSTED'
  | 'OUT_OF_STOCK'
  | 'OFFER_PRICE_MISMATCH'
  | 'OFFER_WITHOUT_DISCOUNT'
  | 'EMPTY_CART';

export interface Issue {
  code: IssueCode;
  /** Ready to show to the customer. */
  message: string;
}

/** A revalidated line. All amounts are integer cents. */
export interface CartItemResult {
  sku: string;
  /** As sent, or "" when absent. */
  ean: string;
  name: string;
  status: ItemStatus;
  requestedQuantity: number;
  /** What will be charged: min(requestedQuantity, stock). */
  quantity: number;
  unitPriceCents: number;
  /** After the offer, if any. */
  finalUnitPriceCents: number;
  /** unitPriceCents × quantity */
  subtotalCents: number;
  discountCents: number;
  /** finalUnitPriceCents × quantity */
  totalCents: number;
  offerApplied: boolean;
  issues: Issue[];
}

/** All amounts are integer cents; totalCents = subtotalCents − discountCents. */
export interface CartSummary {
  /** Distinct products being bought. */
  itemCount: number;
  /** Units being bought. */
  unitCount: number;
  subtotalCents: number;
  discountCents: number;
  totalCents: number;
}

export interface CartResult {
  /** The cart can be checked out exactly as submitted: everything is available. */
  valid: boolean;
  /** Cart-level issues. Item issues are in each item. */
  issues: Issue[];
  summary: CartSummary;
  /** One entry per distinct SKU, in order of first appearance. */
  items: CartItemResult[];
}

export type CartErrorCode = 'INVALID_JSON' | 'INVALID_INPUT' | 'ENGINE_ERROR' | 'INTERNAL_ERROR';

/** Thrown when the cart cannot be calculated. */
export declare class CartError extends Error {
  readonly name: 'CartError';
  readonly code: CartErrorCode;
  /** The offending field, e.g. "cartItems[2].price". */
  readonly path?: string;
}

/**
 * Revalidates the cart against stock and offers and computes its totals in
 * integer cents. Accepts the cart object or its JSON text.
 *
 * @throws {CartError}
 */
export declare function calculate(cart: CartInput | string): Promise<CartResult>;
