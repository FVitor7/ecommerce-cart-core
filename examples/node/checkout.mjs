// Simulates a checkout: revalidates fixtures/cart.json with the offers in fixtures/offers.json.
//
// Runs in the container of the npm package (make demo compares it with the Python output):
//
//   docker compose run --rm js node /src/examples/node/checkout.mjs
import { readFileSync } from 'node:fs';

import { calculate } from 'ecommerce-cart-core';

const fixtures = new URL('../../fixtures/', import.meta.url);
const cart = JSON.parse(readFileSync(new URL('cart.json', fixtures), 'utf8'));
const offers = JSON.parse(readFileSync(new URL('offers.json', fixtures), 'utf8'));

const result = await calculate({ ...cart, ...offers });
console.log(JSON.stringify(result, null, 2));
