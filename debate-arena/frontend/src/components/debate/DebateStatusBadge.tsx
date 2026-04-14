import type { DebateStatus } from "../../types/debate";

interface DebateStatusBadgeProps {
  status: DebateStatus;
}

const mapStatus = {
  PENDING: {
    label: "Pending",
    className: "border-status-pending/40 text-status-pending",
  },
  ACTIVE: {
    label: "Active",
    className: "border-status-active/40 text-status-active",
  },
  COMPLETED: {
    label: "Completed",
    className: "border-status-completed/50 text-status-completed",
  },
} as const;

export function DebateStatusBadge({ status }: DebateStatusBadgeProps) {
  const entry = mapStatus[status];

  return (
    <span
      className={`inline-flex items-center gap-2 rounded-full border px-3 py-1 font-mono text-xs uppercase tracking-wide ${entry.className}`}
    >
      {status === "ACTIVE" ? (
        <span className="h-2 w-2 rounded-full bg-status-active animate-pulseDot" />
      ) : null}
      {entry.label}
    </span>
  );
}
