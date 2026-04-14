import type { AgentProvider } from "../../types/debate";

interface AgentTagProps {
  name: string;
  provider: AgentProvider;
  turnOrder?: number;
  compact?: boolean;
}

function colorForTurn(turnOrder?: number) {
  if (turnOrder === 1) {
    return {
      text: "var(--agent-b-color)",
      bg: "color-mix(in srgb, var(--agent-b-color) 22%, transparent)",
    };
  }
  return {
    text: "var(--agent-a-color)",
    bg: "color-mix(in srgb, var(--agent-a-color) 22%, transparent)",
  };
}

export function AgentTag({ name, provider: _provider, turnOrder, compact = false }: AgentTagProps) {
  const style = colorForTurn(turnOrder);
  const initials = name
    .split(" ")
    .map((part) => part[0])
    .join("")
    .slice(0, 2)
    .toUpperCase();

  return (
    <div className={`inline-flex items-center gap-2 ${compact ? "" : "px-1"}`}>
      <div
        className="flex h-9 w-9 items-center justify-center rounded-full font-mono text-xs"
        style={{ color: style.text, backgroundColor: style.bg }}
      >
        {initials}
      </div>
      <div className="flex flex-col leading-tight">
        <span className="text-sm font-semibold" style={{ color: style.text }}>
          {name}
        </span>
      </div>
    </div>
  );
}
