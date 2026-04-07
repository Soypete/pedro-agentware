package middleware

type ConditionEvaluator struct{}

func NewConditionEvaluator() *ConditionEvaluator {
	return &ConditionEvaluator{}
}

func (e *ConditionEvaluator) Evaluate(cond Condition, args map[string]any, caller CallerContext) bool {
	return cond.evaluate(args, caller)
}
