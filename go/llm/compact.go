package llm

import "errors"

var ErrEmptyMessages = errors.New("no messages to compact")

type TokenCounter func([]Message) int

type CompactionStrategy interface {
	Compact(messages []Message, targetTokens int, counter TokenCounter) ([]Message, error)
	Name() string
}

type TieredCompact struct {
	KeepRecent      int
	PhaseThresholds [3]float64
	TruncateChars   int
}

func NewTieredCompact() *TieredCompact {
	return &TieredCompact{
		KeepRecent:      2,
		PhaseThresholds: [3]float64{0.75, 0.75, 0.75},
		TruncateChars:   200,
	}
}

func (t *TieredCompact) Name() string {
	return "TieredCompact"
}

func (t *TieredCompact) Compact(messages []Message, targetTokens int, counter TokenCounter) ([]Message, error) {
	if len(messages) == 0 {
		return nil, ErrEmptyMessages
	}

	result := make([]Message, len(messages))
	copy(result, messages)

	eligibleEnd := findEligibleEnd(result, t.KeepRecent)

	currentTokens := counter(result)

	if currentTokens <= targetTokens {
		return result, nil
	}

	result = t.phase1Compact(result, eligibleEnd, counter)
	currentTokens = counter(result)
	if currentTokens <= targetTokens {
		return result, nil
	}

	result = t.phase2Compact(result, eligibleEnd, counter)
	currentTokens = counter(result)
	if currentTokens <= targetTokens {
		return result, nil
	}

	result = t.phase3Compact(result, eligibleEnd, counter)
	return result, nil
}

func findEligibleEnd(messages []Message, keepRecent int) int {
	if keepRecent <= 0 {
		return len(messages) - 1
	}

	maxStep := -1
	for _, m := range messages {
		if m.Meta.StepIndex != nil && *m.Meta.StepIndex > maxStep {
			maxStep = *m.Meta.StepIndex
		}
	}

	if maxStep < 0 {
		protectedCount := keepRecent
		if protectedCount >= len(messages) {
			protectedCount = len(messages)
		}
		return len(messages) - 1 - protectedCount
	}

	protectedSteps := make(map[int]bool)
	currentStep := maxStep
	for i := 0; i < keepRecent; i++ {
		protectedSteps[currentStep] = true
		currentStep--
		if currentStep < 0 {
			break
		}
	}

	for i := len(messages) - 1; i >= 0; i-- {
		stepIdx := 0
		if messages[i].Meta.StepIndex != nil {
			stepIdx = *messages[i].Meta.StepIndex
		}
		if !protectedSteps[stepIdx] {
			return i
		}
	}

	return -1
}

func isNudgeType(msgType MessageType) bool {
	return msgType == MessageTypeStepNudge ||
		msgType == MessageTypeRetryNudge ||
		msgType == MessageTypePrerequisiteNudge
}

func (t *TieredCompact) protectedSteps(messages []Message) map[int]bool {
	protected := make(map[int]bool)

	if t.KeepRecent <= 0 {
		return protected
	}

	steps := make([]int, 0)
	stepSet := make(map[int]bool)
	for _, m := range messages {
		if m.Meta.StepIndex != nil && !stepSet[*m.Meta.StepIndex] {
			steps = append(steps, *m.Meta.StepIndex)
			stepSet[*m.Meta.StepIndex] = true
		}
	}

	if len(steps) == 0 {
		return protected
	}

	if t.KeepRecent >= len(steps) {
		for _, s := range steps {
			protected[s] = true
		}
		return protected
	}

	startIdx := len(steps) - t.KeepRecent
	for i := startIdx; i < len(steps); i++ {
		protected[steps[i]] = true
	}

	return protected
}

func (t *TieredCompact) isProtected(m *Message, protectedSteps map[int]bool) bool {
	if m.Meta.StepIndex == nil {
		return false
	}
	return protectedSteps[*m.Meta.StepIndex]
}

func (t *TieredCompact) phase1Compact(messages []Message, eligibleEnd int, counter TokenCounter) []Message {
	result := make([]Message, 0, len(messages))
	protected := t.protectedSteps(messages)

	for i, m := range messages {
		if i == 0 || i == 1 {
			result = append(result, m)
			continue
		}

		if t.isProtected(&m, protected) {
			result = append(result, m)
			continue
		}

		if isNudgeType(m.Meta.Type) {
			continue
		}

		if m.Meta.Type == MessageTypeToolResult {
			if len(m.Content) > t.TruncateChars {
				truncated := m
				truncated.Content = m.Content[:t.TruncateChars]
				result = append(result, truncated)
				continue
			}
		}

		result = append(result, m)
	}

	return result
}

func (t *TieredCompact) phase2Compact(messages []Message, eligibleEnd int, counter TokenCounter) []Message {
	result := make([]Message, 0, len(messages))
	protected := t.protectedSteps(messages)

	for i, m := range messages {
		if i == 0 || i == 1 {
			result = append(result, m)
			continue
		}

		if t.isProtected(&m, protected) {
			result = append(result, m)
			continue
		}

		if m.Meta.Type == MessageTypeToolResult {
			continue
		}

		result = append(result, m)
	}

	return result
}

func (t *TieredCompact) phase3Compact(messages []Message, eligibleEnd int, counter TokenCounter) []Message {
	result := make([]Message, 0, len(messages))
	protected := t.protectedSteps(messages)

	for i, m := range messages {
		if i == 0 || i == 1 {
			result = append(result, m)
			continue
		}

		if t.isProtected(&m, protected) {
			result = append(result, m)
			continue
		}

		if m.Meta.Type == MessageTypeReasoning || m.Meta.Type == MessageTypeTextResponse {
			continue
		}

		result = append(result, m)
	}

	return result
}
