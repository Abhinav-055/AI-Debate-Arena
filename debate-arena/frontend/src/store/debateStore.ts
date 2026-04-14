import { create } from "zustand";
import type { DebateStatus, Judgement, Message } from "../types/debate";

interface DebateState {
  activeDebateId: string | null;
  messages: Message[];
  debateStatus: DebateStatus;
  currentRound: number;
  speakingAgentId: string | null;
  judgement: Judgement | null;
  winnerAgentId: string | null;
  setActiveDebate: (debateId: string | null) => void;
  appendMessage: (message: Message) => void;
  setStatus: (status: DebateStatus) => void;
  setCurrentRound: (round: number) => void;
  setSpeakingAgentId: (agentId: string | null) => void;
  setJudgement: (judgement: Judgement | null) => void;
  resetDebate: () => void;
}

const initialState = {
  activeDebateId: null,
  messages: [],
  debateStatus: "PENDING" as DebateStatus,
  currentRound: 1,
  speakingAgentId: null,
  judgement: null,
  winnerAgentId: null,
};

export const useDebateStore = create<DebateState>((set) => ({
  ...initialState,
  setActiveDebate: (activeDebateId) => set({ activeDebateId }),
  appendMessage: (message) =>
    set((state) => {
      const exists = state.messages.some((item) => item.id === message.id);
      const nextMessages = exists ? state.messages : [...state.messages, message];
      return {
        messages: nextMessages,
        currentRound: Math.max(state.currentRound, message.roundNumber),
        speakingAgentId: message.agentId,
      };
    }),
  setStatus: (debateStatus) =>
    set((state) => ({
      debateStatus,
      speakingAgentId: debateStatus === "COMPLETED" ? null : state.speakingAgentId,
    })),
  setCurrentRound: (currentRound) => set({ currentRound }),
  setSpeakingAgentId: (speakingAgentId) => set({ speakingAgentId }),
  setJudgement: (judgement) =>
    set({
      judgement,
      winnerAgentId: judgement?.winnerAgentId ?? null,
    }),
  resetDebate: () => set(initialState),
}));
