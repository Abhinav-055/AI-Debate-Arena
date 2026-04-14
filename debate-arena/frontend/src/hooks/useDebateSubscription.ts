import { useApolloClient } from "@apollo/client";
import { useEffect } from "react";
import {
  DEBATE_STATUS_CHANGED,
  JUDGEMENT_READY,
  MESSAGE_ADDED,
} from "../graphql/subscriptions";
import { useDebateStore } from "../store/debateStore";
import type { DebateStatus, Judgement, Message } from "../types/debate";

interface MessageAddedData {
  messageAdded: Message;
}

interface StatusChangedData {
  debateStatusChanged: {
    id: string;
    status: DebateStatus;
    roundsDone?: number;
  };
}

interface JudgementReadyData {
  judgementReady: Judgement;
}

export function useDebateSubscription(debateId: string | undefined) {
  const client = useApolloClient();
  const appendMessage = useDebateStore((s) => s.appendMessage);
  const setStatus = useDebateStore((s) => s.setStatus);
  const setCurrentRound = useDebateStore((s) => s.setCurrentRound);
  const setJudgement = useDebateStore((s) => s.setJudgement);

  useEffect(() => {
    if (!debateId) {
      return;
    }

    const variables = { debateId };

    const messageSub = client
      .subscribe<MessageAddedData>({ query: MESSAGE_ADDED, variables })
      .subscribe({
        next: ({ data }) => {
          if (data?.messageAdded) {
            appendMessage(data.messageAdded);
          }
        },
        error: () => undefined,
      });

    const statusSub = client
      .subscribe<StatusChangedData>({ query: DEBATE_STATUS_CHANGED, variables })
      .subscribe({
        next: ({ data }) => {
          if (data?.debateStatusChanged) {
            setStatus(data.debateStatusChanged.status);
            if (typeof data.debateStatusChanged.roundsDone === "number") {
              setCurrentRound(data.debateStatusChanged.roundsDone + 1);
            }
          }
        },
        error: () => undefined,
      });

    const judgementSub = client
      .subscribe<JudgementReadyData>({ query: JUDGEMENT_READY, variables })
      .subscribe({
        next: ({ data }) => {
          if (data?.judgementReady) {
            setJudgement(data.judgementReady);
          }
        },
        error: () => undefined,
      });

    return () => {
      messageSub.unsubscribe();
      statusSub.unsubscribe();
      judgementSub.unsubscribe();
    };
  }, [
    appendMessage,
    client,
    debateId,
    setCurrentRound,
    setJudgement,
    setStatus,
  ]);
}
