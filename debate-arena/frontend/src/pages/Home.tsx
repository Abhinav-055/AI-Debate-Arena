import { useMemo, useState } from "react";
import { useQuery } from "@apollo/client";
import { PlusCircle } from "lucide-react";
import { GET_DEBATES } from "../graphql/queries";
import { CreateDebateModal } from "../components/debate/CreateDebateModal";
import { DebateCard } from "../components/debate/DebateCard";
import type { Debate } from "../types/debate";

interface DebatesData {
  debates: Debate[];
}

export default function Home() {
  const [open, setOpen] = useState(false);
  const { data, loading } = useQuery<DebatesData>(GET_DEBATES, {
    variables: { limit: 10, offset: 0 },
    pollInterval: 10000,
  });

  const debates = useMemo(() => data?.debates ?? [], [data?.debates]);

  return (
    <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
      <section className="rounded-xl border border-border bg-card p-6">
        <p className="font-mono text-xs uppercase tracking-wide text-text-muted">
          AI Debate Arena
        </p>
        <h1 className="mt-2 font-display text-4xl leading-tight text-text-primary">
          Start a Debate
        </h1>
        <p className="mt-4 max-w-lg text-sm leading-7 text-text-muted">
          Spin up a real-time debate between specialized AI personalities and watch
          arguments unfold live with judged outcomes.
        </p>
        <button
          type="button"
          onClick={() => setOpen(true)}
          className="mt-6 inline-flex items-center gap-2 rounded-xl border border-border bg-elevated px-4 py-3 text-sm text-text-primary transition-all duration-200 hover:border-text-subtle"
        >
          <PlusCircle className="h-4 w-4" />
          New Debate
        </button>
      </section>

      <section className="space-y-3">
        <h2 className="font-display text-2xl text-text-primary">Recent Debates</h2>
        {loading ? (
          <div className="space-y-3">
            {Array.from({ length: 3 }).map((_, idx) => (
              <div
                key={idx}
                className="h-28 rounded-xl border border-border bg-[linear-gradient(110deg,#1C1C1F,45%,#25252A,55%,#1C1C1F)] bg-[length:200%_100%] animate-shimmer"
              />
            ))}
          </div>
        ) : debates.length === 0 ? (
          <div className="rounded-xl border border-dashed border-border bg-card p-8 text-center text-text-muted">
            <p className="font-display text-xl text-text-primary">No debates yet</p>
            <p className="mt-2 text-sm">Create your first debate to start the arena.</p>
          </div>
        ) : (
          <div className="space-y-3">
            {debates.map((debate) => (
              <DebateCard key={debate.id} debate={debate} />
            ))}
          </div>
        )}
      </section>

      <CreateDebateModal open={open} onOpenChange={setOpen} />
    </div>
  );
}
