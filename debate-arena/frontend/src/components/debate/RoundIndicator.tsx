interface RoundIndicatorProps {
  roundsTotal: number;
  roundsDone: number;
}

export function RoundIndicator({ roundsTotal, roundsDone }: RoundIndicatorProps) {
  return (
    <div className="flex items-center gap-3">
      {Array.from({ length: roundsTotal }).map((_, idx) => {
        const roundIndex = idx + 1;
        const done = roundIndex <= roundsDone;
        const active = roundIndex === roundsDone + 1 && roundIndex <= roundsTotal;

        return (
          <div key={roundIndex} className="flex items-center gap-3">
            <div
              className={`h-3 w-3 rounded-full border transition-all duration-200 ${
                done
                  ? "border-text-primary bg-text-primary"
                  : active
                    ? "border-status-active bg-status-active animate-pulseDot"
                    : "border-border bg-transparent"
              }`}
              title={`Round ${roundIndex}`}
            />
            {roundIndex < roundsTotal ? (
              <div className="h-px w-7 bg-border" />
            ) : null}
          </div>
        );
      })}
    </div>
  );
}
