package guardrails

import "fmt"

type NudgeKind string

const (
	NudgeKindRetry        NudgeKind = "retry"
	NudgeKindUnknownTool  NudgeKind = "unknown_tool"
	NudgeKindStep         NudgeKind = "step"
	NudgeKindPrerequisite NudgeKind = "prerequisite"
)

type Nudge struct {
	Role    string
	Content string
	Kind    NudgeKind
	Tier    int
}

func RetryNudge(rawResponse string, toolNames []string) *Nudge {
	toolsList := joinToolNames(toolNames)
	return &Nudge{
		Role:    "user",
		Content: fmt.Sprintf("Your previous response was not a valid tool call. You must respond with a tool call, not free text. Available tools: %s. Please try again with a valid tool call.", toolsList),
		Kind:    NudgeKindRetry,
		Tier:    0,
	}
}

func UnknownToolNudge(toolName string, toolNames []string) *Nudge {
	toolsList := joinToolNames(toolNames)
	return &Nudge{
		Role:    "user",
		Content: fmt.Sprintf("Tool '%s' does not exist. Available tools: %s. Call one of them.", toolName, toolsList),
		Kind:    NudgeKindUnknownTool,
		Tier:    0,
	}
}

func StepNudge(terminalTool string, pendingSteps []string, tier int) *Nudge {
	if tier < 1 {
		tier = 1
	}
	if tier > 3 {
		tier = 3
	}

	steps := joinToolNames(pendingSteps)

	var content string
	switch tier {
	case 1:
		content = fmt.Sprintf("You cannot call %s yet. You must first complete these required steps: %s. Call one of them now.", terminalTool, steps)
	case 2:
		content = fmt.Sprintf("You must call one of these tools now: %s. Pick one.", steps)
	default:
		content = fmt.Sprintf("STOP. You MUST call one of: %s. Do NOT call %s. Your next response MUST be a tool call to one of: %s.", steps, terminalTool, steps)
	}

	return &Nudge{
		Role:    "user",
		Content: content,
		Kind:    NudgeKindStep,
		Tier:    tier,
	}
}

func PrerequisiteNudge(toolName string, missingPrereqs []string) *Nudge {
	prereqs := joinToolNames(missingPrereqs)
	return &Nudge{
		Role:    "user",
		Content: fmt.Sprintf("You cannot call %s yet. You must first call: %s. Call the prerequisite tool now.", toolName, prereqs),
		Kind:    NudgeKindPrerequisite,
		Tier:    0,
	}
}

func joinToolNames(names []string) string {
	if len(names) == 0 {
		return "(no tools available)"
	}
	if len(names) == 1 {
		return names[0]
	}
	if len(names) == 2 {
		return names[0] + " and " + names[1]
	}
	result := names[0]
	for i := 1; i < len(names); i++ {
		if i == len(names)-1 {
			result += ", and " + names[i]
		} else {
			result += ", " + names[i]
		}
	}
	return result
}
