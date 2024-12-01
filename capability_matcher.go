// capability_matcher.go
package main

import "sort"

// MatchResult represents the outcome of capability matching for an agent
type MatchResult struct {
	AgentID   AgentID
	Score     float64  // Overall match score (0-1)
	TaskTypes []string // Supported task types
	Skills    []string // Available skills for the matched task type
}

// MatcherConfig holds weights for different matching criteria
type MatcherConfig struct {
	TypeWeight   float64 // Weight for task type match (0-1)
	SkillWeight  float64 // Weight for skills match (0-1)
	MinimumScore float64 // Minimum score threshold for valid matches
	MaxMatches   int     // Maximum number of matches to return
}

// DefaultMatcherConfig provides sensible default configuration
func DefaultMatcherConfig() MatcherConfig {
	return MatcherConfig{
		TypeWeight:   0.4,
		SkillWeight:  0.6,
		MinimumScore: 0.7,
		MaxMatches:   5,
	}
}

// CapabilityMatcher handles matching tasks to capable agents
type CapabilityMatcher struct {
	registry *CapabilityRegistry
	config   MatcherConfig
}

// NewCapabilityMatcher creates a new matcher with given configuration
func NewCapabilityMatcher(registry *CapabilityRegistry, config MatcherConfig) *CapabilityMatcher {
	return &CapabilityMatcher{
		registry: registry,
		config:   config,
	}
}

// FindMatchingAgents returns a list of agents sorted by match score
func (m *CapabilityMatcher) FindMatchingAgents(task *Task) []AgentID {
	// Get all matches with their scores
	matches := m.scoreAgents(task)
	if len(matches) == 0 {
		return nil
	}

	// Sort matches by score in descending order
	sort.Slice(matches, func(i, j int) bool {
		return matches[i].Score > matches[j].Score
	})

	// Select top N matches that meet minimum score
	result := make([]AgentID, 0, m.config.MaxMatches)
	for _, match := range matches {
		if match.Score < m.config.MinimumScore {
			break
		}
		if len(result) >= m.config.MaxMatches {
			break
		}
		result = append(result, match.AgentID)
	}

	return result
}

// scoreAgents calculates match scores for all agents
func (m *CapabilityMatcher) scoreAgents(task *Task) []MatchResult {
	m.registry.mu.RLock()
	defer m.registry.mu.RUnlock()

	var matches []MatchResult
	for agentID, cap := range m.registry.capabilities {
		// Check if agent supports the task type
		typeScore := m.matchTaskType(task.Type, cap.TaskTypes)
		if typeScore == 0 {
			continue // Skip if task type doesn't match at all
		}

		// Calculate skill match score
		skillScore := m.matchSkills(task.Type, task.SkillsRequired, cap.SkillsByType)

		// Calculate weighted total score
		totalScore := (typeScore * m.config.TypeWeight) +
			(skillScore * m.config.SkillWeight)

		matches = append(matches, MatchResult{
			AgentID:   agentID,
			Score:     totalScore,
			TaskTypes: cap.TaskTypes,
			Skills:    cap.SkillsByType[task.Type],
		})
	}

	return matches
}

// matchTaskType checks if the agent supports the required task type
func (m *CapabilityMatcher) matchTaskType(taskType string, supportedTypes []string) float64 {
	for _, t := range supportedTypes {
		if t == taskType {
			return 1.0
		}
	}
	return 0.0
}

// matchSkills calculates how well an agent's skills match the required skills
func (m *CapabilityMatcher) matchSkills(taskType string, required []string, skillsByType map[string][]string) float64 {
	if len(required) == 0 {
		return 1.0 // No skills required means perfect match
	}

	available, exists := skillsByType[taskType]
	if !exists {
		return 0.0 // Agent doesn't have any skills for this task type
	}

	matchedSkills := 0
	for _, reqSkill := range required {
		for _, availSkill := range available {
			if reqSkill == availSkill {
				matchedSkills++
				break
			}
		}
	}

	return float64(matchedSkills) / float64(len(required))
}
