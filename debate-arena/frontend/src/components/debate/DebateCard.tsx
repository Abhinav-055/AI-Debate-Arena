import { format } from "date-fns";
import { Link } from "react-router-dom";
import type { Debate } from "../../types/debate";
import { AgentTag } from "./AgentTag";
import { DebateStatusBadge } from "./DebateStatusBadge";

interface DebateCardProps {
  debate: Debate;
}

export function DebateCard({ debate }: DebateCardProps) {
  const progress = Math.min(
    100,
    Math.round((debate.roundsDone / Math.max(debate.roundsTotal, 1)) * 100),
  );

  return (
    <Link
      to={`/debate/${debate.id}`}
      className="block rounded-xl border border-border bg-card p-4 transition-all duration-200 hover:border-text-subtle"
    >
      <div className="mb-3 flex items-start justify-between gap-3">
        <div>
          <h3 className="font-display text-lg text-text-primary">{debate.topic}</h3>
          <p className="mt-1 text-xs text-text-muted">
            Created {format(new Date(debate.createdAt), "PPpp")}
          </p>
        </div>
        <DebateStatusBadge status={debate.status} />
      </div>

      <div className="mb-3 h-2 overflow-hidden rounded bg-elevated">
        <div
          className="h-full bg-text-primary transition-all duration-200"
          style={{ width: `${progress}%` }}
        />
      </div>

      <div className="grid grid-cols-1 gap-2 md:grid-cols-3">
        {debate.agents.map((agent) => (
          <AgentTag key={agent.id} name={agent.name} provider={agent.provider} turnOrder={agent.turnOrder} compact />
        ))}
      </div>
    </Link>
  );
}
