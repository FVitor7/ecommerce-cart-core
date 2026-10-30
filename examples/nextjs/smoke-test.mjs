// Smoke test of the production build, run inside its container (make test-nextjs):
// starts the server, then checks the Server Component page and the Route
// Handler against the shared fixtures.
import assert from 'node:assert/strict';
import { spawn } from 'node:child_process';
import { readFileSync } from 'node:fs';
import { setTimeout as sleep } from 'node:timers/promises';

const fixtures = new URL('../../fixtures/', import.meta.url);
const read = (name) => JSON.parse(readFileSync(new URL(name, fixtures), 'utf8'));
const base = 'http://localhost:3000';
const post = (body) => fetch(`${base}/api/cart`, { method: 'POST', body });

const server = spawn(process.execPath, ['node_modules/next/dist/bin/next', 'start', '-p', '3000'], {
  stdio: ['ignore', 'ignore', 'inherit'],
});
try {
  for (let i = 0; ; i++) {
    try {
      await fetch(base);
      break;
    } catch (err) {
      if (i === 60) throw err;
      await sleep(500);
    }
  }

  const page = await (await fetch(base)).text();
  assert.match(page, /R\$167\.60/, 'the Server Component renders the total');

  const ok = await post(JSON.stringify({ ...read('cart.json'), ...read('offers.json') }));
  assert.equal(ok.status, 200);
  assert.deepEqual(await ok.json(), read('expected.json'));

  const invalid = await post('{"cartItems": [{"sku": "A", "price": "49,90", "quantity": 1, "stock": 1}]}');
  assert.equal(invalid.status, 400);
  assert.equal((await invalid.json()).error.path, 'cartItems[0].price');

  console.log('ok: the Server Component renders R$167.60; POST /api/cart returns fixtures/expected.json, and 400 for bad input');
} finally {
  server.kill();
}
