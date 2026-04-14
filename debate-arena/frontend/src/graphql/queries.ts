import { gql } from "@apollo/client";

export const GET_DEBATES = gql`
  query GetDebates($status: DebateStatus, $limit: Int, $offset: Int) {
    debates(status: $status, limit: $limit, offset: $offset) {
      id
      topic
      status
      roundsTotal
      roundsDone
      createdAt
      agents {
        id
        name
        provider
        personality
        turnOrder
      }
    }
  }
`;

export const GET_DEBATE = gql`
  query GetDebate($id: ID!) {
    debate(id: $id) {
      id
      topic
      status
      roundsTotal
      roundsDone
      createdAt
      updatedAt
      agents {
        id
        name
        provider
        personality
        turnOrder
      }
    }
  }
`;

export const GET_MESSAGES = gql`
  query GetMessages($debateId: ID!, $round: Int) {
    messages(debateId: $debateId, round: $round) {
      id
      debateId
      agentId
      roundNumber
      content
      createdAt
    }
  }
`;

export const GET_JUDGEMENT = gql`
  query GetJudgement($debateId: ID!) {
    judgement(debateId: $debateId) {
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
