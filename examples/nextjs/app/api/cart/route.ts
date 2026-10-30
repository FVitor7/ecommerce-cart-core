import { CartError, calculate } from 'ecommerce-cart-core';

// Route Handler: the checkout revalidates the cart on the server.
// POST /api/cart with { cartItems, offers } -> CartResult
export async function POST(request: Request) {
  try {
    // The raw body goes straight to the engine, so prices are read from the
    // exact text the client sent.
    return Response.json(await calculate(await request.text()));
  } catch (err) {
    if (err instanceof CartError) {
      const status = err.code === 'ENGINE_ERROR' ? 500 : 400;
      return Response.json({ error: { code: err.code, message: err.message, path: err.path } }, { status });
    }
    throw err;
  }
}
