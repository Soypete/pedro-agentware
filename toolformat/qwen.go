package toolformat

import (
	"encoding/json"
	"fmt"
	"regexp"

	"github.com/soypete/pedro-agentware/tools"
)

type QwenFormatter struct{}

var qwenToolCallRegex = regexp.MustCompile(`<tool_call>\s*<tool name="([^"]+)">\s*(.*?)\s*</tool>\s*</tool_call>`)

func (f *QwenFormatter) FormatToolDefinitions(toolsList []tools.Tool) string {
	if len(toolsList) == 0 {
		return ""
	}

	var result string
	for _, t := range toolsList {
		result += fmt.Sprintf(`<tool_description>
<tool_name>%s</tool_name>
<description>%s</description>
`, t.Name(), t.Description())

		if et, ok := t.(tools.ExtendedTool); ok {
			if schema := et.InputSchema(); schema != nil {
				schemaJSON, _ := json.Marshal(schema)
				result += fmt.Sprintf(`<parameters>%s</parameters>`, schemaJSON)
			}
		}
		result += "</tool_description>\n"
	}
	return result
}

func (f *QwenFormatter) ParseToolCalls(response string) ([]ParsedToolCall, error) {
	if response == "" {
		return nil, nil
	}

	matches := qwenToolCallRegex.FindAllStringSubmatch(response, -1)
	if len(matches) == 0 {
		return nil, nil
	}

	result := make([]ParsedToolCall, 0, len(matches))
	for _, m := range matches {
		if len(m) < 3 {
			continue
		}
		name := m[1]
		argsRaw := m[2]

		var args map[string]any
		if err := json.Unmarshal([]byte(argsRaw), &args); err != nil {
			args = map[string]any{"_raw": argsRaw}
		}

		result = append(result, ParsedToolCall{
			Name: name,
			Args: args,
			Raw:  m[0],
		})
	}
	return result, nil
}

func (f *QwenFormatter) FormatToolResult(name string, result *tools.Result) string {
	if result.Success {
		return fmt.Sprintf(`<tool_response>
<tool_name>%s</tool_name>
<result>%s</result>
</tool_response>`, name, result.Output)
	}
	return fmt.Sprintf(`<tool_response>
<tool_name>%s</tool_name>
<error>%s</error>
</tool_response>`, name, result.Error)
}

func (f *QwenFormatter) ModelFamily() string {
	return "qwen"
}
