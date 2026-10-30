'use client';

import { type CartInput, type CartResult, calculate } from 'ecommerce-cart-core';
import { useEffect, useState } from 'react';

import { CartTable, brl } from './cart-table';

// Client Component: the browser runs calc.wasm itself, so the cart updates
// instantly; the Route Handler revalidates the same cart on the server.
export function CartClient({ initialCart }: { initialCart: CartInput }) {
  const [cart, setCart] = useState(initialCart);
  const [browser, setBrowser] = useState<CartResult>();
  const [server, setServer] = useState<CartResult>();
  const [error, setError] = useState<string>();

  useEffect(() => {
    let current = true;
    calculate(cart).then(
      (result) => current && setBrowser(result),
      (err: Error) => current && setError(err.message),
    );
    fetch('/api/cart', { method: 'POST', body: JSON.stringify(cart) })
      .then((response) => response.json())
      .then(
        (result: CartResult) => current && setServer(result),
        (err: Error) => current && setError(err.message),
      );
    return () => {
      current = false;
    };
  }, [cart]);

  // Sets the requested quantity of a product, keeping the original order of
  // the lines; lines with the same SKU become one.
  function changeQuantity(sku: string, quantity: number) {
    const position = (line: { sku: string }) => initialCart.cartItems.findIndex((l) => l.sku === line.sku);
    const line = cart.cartItems.find((l) => l.sku === sku)!;
    const others = cart.cartItems.filter((l) => l.sku !== sku);
    const cartItems = quantity < 1 ? others : [...others, { ...line, quantity }];
    setCart({ ...cart, cartItems: cartItems.sort((a, b) => position(a) - position(b)) });
  }

  if (error) return <p className="error">{error}</p>;
  if (!browser) return <p>Loading the calculation engine…</p>;

  const same = server && JSON.stringify(server) === JSON.stringify(browser);
  return (
    <>
      <CartTable result={browser} onQuantityChange={changeQuantity} />
      <dl className="compare">
        <dt>Total in the browser</dt>
        <dd id="browser-total">{brl(browser.summary.totalCents)}</dd>
        <dt>Total on the server (Route Handler)</dt>
        <dd id="server-total">{server ? brl(server.summary.totalCents) : '…'}</dd>
      </dl>
      {server && (
        <p id="match" className={same ? 'match' : 'error'}>
          {same ? '✓ The browser and the server got the same result.' : '✗ The browser and the server disagree.'}
        </p>
      )}
    </>
  );
}
