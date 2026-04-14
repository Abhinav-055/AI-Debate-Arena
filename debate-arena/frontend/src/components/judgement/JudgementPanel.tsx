import { Trophy } from "lucide-react";
import type { Agent, Judgement } from "../../types/debate";
import { VerdictCard } from "./VerdictCard";

interface JudgementPanelProps {
  judgement: Judgement;
  agents: Agent[];
  winnerAgentId?: string | null;
  onWatchReplay: () => void;
}

function colorForTurn(turnOrder?: number) {
  return turnOrder === 1 ? "var(--agent-b-color)" : "var(--agent-a-color)";
}

export function JudgementPanel({ judgement, agents, winnerAgentId, onWatchReplay }: JudgementPanelProps) {
  const agentNameByID = new Map(agents.map((agent) => [agent.id, agent.name]));
  const resolvedWinnerId = winnerAgentId ?? judgement.winnerAgentId ?? null;
  const winnerAgent = resolvedWinnerId ? agents.find((agent) => agent.id === resolvedWinnerId) : undefined;
  const winnerName = judgement.winner?.name ?? (resolvedWinnerId ? agentNameByID.get(resolvedWinnerId) : undefined);
  const winnerColor = colorForTurn(winnerAgent?.turnOrder);

  return (
    <section className="space-y-4 rounded-xl border border-border bg-card p-4">
      <header className="flex items-center justify-between">
        <div>
          <p className="text-xs uppercase tracking-wide text-text-muted">Winner</p>
          <div
            className="mt-1 inline-flex items-center gap-2 rounded-full border px-3 py-1 text-sm font-semibold"
            style={{
              color: winnerColor,
              borderColor: "color-mix(in srgb, " + winnerColor + " 55%, transparent)",
              backgroundColor: "color-mix(in srgb, " + winnerColor + " 18%, transparent)",
            }}
          >
            <Trophy className="h-4 w-4" />
            {winnerName ?? "TBD"}
          </div>
        </div>
        <button
          type="button"
          onClick={onWatchReplay}
          className="rounded-xl border border-border px-3 py-2 text-sm text-text-primary transition-all duration-200 hover:border-text-subtle"
        >
          Watch Replay
        </button>
      </header>

      <div className="grid grid-cols-1 gap-3 lg:grid-cols-2">
        <div className="rounded-xl border border-green-500/20 bg-green-500/5 p-3">
          <h5 className="mb-2 text-sm font-semibold text-green-400">Strengths</h5>
          <ul className="space-y-2 text-sm text-text-primary">
            {judgement.strengths.map((item) => (
              <li key={`${item.agentId}-${item.reason}`}>
                <span className="font-mono text-xs text-green-300">{agentNameByID.get(item.agentId) ?? "Unknown Agent"}</span>
                <p>{item.reason}</p>
              </li>
            ))}
          </ul>
        </div>
        <div className="rounded-xl border border-red-500/20 bg-red-500/5 p-3">
          <h5 className="mb-2 text-sm font-semibold text-red-400">Weaknesses</h5>
          <ul className="space-y-2 text-sm text-text-primary">
            {judgement.weaknesses.map((item) => (
              <li key={`${item.agentId}-${item.reason}`}>
                <span className="font-mono text-xs text-red-300">{agentNameByID.get(item.agentId) ?? "Unknown Agent"}</span>
                <p>{item.reason}</p>
              </li>
            ))}
          </ul>
        </div>
      </div>

      <VerdictCard judgement={judgement} winnerName={winnerName} winnerColor={winnerColor} />
    </section>
  );
}
