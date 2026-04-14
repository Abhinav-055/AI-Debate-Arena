import { useState } from "react";
import { useQuery } from "@apollo/client";
import { ChevronDown, ChevronUp } from "lucide-react";
import { format } from "date-fns";
import { GET_DEBATES, GET_JUDGEMENT, GET_MESSAGES } from "../graphql/queries";
import { VerdictCard } from "../components/judgement/VerdictCard";
import type { Debate, Judgement, Message } from "../types/debate";

interface DebatesData {
  debates: Debate[];
}

interface MessagesData {
  messages: Message[];
}

interface JudgementData {
  judgement: Judgement | null;
}

function HistoryRow({ debate }: { debate: Debate }) {
  const [open, setOpen] = useState(false);
  const { data: messagesData } = useQuery<MessagesData>(GET_MESSAGES, {
    variables: { debateId: debate.id },
    skip: !open,
  });
  const { data: judgementData } = useQuery<JudgementData>(GET_JUDGEMENT, {
    variables: { debateId: debate.id },
    fetchPolicy: "cache-and-network",
    pollInterval: 3000,
  });

  const verdict = judgementData?.judgement?.verdict ?? "No verdict available yet.";
  const winner = debate.agents.find((a) => a.id === judgementData?.judgement?.winnerAgentId);
  const winnerColor = winner?.turnOrder === 1 ? "var(--agent-b-color)" : "var(--agent-a-color)";

  return (
    <article className="rounded-xl border border-border bg-card p-4">
      <button
        type="button"
        className="flex w-full items-start justify-between gap-4 text-left"
        onClick={() => setOpen((v) => !v)}
      >
        <div>
          <h3 className="font-display text-lg text-text-primary">{debate.topic}</h3>
          <p className="mt-1 text-xs text-text-muted">
            {format(new Date(debate.createdAt), "PPpp")}
          </p>
          <p className="mt-2 text-sm text-text-muted">{verdict.slice(0, 100)}</p>
        </div>
        {open ? (
          <ChevronUp className="mt-1 h-5 w-5 text-text-muted" />
        ) : (
          <ChevronDown className="mt-1 h-5 w-5 text-text-muted" />
        )}
      </button>

      {open ? (
        <div className="mt-4 space-y-3 border-t border-border pt-4">
          <h4 className="font-display text-base text-text-primary">Message Thread</h4>
          <div className="space-y-2">
            {(messagesData?.messages ?? []).map((m) => (
              <div key={m.id} className="rounded-lg border border-border bg-elevated p-3">
                <p className="text-xs text-text-muted">
                  Round {m.roundNumber} - {debate.agents.find((a) => a.id === m.agentId)?.name ?? "Agent"}
                </p>
                <p className="mt-1 text-sm text-text-primary">{m.content}</p>
              </div>
            ))}
          </div>
          {judgementData?.judgement ? (
            <VerdictCard
              judgement={judgementData.judgement}
              winnerName={winner?.name}
              winnerColor={winnerColor}
            />
          ) : null}
        </div>
      ) : null}
    </article>
  );
}

export default function DebateHistory() {
  const { data, loading } = useQuery<DebatesData>(GET_DEBATES, {
    variables: { status: "COMPLETED", limit: 100, offset: 0 },
    fetchPolicy: "cache-and-network",
    pollInterval: 3000,
  });

  const debates = data?.debates ?? [];

  if (loading) {
    return (
      <div className="space-y-3">
        {Array.from({ length: 4 }).map((_, idx) => (
          <div
            key={idx}
            className="h-28 rounded-xl border border-border bg-[linear-gradient(110deg,#1C1C1F,45%,#25252A,55%,#1C1C1F)] bg-[length:200%_100%] animate-shimmer"
          />
        ))}
      </div>
    );
  }

  if (debates.length === 0) {
    return (
      <section className="rounded-xl border border-dashed border-border bg-card p-10 text-center">
        <h2 className="font-display text-2xl text-text-primary">No Debate History Yet</h2>
        <p className="mt-2 text-sm text-text-muted">
          Completed debates will appear here with winner and verdict details.
        </p>
      </section>
    );
  }

  return (
    <section className="space-y-3">
      <h1 className="font-display text-3xl text-text-primary">Debate History</h1>
      {debates.map((debate) => (
        <HistoryRow key={debate.id} debate={debate} />
      ))}
    </section>
  );
}
