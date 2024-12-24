package core

import (
	"sort"
	"strings"
)

type MatchResult struct {
	AgentID AgentID
	Score   float64
	Action  Action
}

type MatcherConfig struct {
	PathWeight   float64
	ActionWeight float64
	MinimumScore float64
	MaxMatches   int
}

func DefaultMatcherConfig() MatcherConfig {
	return MatcherConfig{
		PathWeight:   0.6,
		ActionWeight: 0.4,
		MinimumScore: 0.7,
		MaxMatches:   5,
	}
}

type CapabilityMatcher struct {
	registry *CapabilityRegistry
	config   MatcherConfig
}

func NewCapabilityMatcher(registry *CapabilityRegistry, config MatcherConfig) *CapabilityMatcher {
	return &CapabilityMatcher{
		registry: registry,
		config:   config,
	}
}

func (m *CapabilityMatcher) FindMatchingAgents(task *Task) []MatchResult {
	m.registry.mu.RLock()
	defer m.registry.mu.RUnlock()

	var matches []MatchResult
	for agentID, agentCap := range m.registry.capabilities {
		// Find best matching action and capability
		bestMatch := m.calculateAgentMatch(task.Requirements, agentCap)
		if bestMatch.Score >= m.config.MinimumScore {
			bestMatch.AgentID = agentID
			matches = append(matches, bestMatch)
		}
	}
	return m.selectTopMatches(matches)
}

func (m *CapabilityMatcher) calculateAgentMatch(req TaskRequirement, agentCap AgentCapability) MatchResult {
	var bestMatch MatchResult
	bestMatch.Score = -1

	for _, cap := range agentCap.Capabilities {
		pathScore := m.calculatePathScore(req.SkillPath, cap.SkillPath)

		for _, action := range agentCap.Actions {
			actionScore := m.calculateActionScore(req.Action, action.Name)
			totalScore := (pathScore * m.config.PathWeight) + (actionScore * m.config.ActionWeight)

			if totalScore > bestMatch.Score {
				bestMatch.Score = totalScore
				bestMatch.Action = action
			}
		}
	}

	return bestMatch
}

func (m *CapabilityMatcher) calculatePathScore(reqPath TaskPath, capPath []string) float64 {
	if len(reqPath) == 0 || len(capPath) == 0 {
		return 0.0
	}

	matches := 0
	maxLen := min(len(reqPath), len(capPath))

	for i := 0; i < maxLen; i++ {
		if strings.EqualFold(reqPath[i], capPath[i]) {
			matches++
		} else {
			break
		}
	}

	return float64(matches) / float64(len(reqPath))
}

func (m *CapabilityMatcher) calculateActionScore(reqAction string, actionName string) float64 {
	if strings.EqualFold(reqAction, actionName) {
		return 1.0
	}
	return 0.0
}

func (m *CapabilityMatcher) selectTopMatches(matches []MatchResult) []MatchResult {
	sort.Slice(matches, func(i, j int) bool {
		return matches[i].Score > matches[j].Score
	})

	if len(matches) > m.config.MaxMatches {
		matches = matches[:m.config.MaxMatches]
	}
	return matches
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
