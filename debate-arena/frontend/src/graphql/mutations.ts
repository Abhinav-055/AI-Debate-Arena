import { gql } from "@apollo/client";

export const CREATE_DEBATE = gql`
  mutation CreateDebate($input: CreateDebateInput!) {
    createDebate(input: $input) {
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
      }
    }
  }
`;

export const START_DEBATE = gql`
  mutation StartDebate($id: ID!) {
    startDebate(id: $id) {
      id
      status
      roundsTotal
      roundsDone
    }
  }
`;

export const CAST_VOTE = gql`
  mutation CastVote($debateId: ID!, $agentId: ID!) {
    castVote(debateId: $debateId, agentId: $agentId)
  }
`;
