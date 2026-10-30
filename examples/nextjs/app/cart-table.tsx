import type { CartResult } from 'ecommerce-cart-core';

// The engine does all the math; formatting cents for display is all that is left.
const currency = new Intl.NumberFormat('en-US', { style: 'currency', currency: 'BRL' });
export const brl = (cents: number) => currency.format(cents / 100);

// Used by both the Server Component and the Client Component.
export function CartTable({
  result,
  onQuantityChange,
}: {
  result: CartResult;
  onQuantityChange?: (sku: string, quantity: number) => void;
}) {
  const { summary } = result;
  return (
    <table>
      <thead>
        <tr>
          <th>Product</th>
          <th>Qty</th>
          <th className="num">Unit price</th>
          <th className="num">Total</th>
        </tr>
      </thead>
      <tbody>
        {result.items.map((item) => (
          <tr key={item.sku}>
            <td>
              {item.name}
              <span className={`status ${item.status}`}>{item.status}</span>
              {item.issues.map((issue, i) => (
                <small key={i} className="issue">
                  {issue.message}
                </small>
              ))}
            </td>
            <td className="qty">
              {onQuantityChange && (
                <button aria-label="Decrease" onClick={() => onQuantityChange(item.sku, item.requestedQuantity - 1)}>
                  −
                </button>
              )}
              {item.quantity === item.requestedQuantity ? item.quantity : `${item.quantity} of ${item.requestedQuantity}`}
              {onQuantityChange && (
                <button aria-label="Increase" onClick={() => onQuantityChange(item.sku, item.requestedQuantity + 1)}>
                  +
                </button>
              )}
            </td>
            <td className="num">
              {item.offerApplied && <s>{brl(item.unitPriceCents)}</s>} {brl(item.finalUnitPriceCents)}
            </td>
            <td className="num">{brl(item.totalCents)}</td>
          </tr>
        ))}
      </tbody>
      <tfoot>
        <tr>
          <td colSpan={3}>Subtotal ({summary.unitCount} units)</td>
          <td className="num">{brl(summary.subtotalCents)}</td>
        </tr>
        <tr>
          <td colSpan={3}>Discounts</td>
          <td className="num">−{brl(summary.discountCents)}</td>
        </tr>
        <tr className="total">
          <td colSpan={3}>Total</td>
          <td className="num">{brl(summary.totalCents)}</td>
        </tr>
      </tfoot>
    </table>
  );
}
