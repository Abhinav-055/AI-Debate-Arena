import { gql } from "@apollo/client";

export const MESSAGE_ADDED = gql`
  subscription MessageAdded($debateId: ID!) {
    messageAdded(debateId: $debateId) {
      id
      debateId
      agentId
      roundNumber
      content
      createdAt
    }
  }
`;

export const DEBATE_STATUS_CHANGED = gql`
  subscription DebateStatusChanged($debateId: ID!) {
    debateStatusChanged(debateId: $debateId) {
      id
      status
      roundsTotal
      roundsDone
    }
  }
`;

export const JUDGEMENT_READY = gql`
  subscription JudgementReady($debateId: ID!) {
    judgementReady(debateId: $debateId) {
      id
      debateId
      winnerAgentId
      strengths {
        agentId
        reason
      }
      weaknesses {
        agentId
        reason
      }
      verdict
      createdAt
    }
  }
`;
