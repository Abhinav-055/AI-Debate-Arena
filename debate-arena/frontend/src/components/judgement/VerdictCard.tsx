import type { Judgement } from "../../types/debate";

interface VerdictCardProps {
  judgement: Judgement;
  winnerName?: string;
  winnerColor: string;
}

export function VerdictCard({ judgement, winnerName, winnerColor }: VerdictCardProps) {
  return (
    <article className="rounded-xl border border-border bg-elevated p-4">
      <h4 className="mb-2 font-display text-lg text-text-primary">Verdict</h4>
      <p
        className="mb-3 inline-flex rounded-full border px-3 py-1 text-sm font-semibold"
        style={{
          color: winnerColor,
          borderColor: "color-mix(in srgb, " + winnerColor + " 50%, transparent)",
          backgroundColor: "color-mix(in srgb, " + winnerColor + " 15%, transparent)",
        }}
      >
        Winner: {winnerName ?? "TBD"}
      </p>
      <blockquote className="border-l-2 border-status-active pl-3 text-sm leading-7 text-text-primary">
        {judgement.verdict}
      </blockquote>
    </article>
  );
}
