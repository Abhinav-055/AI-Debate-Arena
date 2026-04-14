import * as Dialog from "@radix-ui/react-dialog";
import { X } from "lucide-react";
import { useMemo, useState } from "react";
import toast from "react-hot-toast";
import { useCreateDebate } from "../../hooks/useCreateDebate";
import type { CreateDebateInput } from "../../types/debate";

interface CreateDebateModalProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

type Slot = "A" | "B";

export function CreateDebateModal({ open, onOpenChange }: CreateDebateModalProps) {
  const [topic, setTopic] = useState("");
  const [rounds, setRounds] = useState(3);
  const [agentAName, setAgentAName] = useState("Agent A");
  const [agentBName, setAgentBName] = useState("Agent B");
  const [slotAPersonality, setSlotAPersonality] = useState("");
  const [slotBPersonality, setSlotBPersonality] = useState("");
  const [forSlot, setForSlot] = useState<Slot>("A");

  const { createAndStart, loading } = useCreateDebate();

  const roleSummary = useMemo(() => {
    const forPersonality = forSlot === "A" ? slotAPersonality : slotBPersonality;
    const againstPersonality = forSlot === "A" ? slotBPersonality : slotAPersonality;
    return { forPersonality, againstPersonality };
  }, [forSlot, slotAPersonality, slotBPersonality]);

  const onSubmit = async () => {
    try {
      const trimmedTopic = topic.trim();
      if (!trimmedTopic) {
        toast.error("Topic is required.");
        return;
      }
      if (!slotAPersonality.trim() || !slotBPersonality.trim()) {
        toast.error("Please describe both personalities.");
        return;
      }
      if (slotAPersonality.trim().toLowerCase() === slotBPersonality.trim().toLowerCase()) {
        toast.error("Please provide two distinct personality descriptions.");
        return;
      }

      const forPersonality =
        forSlot === "A" ? slotAPersonality : slotBPersonality;
      const againstPersonality =
        forSlot === "A" ? slotBPersonality : slotAPersonality;

      const payload: CreateDebateInput = {
        topic: trimmedTopic,
        rounds,
        statementsPerRound: 1,
        agents: [
          {
            name: (forSlot === "A" ? agentAName : agentBName).trim() || "For Agent",
            provider: "GROQ",
            personality: ` ${forPersonality}. Stance: FOR the topic.`,
          },
          {
            name: (forSlot === "A" ? agentBName : agentAName).trim() || "Against Agent",
            provider: "GROQ",
            personality: ` ${againstPersonality}. Stance: AGAINST the topic.`,
          },
        ],
      };

      const createdId = await createAndStart(payload, false);
      if (createdId) {
        onOpenChange(false);
      }
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "Failed to create debate");
    }
  };

  return (
    <Dialog.Root open={open} onOpenChange={onOpenChange}>
      <Dialog.Portal>
        <Dialog.Overlay className="fixed inset-0 bg-black/60" />
        <Dialog.Content className="fixed left-1/2 top-1/2 max-h-[88vh] w-[min(760px,94vw)] -translate-x-1/2 -translate-y-1/2 overflow-y-auto rounded-xl border border-border bg-card p-5">
          <div className="mb-4 flex items-center justify-between">
            <Dialog.Title className="font-display text-2xl text-text-primary">
              Create New Debate
            </Dialog.Title>
            <Dialog.Close className="rounded border border-border p-2 text-text-muted transition-all duration-200 hover:text-text-primary">
              <X className="h-4 w-4" />
            </Dialog.Close>
          </div>

          <div className="space-y-4">
            <label className="block">
              <span className="mb-1 block text-sm text-text-muted">Topic</span>
              <textarea
                value={topic}
                onChange={(e) => setTopic(e.target.value)}
                className="min-h-24 w-full rounded-xl border border-border bg-elevated p-3 text-sm text-text-primary outline-none transition-all duration-200 focus:border-text-subtle"
                placeholder="What should the agents debate?"
                required
              />
            </label>

            <label>
              <span className="mb-1 block text-sm text-text-muted">Number of rounds</span>
              <input
                type="number"
                min={1}
                max={10}
                value={rounds}
                onChange={(e) => setRounds(Number(e.target.value))}
                className="w-full rounded-xl border border-border bg-elevated p-3 text-sm text-text-primary outline-none transition-all duration-200 focus:border-text-subtle"
              />
            </label>

            <section className="space-y-3 rounded-xl border border-border bg-elevated p-3">
              <h3 className="text-sm font-semibold text-text-primary">Describe both agents</h3>
              <div className="grid grid-cols-1 gap-3 md:grid-cols-2">
                <label>
                  <span className="mb-1 block text-xs uppercase tracking-wide text-text-muted">
                    Agent A name
                  </span>
                  <input
                    value={agentAName}
                    onChange={(e) => setAgentAName(e.target.value)}
                    className="w-full rounded-lg border border-border bg-card p-2 text-sm text-text-primary outline-none"
                    placeholder="e.g. The Logician"
                  />
                </label>

                <label>
                  <span className="mb-1 block text-xs uppercase tracking-wide text-text-muted">
                    Agent B name
                  </span>
                  <input
                    value={agentBName}
                    onChange={(e) => setAgentBName(e.target.value)}
                    className="w-full rounded-lg border border-border bg-card p-2 text-sm text-text-primary outline-none"
                    placeholder="e.g. The Firebrand"
                  />
                </label>
              </div>

              <label className="block">
                <span className="mb-1 block text-xs uppercase tracking-wide text-text-muted">
                  Personality A description
                </span>
                <textarea
                  value={slotAPersonality}
                  onChange={(e) => setSlotAPersonality(e.target.value)}
                  className="min-h-20 w-full rounded-lg border border-border bg-card p-2 text-sm text-text-primary outline-none"
                  placeholder="Describe tone, argument style, strengths, and behavior for Agent A"
                />
              </label>

              <label className="block">
                <span className="mb-1 block text-xs uppercase tracking-wide text-text-muted">
                  Personality B description
                </span>
                <textarea
                  value={slotBPersonality}
                  onChange={(e) => setSlotBPersonality(e.target.value)}
                  className="min-h-20 w-full rounded-lg border border-border bg-card p-2 text-sm text-text-primary outline-none"
                  placeholder="Describe tone, argument style, strengths, and behavior for Agent B"
                />
              </label>
            </section>

            <section className="space-y-3 rounded-xl border border-border bg-elevated p-3">
              <h3 className="text-sm font-semibold text-text-primary">Assign sides</h3>
              <label>
                <span className="mb-1 block text-xs uppercase tracking-wide text-text-muted">
                  Which personality argues FOR the topic?
                </span>
                <select
                  value={forSlot}
                  onChange={(e) => setForSlot(e.target.value as Slot)}
                  className="w-full rounded-lg border border-border bg-card p-2 text-sm text-text-primary outline-none"
                >
                  <option value="A">Personality A</option>
                  <option value="B">Personality B</option>
                </select>
              </label>
              <p className="text-xs text-text-muted">
                FOR: {roleSummary.forPersonality} | AGAINST: {roleSummary.againstPersonality}
              </p>
            </section>

            <button
              type="button"
              disabled={loading}
              onClick={onSubmit}
              className="w-full rounded-xl border border-border bg-text-primary px-4 py-3 text-sm font-semibold text-bg transition-all duration-200 hover:opacity-90 disabled:cursor-not-allowed disabled:opacity-60"
            >
              {loading ? "Creating debate..." : "Create Debate"}
            </button>
          </div>
        </Dialog.Content>
      </Dialog.Portal>
    </Dialog.Root>
  );
}
