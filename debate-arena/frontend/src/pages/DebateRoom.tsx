import { useEffect, useMemo, useRef, useState } from "react";
import { useMutation, useQuery } from "@apollo/client";
import { useNavigate, useParams } from "react-router-dom";
import toast from "react-hot-toast";
import { GET_DEBATE, GET_JUDGEMENT, GET_MESSAGES } from "../graphql/queries";
import { START_DEBATE } from "../graphql/mutations";
import { useDebateSubscription } from "../hooks/useDebateSubscription";
import { useDebateStore } from "../store/debateStore";
import type { Debate, Judgement, Message } from "../types/debate";
import { MessageBubble } from "../components/debate/MessageBubble";
import { DebateStatusBadge } from "../components/debate/DebateStatusBadge";
import { RoundIndicator } from "../components/debate/RoundIndicator";
import { AgentTag } from "../components/debate/AgentTag";
import { JudgementPanel } from "../components/judgement/JudgementPanel";

interface DebateData {
  debate: Debate | null;
}

interface MessagesData {
  messages: Message[];
}

interface JudgementData {
  judgement: Judgement | null;
}

export default function DebateRoom() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();

  const [isThinking, setIsThinking] = useState(false);
  const [renderedMessages, setRenderedMessages] = useState<Message[]>([]);
  const [replaying, setReplaying] = useState(false);
  const [visibleCount, setVisibleCount] = useState(0);

  const endRef = useRef<HTMLDivElement | null>(null);

  const {
    messages,
    debateStatus,
    currentRound,
    judgement,
    winnerAgentId,
    setActiveDebate,
    setStatus,
    setCurrentRound,
    setJudgement,
    appendMessage,
    resetDebate,
  } = useDebateStore();

  const { data, loading, error } = useQuery<DebateData>(GET_DEBATE, {
    variables: { id },
    skip: !id,
    fetchPolicy: "cache-and-network",
    pollInterval: 2000,
  });

  const { data: judgementData } = useQuery<JudgementData>(GET_JUDGEMENT, {
    variables: { debateId: id },
    skip: !id || debateStatus !== "COMPLETED",
    pollInterval: debateStatus === "COMPLETED" ? 2000 : 0,
  });

  const { data: messagesData } = useQuery<MessagesData>(GET_MESSAGES, {
    variables: { debateId: id },
    skip: !id,
    fetchPolicy: "cache-and-network",
    pollInterval: 2000,
  });

  const [startDebate] = useMutation(START_DEBATE);

  useDebateSubscription(id);

  const debate = data?.debate;
  const activeMessages = replaying ? renderedMessages : messages;

  useEffect(() => {
    if (!id) {
      navigate("/");
      return;
    }

    setActiveDebate(id);
    setVisibleCount(0);
    return () => {
      resetDebate();
    };
  }, [id, navigate, resetDebate, setActiveDebate]);

  useEffect(() => {
    if (!debate) {
      return;
    }
    setStatus(debate.status);
    setCurrentRound(Math.max(1, debate.roundsDone + 1));
  }, [debate, setCurrentRound, setStatus]);

  useEffect(() => {
    if (!messagesData?.messages) {
      return;
    }
    messagesData.messages.forEach((message) => appendMessage(message));
  }, [appendMessage, messagesData?.messages, resetDebate]);

  useEffect(() => {
    if (judgementData?.judgement) {
      setJudgement(judgementData.judgement);
    }
  }, [judgementData?.judgement, setJudgement]);

  useEffect(() => {
    endRef.current?.scrollIntoView({ behavior: "smooth", block: "end" });
  }, [activeMessages]);

  useEffect(() => {
    if (debateStatus !== "ACTIVE") {
      setIsThinking(false);
      return;
    }

    const timer = setTimeout(() => {
      setIsThinking(true);
    }, 3000);

    return () => {
      clearTimeout(timer);
      setIsThinking(false);
    };
  }, [debateStatus, messages.length]);

  const speakingAgent = useMemo(() => {
    if (!debate || debate.agents.length === 0 || debateStatus === "COMPLETED") {
      return null;
    }
    if (debateStatus === "PENDING") {
      return debate.agents[0] ?? null;
    }
    const nextIndex = messages.length % debate.agents.length;
    return debate.agents[nextIndex] ?? null;
  }, [debate, debateStatus, messages.length]);

  const messagesWithAgents = useMemo(() => {
    return [...activeMessages]
      .sort((a, b) => new Date(a.createdAt).getTime() - new Date(b.createdAt).getTime())
      .map((message) => ({
      ...message,
      agent:
        message.agent ??
        debate?.agents.find((agent) => agent.id === message.agentId),
    }));
  }, [activeMessages, debate?.agents]);

  useEffect(() => {
    if (replaying) {
      return;
    }
    if (messagesWithAgents.length === 0) {
      setVisibleCount(0);
      return;
    }
    if (visibleCount >= messagesWithAgents.length) {
      return;
    }
    const timer = setTimeout(() => {
      setVisibleCount((count) => Math.min(count + 1, messagesWithAgents.length));
    }, 450);
    return () => clearTimeout(timer);
  }, [messagesWithAgents.length, replaying, visibleCount]);

  const displayedMessages = replaying
    ? messagesWithAgents
    : messagesWithAgents.slice(0, visibleCount);

  const messageCountByAgent = useMemo(() => {
    const counts = new Map<string, number>();
    messages.forEach((m) => {
      counts.set(m.agentId, (counts.get(m.agentId) ?? 0) + 1);
    });
    return counts;
  }, [messages]);

  if (!id) {
    return null;
  }

  if (error) {
    toast.error(error.message);
    navigate("/");
    return null;
  }

  const onStartDebate = async () => {
    try {
      await startDebate({ variables: { id } });
      toast.success("Debate started");
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Failed to start debate");
    }
  };

  const onWatchReplay = async () => {
    try {
      setReplaying(true);
      setRenderedMessages([]);
      for (const message of messages) {
        await new Promise((resolve) => setTimeout(resolve, 1500));
        setRenderedMessages((current) => [...current, message]);
      }
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Replay failed");
    } finally {
      setReplaying(false);
    }
  };

  if (loading || !debate) {
    return (
      <div className="grid h-[80vh] grid-cols-1 gap-4 lg:grid-cols-5">
        <div className="lg:col-span-3 rounded-xl border border-border bg-[linear-gradient(110deg,#1C1C1F,45%,#25252A,55%,#1C1C1F)] bg-[length:200%_100%] animate-shimmer" />
        <div className="lg:col-span-2 rounded-xl border border-border bg-[linear-gradient(110deg,#1C1C1F,45%,#25252A,55%,#1C1C1F)] bg-[length:200%_100%] animate-shimmer" />
      </div>
    );
  }

  return (
    <div className="space-y-4">
      <div className="grid h-[82vh] grid-cols-1 gap-4 lg:grid-cols-5">
        <section className="lg:col-span-3 flex h-full flex-col rounded-xl border border-border bg-card p-4">
          <header className="mb-4 flex items-center justify-between">
            <h1 className="font-display text-2xl text-text-primary">{debate.topic}</h1>
            <DebateStatusBadge status={debateStatus} />
          </header>

          <div className="flex-1 space-y-3 overflow-y-auto pr-2">
            {displayedMessages.length === 0 ? (
              <div className="rounded-xl border border-dashed border-border p-8 text-center text-text-muted">
                No messages yet. Start the debate to begin.
              </div>
            ) : (
              displayedMessages.map((message) => (
                <MessageBubble key={message.id} message={message} />
              ))
            )}

            {isThinking ? (
              <div className="rounded-xl border border-border bg-[linear-gradient(110deg,#1C1C1F,45%,#25252A,55%,#1C1C1F)] bg-[length:200%_100%] p-4 animate-shimmer">
                <p className="text-sm text-text-muted">Agent is thinking...</p>
              </div>
            ) : null}
            <div ref={endRef} />
          </div>
        </section>

        <aside className="lg:col-span-2 h-full space-y-4 overflow-y-auto rounded-xl border border-border bg-card p-4">
          <div>
            <RoundIndicator roundsTotal={debate.roundsTotal} roundsDone={debate.roundsDone} />
            <p className="mt-2 text-sm text-text-muted">
              {debateStatus === "COMPLETED"
                ? "Debate over"
                : `Round ${Math.min(currentRound, debate.roundsTotal)} of ${debate.roundsTotal}${speakingAgent ? ` - ${speakingAgent.name} is speaking...` : ""}`}
            </p>
          </div>

          {debateStatus === "PENDING" && debate.roundsDone === 0 && messages.length === 0 ? (
            <button
              type="button"
              onClick={onStartDebate}
              className="w-full rounded-xl border border-status-active/40 bg-status-active/10 px-4 py-3 text-sm font-semibold text-status-active transition-all duration-200 hover:bg-status-active/20"
            >
              Start Debate
            </button>
          ) : null}

          <div className="space-y-3">
            {debate.agents.map((agent) => (
              <article
                key={agent.id}
                className="rounded-xl border border-border bg-elevated p-3"
              >
                <AgentTag name={agent.name} provider={agent.provider} turnOrder={agent.turnOrder} />
                <p className="mt-2 text-xs text-text-muted">{agent.personality}</p>
                <p className="mt-2 font-mono text-xs text-text-subtle">
                  Messages: {messageCountByAgent.get(agent.id) ?? 0}
                </p>
              </article>
            ))}
          </div>

          {judgement ? (
            <JudgementPanel
              judgement={judgement}
              agents={debate.agents}
              winnerAgentId={winnerAgentId}
              onWatchReplay={onWatchReplay}
            />
          ) : null}
        </aside>
      </div>
    </div>
  );
}
