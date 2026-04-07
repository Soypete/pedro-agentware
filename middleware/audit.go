package middleware

import "time"

type AuditRecord struct {
	SessionID string
	ToolName  string
	Args      map[string]any
	Decision  Decision
	Timestamp time.Time
}

type Auditor interface {
	Record(record AuditRecord)
	Query(filter AuditFilter) []AuditRecord
}

type AuditFilter struct {
	SessionID string
	ToolName  string
	Action    Action
	Since     time.Time
	Limit     int
}

type InMemoryAuditor struct {
	records []AuditRecord
}

func NewInMemoryAuditor() *InMemoryAuditor {
	return &InMemoryAuditor{
		records: make([]AuditRecord, 0),
	}
}

func (a *InMemoryAuditor) Record(record AuditRecord) {
	a.records = append(a.records, record)
}

func (a *InMemoryAuditor) Query(filter AuditFilter) []AuditRecord {
	result := make([]AuditRecord, 0)
	for _, r := range a.records {
		if filter.SessionID != "" && r.SessionID != filter.SessionID {
			continue
		}
		if filter.ToolName != "" && r.ToolName != filter.ToolName {
			continue
		}
		if filter.Action != "" && r.Decision.Action != filter.Action {
			continue
		}
		if !filter.Since.IsZero() && r.Timestamp.Before(filter.Since) {
			continue
		}
		result = append(result, r)
		if filter.Limit > 0 && len(result) >= filter.Limit {
			break
		}
	}
	return result
}
