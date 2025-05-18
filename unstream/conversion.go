package unstream

import (
	"strings"
)

// OAIStreamCollector collects OpenAI stream chunks and builds a standard response.
type OAIStreamCollector struct {
	ID                  string
	Object              string
	Created             int64
	Model               string
	SystemFingerprint   string
	PromptFilterResults []OAIPromptFilterResult
	Usage               *OAIUsage

	// For each choice index, collect content and tool calls
	choices map[int]*collectedChoice
}

type collectedChoice struct {
	role                 string
	content              *strings.Builder
	toolCalls            map[int]*OAIToolCall // index -> tool call
	toolCallsOrder       []int                // preserve order
	finishReason         *string
	contentFilterResults map[string]OAIContentFilterResult
}

func NewOAIStreamCollector() *OAIStreamCollector {
	return &OAIStreamCollector{
		choices: make(map[int]*collectedChoice),
	}
}

// AddChunk processes a single OAIStreamChunk and updates the collector.
func (c *OAIStreamCollector) AddChunk(chunk *OAIStreamChunk) {
	// Set top-level fields if present
	if chunk.ID != "" {
		c.ID = chunk.ID
	}
	if chunk.Object != "" {
		c.Object = chunk.Object
	}
	if chunk.Created != 0 {
		c.Created = chunk.Created
	}
	if chunk.Model != "" {
		c.Model = chunk.Model
	}
	if chunk.ModelFingerprint != "" {
		c.SystemFingerprint = chunk.ModelFingerprint
	}
	if len(chunk.PromptFilterResults) > 0 {
		c.PromptFilterResults = chunk.PromptFilterResults
	}
	if chunk.Usage != nil {
		c.Usage = &OAIUsage{
			PromptTokens:     chunk.Usage.PromptTokens,
			CompletionTokens: chunk.Usage.CompletionTokens,
			TotalTokens:      chunk.Usage.TotalTokens,
		}
	}

	for _, ch := range chunk.Choices {
		idx := ch.Index
		choice, ok := c.choices[idx]
		if !ok {
			choice = &collectedChoice{
				content:              &strings.Builder{},
				toolCalls:            make(map[int]*OAIToolCall),
				contentFilterResults: make(map[string]OAIContentFilterResult),
			}
			c.choices[idx] = choice
		}
		// Role
		if ch.Delta.Role != "" {
			choice.role = ch.Delta.Role
		}
		// Content
		if ch.Delta.Content != nil {
			choice.content.WriteString(*ch.Delta.Content)
		}
		// Tool calls
		for _, tc := range ch.Delta.ToolCalls {
			// Tool calls are streamed by index, accumulate arguments
			existing, ok := choice.toolCalls[tc.Index]
			if !ok {
				// Copy the struct to avoid pointer aliasing
				copyTC := tc
				choice.toolCalls[tc.Index] = &copyTC
				choice.toolCallsOrder = append(choice.toolCallsOrder, tc.Index)
			} else {
				// Append arguments if present
				if tc.Function.Arguments != "" {
					existing.Function.Arguments += tc.Function.Arguments
				}
			}
		}
		// Finish reason
		if ch.FinishReason != nil {
			choice.finishReason = ch.FinishReason
		}
		// Content filter results
		// Defensive nil check for ch.Delta.Content
		contentLen := 0
		if ch.Delta.Content != nil {
			contentLen = len(*ch.Delta.Content)
		}
		if len(ch.Delta.ToolCalls) == 0 && contentLen == 0 && len(ch.Delta.Role) == 0 && len(ch.Delta.ToolCalls) == 0 {
			// No-op for now, but if content filter results are streamed, handle here
		}
	}
}

// BuildResponse returns the final OAIChatResponse.
func (c *OAIStreamCollector) BuildResponse() *OAIChatResponse {
	choices := make([]OAIChatChoice, 0, len(c.choices))
	for idx, ch := range c.choices {
		// Collect tool calls in order
		var toolCalls []OAIToolCall
		for _, i := range ch.toolCallsOrder {
			tc := ch.toolCalls[i]
			toolCalls = append(toolCalls, *tc)
		}
		contentStr := ch.content.String()
		var contentPtr *string
		if contentStr != "" {
			contentPtr = &contentStr
		}
		choices = append(choices, OAIChatChoice{
			FinishReason:         derefString(ch.finishReason),
			Index:                idx,
			ContentFilterResults: ch.contentFilterResults,
			Message: OAIChatMessage{
				Role:      ch.role,
				Content:   contentPtr,
				ToolCalls: toolCalls,
				Padding:   "", // Not streamed
			},
		})
	}
	return &OAIChatResponse{
		ID:                  c.ID,
		Object:              c.Object,
		Created:             c.Created,
		Model:               c.Model,
		SystemFingerprint:   c.SystemFingerprint,
		Choices:             choices,
		Usage:               c.Usage,
		PromptFilterResults: c.PromptFilterResults,
	}
}

func derefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
