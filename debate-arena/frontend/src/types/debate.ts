export type DebateStatus = "PENDING" | "ACTIVE" | "COMPLETED";
export type AgentProvider = "GROQ";

export interface Agent {
  id: string;
  debateId?: string;
  name: string;
  provider: AgentProvider;
  personality: string;
  turnOrder?: number;
}

export interface Message {
  id: string;
  debateId: string;
  agentId: string;
  roundNumber: number;
  content: string;
  createdAt: string;
  agent?: Agent;
}

export interface JudgementDetail {
  agentId: string;
  reason: string;
}

export interface Judgement {
  id?: string;
  debateId: string;
  winnerAgentId?: string;
  winner?: Agent;
  strengths: JudgementDetail[];
  weaknesses: JudgementDetail[];
  verdict: string;
  createdAt: string;
}

export interface Debate {
  id: string;
  topic: string;
  status: DebateStatus;
  roundsTotal: number;
  roundsDone: number;
  createdAt: string;
  updatedAt?: string;
  agents: Agent[];
  messages?: Message[];
  judgement?: Judgement | null;
}

export interface CreateDebateAgentInput {
  name: string;
  provider: AgentProvider;
  personality: string;
}

export interface CreateDebateInput {
  topic: string;
  rounds: number;
  statementsPerRound: number;
  agents: CreateDebateAgentInput[];
}
