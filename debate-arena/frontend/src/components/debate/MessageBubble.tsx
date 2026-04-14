import { formatDistanceToNow } from "date-fns";
import type { Message } from "../../types/debate";
import { AgentTag } from "./AgentTag";

interface MessageBubbleProps {
  message: Message;
}

function colorForTurn(turnOrder?: number) {
  return turnOrder === 1 ? "var(--agent-b-color)" : "var(--agent-a-color)";
}

export function MessageBubble({ message }: MessageBubbleProps) {
  const color = colorForTurn(message.agent?.turnOrder);

  return (
    <article
      className="animate-slideIn rounded-xl border border-border bg-card p-4"
      style={{
        borderLeftWidth: "4px",
        borderLeftColor: color,
        backgroundColor: "color-mix(in srgb, " + color + " 6%, #1C1C1F)",
      }}
    >
      <header className="mb-2 flex items-center justify-between gap-3">
        <AgentTag
          name={message.agent?.name ?? "Unknown Agent"}
          provider={message.agent?.provider ?? "GROQ"}
          turnOrder={message.agent?.turnOrder}
          compact
        />
        <div className="flex items-center gap-2 font-mono text-xs text-text-muted">
          <span>{formatDistanceToNow(new Date(message.createdAt))} ago</span>
        </div>
      </header>
      <p className="whitespace-pre-wrap text-sm leading-7 text-text-primary">
        {message.content}
      </p>
    </article>
  );
}
