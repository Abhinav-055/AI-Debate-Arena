import type { PropsWithChildren } from "react";
import { Navbar } from "./Navbar";

export function PageShell({ children }: PropsWithChildren) {
  return (
    <div className="min-h-screen bg-bg text-text-primary">
      <Navbar />
      <main className="mx-auto w-full max-w-7xl px-4 py-6">{children}</main>
    </div>
  );
}
