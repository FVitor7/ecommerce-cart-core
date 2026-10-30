import path from 'node:path';
import type { NextConfig } from 'next';

const nextConfig: NextConfig = {
  // ecommerce-cart-core is linked from ../../packages/js and the page reads
  // ../../fixtures, so Turbopack has to see the whole repository.
  turbopack: { root: path.join(__dirname, '../..') },
};

export default nextConfig;
