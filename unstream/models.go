package unstream

type OAIStreamChunk struct {
	ID                  string                  `json:"id"`
	Object              string                  `json:"object"`
	Created             int64                   `json:"created"`
	Model               string                  `json:"model"`
	Choices             []OAIStreamChoice       `json:"choices"`
	Usage               *OAIUsage               `json:"usage,omitempty"`
	ModelFingerprint    string                  `json:"system_fingerprint,omitempty"`
	PromptFilterResults []OAIPromptFilterResult `json:"prompt_filter_results,omitempty"`
}

type OAIStreamChoice struct {
	Delta        OAIStreamDelta `json:"delta"`
	FinishReason *string        `json:"finish_reason,omitempty"`
	Index        int            `json:"index"`
}

type OAIStreamDelta struct {
	Content   *string       `json:"content,omitempty"`
	ToolCalls []OAIToolCall `json:"tool_calls,omitempty"`
	Role      string        `json:"role,omitempty"`
}

type OAIPromptFilterResult struct {
	ContentFilterResults map[string]OAIContentFilterResult `json:"content_filter_results"`
	PromptIndex          int                               `json:"prompt_index"`
}

type OAIContentFilterResult struct {
	Filtered bool   `json:"filtered"`
	Severity string `json:"severity"`
}

type OAIToolCall struct {
	Function OAIToolCallFunction `json:"function"`
	Id       string              `json:"id"`
	Index    int                 `json:"index"`
	Type     string              `json:"type"`
}

type OAIToolCallFunction struct {
	Arguments string `json:"arguments"`
	Name      string `json:"name"`
}

type OAIChatResponse struct {
	ID                  string                  `json:"id"`
	Object              string                  `json:"object"`
	Created             int64                   `json:"created"`
	Model               string                  `json:"model"`
	SystemFingerprint   string                  `json:"system_fingerprint,omitempty"`
	Choices             []OAIChatChoice         `json:"choices"`
	Usage               *OAIUsage               `json:"usage,omitempty"`
	PromptFilterResults []OAIPromptFilterResult `json:"prompt_filter_results,omitempty"`
}

type OAIChatChoice struct {
	FinishReason         string                            `json:"finish_reason"`
	Index                int                               `json:"index"`
	ContentFilterResults map[string]OAIContentFilterResult `json:"content_filter_results,omitempty"`
	Message              OAIChatMessage                    `json:"message"`
}

type OAIChatMessage struct {
	Role      string        `json:"role"`
	Content   *string       `json:"content,omitempty"`
	ToolCalls []OAIToolCall `json:"tool_calls,omitempty"`
	Padding   string        `json:"padding,omitempty"`
}

type OAIUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}
