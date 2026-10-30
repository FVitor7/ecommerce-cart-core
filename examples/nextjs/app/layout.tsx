import type { Metadata } from 'next';

import './globals.css';

export const metadata: Metadata = {
  title: 'Cart · ecommerce-cart-core',
  description: 'The same calc.wasm calculating the cart on the server and in the browser.',
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en">
      <body>{children}</body>
    </html>
  );
}
