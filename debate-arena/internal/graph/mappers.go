package graph

import (
	"debate-arena/internal/graph/model"
	"debate-arena/internal/models"
)

func toDomainProvider(p model.AgentProvider) models.AgentProvider {
	_ = p
	return models.AgentProviderGroq
}

func toDomainStatus(s model.DebateStatus) models.DebateStatus {
	switch s {
	case model.DebateStatusActive:
		return models.DebateStatusActive
	case model.DebateStatusCompleted:
		return models.DebateStatusCompleted
	default:
		return models.DebateStatusPending
	}
}

func toGraphProvider(p models.AgentProvider) model.AgentProvider {
	_ = p
	return model.AgentProviderGroq
}

func toGraphStatus(s models.DebateStatus) model.DebateStatus {
	switch s {
	case models.DebateStatusActive:
		return model.DebateStatusActive
	case models.DebateStatusCompleted:
		return model.DebateStatusCompleted
	default:
		return model.DebateStatusPending
	}
}

func toGraphDebate(d models.Debate) model.Debate {
	agents := make([]*model.Agent, 0, len(d.Agents))
	for _, a := range d.Agents {
		ag := model.Agent{
			ID:          a.ID,
			DebateID:    a.DebateID,
			Name:        a.Name,
			Provider:    toGraphProvider(a.Provider),
			Personality: a.Personality,
			TurnOrder:   a.TurnOrder,
		}
		agents = append(agents, &ag)
	}
	return model.Debate{
		ID:          d.ID,
		Topic:       d.Topic,
		Status:      toGraphStatus(d.Status),
		RoundsTotal: d.RoundsTotal,
		RoundsDone:  d.RoundsDone,
		CreatedAt:   d.CreatedAt,
		UpdatedAt:   d.UpdatedAt,
		Agents:      agents,
	}
}

func toGraphMessage(m models.Message) model.Message {
	return model.Message{
		ID:          m.ID,
		DebateID:    m.DebateID,
		AgentID:     m.AgentID,
		RoundNumber: m.RoundNumber,
		Content:     m.Content,
		CreatedAt:   m.CreatedAt,
	}
}

func toGraphJudgement(j models.Judgement) model.Judgement {
	strengths := make([]*model.JudgementDetail, 0, len(j.Strengths))
	for _, s := range j.Strengths {
		d := model.JudgementDetail{AgentID: s.AgentID, Reason: s.Reason}
		strengths = append(strengths, &d)
	}
	weaknesses := make([]*model.JudgementDetail, 0, len(j.Weaknesses))
	for _, w := range j.Weaknesses {
		d := model.JudgementDetail{AgentID: w.AgentID, Reason: w.Reason}
		weaknesses = append(weaknesses, &d)
	}
	return model.Judgement{
		ID:            j.ID,
		DebateID:      j.DebateID,
		WinnerAgentID: j.WinnerAgentID,
		Strengths:     strengths,
		Weaknesses:    weaknesses,
		Verdict:       j.Verdict,
		CreatedAt:     j.CreatedAt,
	}
}
