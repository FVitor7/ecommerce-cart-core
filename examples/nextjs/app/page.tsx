import { calculate } from 'ecommerce-cart-core';

import cart from '../../../fixtures/cart.json';
import offers from '../../../fixtures/offers.json';
import { CartClient } from './cart-client';
import { CartTable } from './cart-table';

// A real cart belongs to a user, so it is calculated on every request.
export const dynamic = 'force-dynamic';

// Server Component: the cart is calculated on the server, while rendering.
export default async function Page() {
  const input = { ...cart, ...offers };
  const result = await calculate(input);

  return (
    <main>
      <h1>ecommerce-cart-core in Next.js</h1>
      <p>
        The same <code>calc.wasm</code> calculates this cart on the server, in the browser and in a Route Handler.
      </p>

      <section>
        <h2>Server Component</h2>
        <p className="hint">Calculated on the server, while rendering the page.</p>
        <CartTable result={result} />
      </section>

      <section>
        <h2>Client Component</h2>
        <p className="hint">
          Recalculated in the browser on every change and revalidated on the server by <code>POST /api/cart</code>.
        </p>
        <CartClient initialCart={input} />
      </section>
    </main>
  );
}
