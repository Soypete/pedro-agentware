package tools

type Result struct {
	Success       bool
	Output        string
	Error         string
	ModifiedFiles []string
	Metadata      map[string]any
}
