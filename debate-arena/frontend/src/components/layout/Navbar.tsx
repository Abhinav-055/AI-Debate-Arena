import { Bot, History, House } from "lucide-react";
import { Link, NavLink } from "react-router-dom";

export function Navbar() {
  const goHistoryWithReload = () => {
    window.location.assign("/history");
  };

  return (
    <header className="sticky top-0 z-40 border-b border-border bg-bg/90 backdrop-blur">
      <div className="mx-auto flex h-16 max-w-7xl items-center justify-between px-4">
        <Link to="/" className="flex items-center gap-2">
          <Bot className="h-6 w-6 text-status-active" />
          <span className="font-display text-xl text-text-primary">AI Debate Arena</span>
        </Link>
        <nav className="flex items-center gap-2">
          <NavLink
            to="/"
            className={({ isActive }) =>
              `inline-flex items-center gap-2 rounded-xl border px-3 py-2 text-sm transition-all duration-200 ${
                isActive
                  ? "border-text-subtle text-text-primary"
                  : "border-border text-text-muted hover:border-text-subtle"
              }`
            }
          >
            <House className="h-4 w-4" />
            Home
          </NavLink>
          <button
            type="button"
            onClick={goHistoryWithReload}
            className="inline-flex items-center gap-2 rounded-xl border border-border px-3 py-2 text-sm text-text-muted transition-all duration-200 hover:border-text-subtle"
          >
            <History className="h-4 w-4" />
            History
          </button>
        </nav>
      </div>
    </header>
  );
}
