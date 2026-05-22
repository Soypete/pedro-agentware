package guardrails

import "errors"

var ErrStepNotAllowed = errors.New("step not allowed: prerequisite steps must be completed first")

type StepEnforcer struct {
	stepDefinitions  map[string][]string
	completedSteps   map[string]map[string]bool
	allowedTerminals map[string]map[string]bool
}

func NewStepEnforcer() *StepEnforcer {
	return &StepEnforcer{
		stepDefinitions:  make(map[string][]string),
		completedSteps:   make(map[string]map[string]bool),
		allowedTerminals: make(map[string]map[string]bool),
	}
}

func (se *StepEnforcer) AddStep(tool string, prerequisites []string) {
	se.stepDefinitions[tool] = prerequisites
}

func (se *StepEnforcer) AddTerminal(tool string, allowed map[string]bool) {
	se.allowedTerminals[tool] = allowed
}

func (se *StepEnforcer) MarkStepComplete(sessionID, step string) {
	if se.completedSteps[sessionID] == nil {
		se.completedSteps[sessionID] = make(map[string]bool)
	}
	se.completedSteps[sessionID][step] = true
}

func (se *StepEnforcer) ResetSession(sessionID string) {
	delete(se.completedSteps, sessionID)
}

func (se *StepEnforcer) CanExecute(sessionID, tool string) (bool, []string) {
	prereqs, ok := se.stepDefinitions[tool]
	if !ok {
		return true, nil
	}

	completed := se.completedSteps[sessionID]
	if completed == nil {
		completed = make(map[string]bool)
	}

	var missing []string
	for _, prereq := range prereqs {
		if !completed[prereq] {
			missing = append(missing, prereq)
		}
	}

	return len(missing) == 0, missing
}

func (se *StepEnforcer) ValidateExecution(sessionID, tool string) error {
	allowed, missing := se.CanExecute(sessionID, tool)
	if allowed {
		return nil
	}
	return &StepNotAllowedError{
		Tool:         tool,
		MissingSteps: missing,
	}
}

func (se *StepEnforcer) IsTerminalAllowed(sessionID, terminalTool string) bool {
	allowed, _ := se.CanExecute(sessionID, terminalTool)
	return allowed
}

func (se *StepEnforcer) GetAllowedTerminals(sessionID string) []string {
	var result []string
	for tool := range se.stepDefinitions {
		if allowed, _ := se.CanExecute(sessionID, tool); allowed {
			result = append(result, tool)
		}
	}
	return result
}

type StepNotAllowedError struct {
	Tool         string
	MissingSteps []string
}

func (e *StepNotAllowedError) Error() string {
	return "step not allowed"
}

func (e *StepNotAllowedError) Is(target error) bool {
	return target == ErrStepNotAllowed
}

func (e *StepNotAllowedError) Missing() []string {
	return e.MissingSteps
}
