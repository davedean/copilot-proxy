package unstream_test

import (
	"bufio"
	. "copilot-proxy/unstream"
	"encoding/json"
	"strings"
	"testing"
)

func TestOAIStreamCollector_ToolCallStreaming(t *testing.T) {
	stream := `
data: {"choices":[],"created":0,"id":"","prompt_filter_results":[{"content_filter_results":{"hate":{"filtered":false,"severity":"safe"},"self_harm":{"filtered":false,"severity":"safe"},"sexual":{"filtered":false,"severity":"safe"},"violence":{"filtered":false,"severity":"safe"}},"prompt_index":0}]}
data: {"choices":[{"index":0,"delta":{"content":null,"role":"assistant","tool_calls":[{"function":{"arguments":"","name":"get_weather"},"id":"call_7m75SYp4UrPhxhtdZdawEK5J","index":0,"type":"function"}]}}],"created":1747591235,"id":"chatcmpl-BYcbLSepxSXIxgUX2WZCFZrjqjp0l","model":"gpt-4o-2024-11-20","system_fingerprint":"fp_ee1d74bde0"}
data: {"choices":[{"index":0,"delta":{"content":null,"tool_calls":[{"function":{"arguments":"{\""},"index":0}]}}],"created":1747591235,"id":"chatcmpl-BYcbLSepxSXIxgUX2WZCFZrjqjp0l","model":"gpt-4o-2024-11-20","system_fingerprint":"fp_ee1d74bde0"}
data: {"choices":[{"index":0,"delta":{"content":null,"tool_calls":[{"function":{"arguments":"location"},"index":0}]}}],"created":1747591235,"id":"chatcmpl-BYcbLSepxSXIxgUX2WZCFZrjqjp0l","model":"gpt-4o-2024-11-20","system_fingerprint":"fp_ee1d74bde0"}
data: {"choices":[{"index":0,"delta":{"content":null,"tool_calls":[{"function":{"arguments":"\":\""},"index":0}]}}],"created":1747591235,"id":"chatcmpl-BYcbLSepxSXIxgUX2WZCFZrjqjp0l","model":"gpt-4o-2024-11-20","system_fingerprint":"fp_ee1d74bde0"}
data: {"choices":[{"index":0,"delta":{"content":null,"tool_calls":[{"function":{"arguments":"San"},"index":0}]}}],"created":1747591235,"id":"chatcmpl-BYcbLSepxSXIxgUX2WZCFZrjqjp0l","model":"gpt-4o-2024-11-20","system_fingerprint":"fp_ee1d74bde0"}
data: {"choices":[{"index":0,"delta":{"content":null,"tool_calls":[{"function":{"arguments":" Francisco"},"index":0}]}}],"created":1747591235,"id":"chatcmpl-BYcbLSepxSXIxgUX2WZCFZrjqjp0l","model":"gpt-4o-2024-11-20","system_fingerprint":"fp_ee1d74bde0"}
data: {"choices":[{"index":0,"delta":{"content":null,"tool_calls":[{"function":{"arguments":"\"}"},"index":0}]}}],"created":1747591235,"id":"chatcmpl-BYcbLSepxSXIxgUX2WZCFZrjqjp0l","model":"gpt-4o-2024-11-20","system_fingerprint":"fp_ee1d74bde0"}
data: {"choices":[{"finish_reason":"tool_calls","index":0,"delta":{"content":null}}],"created":1747591235,"id":"chatcmpl-BYcbLSepxSXIxgUX2WZCFZrjqjp0l","usage":{"completion_tokens":16,"completion_tokens_details":{"accepted_prediction_tokens":0,"rejected_prediction_tokens":0},"prompt_tokens":73,"prompt_tokens_details":{"cached_tokens":0},"total_tokens":89},"model":"gpt-4o-2024-11-20","system_fingerprint":"fp_ee1d74bde0"}
data: [DONE]
`
	collector := NewOAIStreamCollector()
	scanner := bufio.NewScanner(strings.NewReader(stream))
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		payload := strings.TrimPrefix(line, "data: ")
		if payload == "[DONE]" {
			break
		}
		var chunk OAIStreamChunk
		if err := json.Unmarshal([]byte(payload), &chunk); err != nil {
			t.Fatalf("failed to unmarshal chunk: %v", err)
		}
		collector.AddChunk(&chunk)
	}
	resp := collector.BuildResponse()
	if len(resp.Choices) != 1 {
		t.Fatalf("expected 1 choice, got %d", len(resp.Choices))
	}
	choice := resp.Choices[0]
	if choice.Message.Role != "assistant" {
		t.Errorf("expected role 'assistant', got %q", choice.Message.Role)
	}
	if len(choice.Message.ToolCalls) != 1 {
		t.Fatalf("expected 1 tool call, got %d", len(choice.Message.ToolCalls))
	}
	tc := choice.Message.ToolCalls[0]
	if tc.Function.Name != "get_weather" {
		t.Errorf("expected function name 'get_weather', got %q", tc.Function.Name)
	}
	if tc.Function.Arguments != "{\"location\":\"San Francisco\"}" {
		t.Errorf("expected arguments '{\"location\":\"San Francisco\"}', got %q", tc.Function.Arguments)
	}
	if choice.FinishReason != "tool_calls" {
		t.Errorf("expected finish_reason 'tool_calls', got %q", choice.FinishReason)
	}
}

func TestOAIStreamCollector_TextStreaming(t *testing.T) {
	stream := `
data: {"choices":[],"created":0,"id":"","prompt_filter_results":[{"content_filter_results":{"hate":{"filtered":false,"severity":"safe"},"self_harm":{"filtered":false,"severity":"safe"},"sexual":{"filtered":false,"severity":"safe"},"violence":{"filtered":false,"severity":"safe"}},"prompt_index":0}]}
data: {"choices":[{"index":0,"content_filter_offsets":{"check_offset":3458,"start_offset":3458,"end_offset":3494},"content_filter_results":{"hate":{"filtered":false,"severity":"safe"},"self_harm":{"filtered":false,"severity":"safe"},"sexual":{"filtered":false,"severity":"safe"},"violence":{"filtered":false,"severity":"safe"}},"delta":{"content":"","role":"assistant"}}],"created":1747592466,"id":"chatcmpl-BYcvCkaKJjQIM7e2j6vg08RIcY8qp","model":"gpt-4o-2024-11-20","system_fingerprint":"fp_ee1d74bde0"}
data: {"choices":[{"index":0,"content_filter_offsets":{"check_offset":3458,"start_offset":3458,"end_offset":3494},"content_filter_results":{"hate":{"filtered":false,"severity":"safe"},"self_harm":{"filtered":false,"severity":"safe"},"sexual":{"filtered":false,"severity":"safe"},"violence":{"filtered":false,"severity":"safe"}},"delta":{"content":"Hello"}}],"created":1747592466,"id":"chatcmpl-BYcvCkaKJjQIM7e2j6vg08RIcY8qp","model":"gpt-4o-2024-11-20","system_fingerprint":"fp_ee1d74bde0"}
data: {"choices":[{"index":0,"content_filter_offsets":{"check_offset":3458,"start_offset":3458,"end_offset":3494},"content_filter_results":{"hate":{"filtered":false,"severity":"safe"},"self_harm":{"filtered":false,"severity":"safe"},"sexual":{"filtered":false,"severity":"safe"},"violence":{"filtered":false,"severity":"safe"}},"delta":{"content":"!"}}],"created":1747592466,"id":"chatcmpl-BYcvCkaKJjQIM7e2j6vg08RIcY8qp","model":"gpt-4o-2024-11-20","system_fingerprint":"fp_ee1d74bde0"}
data: {"choices":[{"index":0,"content_filter_offsets":{"check_offset":3458,"start_offset":3458,"end_offset":3494},"content_filter_results":{"hate":{"filtered":false,"severity":"safe"},"self_harm":{"filtered":false,"severity":"safe"},"sexual":{"filtered":false,"severity":"safe"},"violence":{"filtered":false,"severity":"safe"}},"delta":{"content":" How"}}],"created":1747592466,"id":"chatcmpl-BYcvCkaKJjQIM7e2j6vg08RIcY8qp","model":"gpt-4o-2024-11-20","system_fingerprint":"fp_ee1d74bde0"}
data: {"choices":[{"index":0,"content_filter_offsets":{"check_offset":3458,"start_offset":3458,"end_offset":3494},"content_filter_results":{"hate":{"filtered":false,"severity":"safe"},"self_harm":{"filtered":false,"severity":"safe"},"sexual":{"filtered":false,"severity":"safe"},"violence":{"filtered":false,"severity":"safe"}},"delta":{"content":" can"}}],"created":1747592466,"id":"chatcmpl-BYcvCkaKJjQIM7e2j6vg08RIcY8qp","model":"gpt-4o-2024-11-20","system_fingerprint":"fp_ee1d74bde0"}
data: {"choices":[{"index":0,"content_filter_offsets":{"check_offset":3458,"start_offset":3458,"end_offset":3494},"content_filter_results":{"hate":{"filtered":false,"severity":"safe"},"self_harm":{"filtered":false,"severity":"safe"},"sexual":{"filtered":false,"severity":"safe"},"violence":{"filtered":false,"severity":"safe"}},"delta":{"content":" I"}}],"created":1747592466,"id":"chatcmpl-BYcvCkaKJjQIM7e2j6vg08RIcY8qp","model":"gpt-4o-2024-11-20","system_fingerprint":"fp_ee1d74bde0"}
data: {"choices":[{"index":0,"content_filter_offsets":{"check_offset":3458,"start_offset":3458,"end_offset":3494},"content_filter_results":{"hate":{"filtered":false,"severity":"safe"},"self_harm":{"filtered":false,"severity":"safe"},"sexual":{"filtered":false,"severity":"safe"},"violence":{"filtered":false,"severity":"safe"}},"delta":{"content":" assist"}}],"created":1747592466,"id":"chatcmpl-BYcvCkaKJjQIM7e2j6vg08RIcY8qp","model":"gpt-4o-2024-11-20","system_fingerprint":"fp_ee1d74bde0"}
data: {"choices":[{"index":0,"content_filter_offsets":{"check_offset":3458,"start_offset":3458,"end_offset":3494},"content_filter_results":{"hate":{"filtered":false,"severity":"safe"},"self_harm":{"filtered":false,"severity":"safe"},"sexual":{"filtered":false,"severity":"safe"},"violence":{"filtered":false,"severity":"safe"}},"delta":{"content":" you"}}],"created":1747592466,"id":"chatcmpl-BYcvCkaKJjQIM7e2j6vg08RIcY8qp","model":"gpt-4o-2024-11-20","system_fingerprint":"fp_ee1d74bde0"}
data: {"choices":[{"index":0,"content_filter_offsets":{"check_offset":3458,"start_offset":3458,"end_offset":3494},"content_filter_results":{"hate":{"filtered":false,"severity":"safe"},"self_harm":{"filtered":false,"severity":"safe"},"sexual":{"filtered":false,"severity":"safe"},"violence":{"filtered":false,"severity":"safe"}},"delta":{"content":" today"}}],"created":1747592466,"id":"chatcmpl-BYcvCkaKJjQIM7e2j6vg08RIcY8qp","model":"gpt-4o-2024-11-20","system_fingerprint":"fp_ee1d74bde0"}
data: {"choices":[{"index":0,"content_filter_offsets":{"check_offset":3458,"start_offset":3458,"end_offset":3494},"content_filter_results":{"hate":{"filtered":false,"severity":"safe"},"self_harm":{"filtered":false,"severity":"safe"},"sexual":{"filtered":false,"severity":"safe"},"violence":{"filtered":false,"severity":"safe"}},"delta":{"content":"?"}}],"created":1747592466,"id":"chatcmpl-BYcvCkaKJjQIM7e2j6vg08RIcY8qp","model":"gpt-4o-2024-11-20","system_fingerprint":"fp_ee1d74bde0"}
data: {"choices":[{"index":0,"content_filter_offsets":{"check_offset":3458,"start_offset":3458,"end_offset":3494},"content_filter_results":{"hate":{"filtered":false,"severity":"safe"},"self_harm":{"filtered":false,"severity":"safe"},"sexual":{"filtered":false,"severity":"safe"},"violence":{"filtered":false,"severity":"safe"}},"delta":{"content":" 🌍"}}],"created":1747592466,"id":"chatcmpl-BYcvCkaKJjQIM7e2j6vg08RIcY8qp","model":"gpt-4o-2024-11-20","system_fingerprint":"fp_ee1d74bde0"}
data: {"choices":[{"finish_reason":"stop","index":0,"content_filter_offsets":{"check_offset":3458,"start_offset":3458,"end_offset":3494},"content_filter_results":{"hate":{"filtered":false,"severity":"safe"},"self_harm":{"filtered":false,"severity":"safe"},"sexual":{"filtered":false,"severity":"safe"},"violence":{"filtered":false,"severity":"safe"}},"delta":{"content":null}}],"created":1747592466,"id":"chatcmpl-BYcvCkaKJjQIM7e2j6vg08RIcY8qp","usage":{"completion_tokens":13,"completion_tokens_details":{"accepted_prediction_tokens":0,"rejected_prediction_tokens":0},"prompt_tokens":1675,"prompt_tokens_details":{"cached_tokens":1536},"total_tokens":1688},"model":"gpt-4o-2024-11-20","system_fingerprint":"fp_ee1d74bde0"}
data: [DONE]
`
	collector := NewOAIStreamCollector()
	scanner := bufio.NewScanner(strings.NewReader(stream))
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		payload := strings.TrimPrefix(line, "data: ")
		if payload == "[DONE]" {
			break
		}
		var chunk OAIStreamChunk
		if err := json.Unmarshal([]byte(payload), &chunk); err != nil {
			t.Fatalf("failed to unmarshal chunk: %v", err)
		}
		collector.AddChunk(&chunk)
	}
	resp := collector.BuildResponse()
	if len(resp.Choices) != 1 {
		t.Fatalf("expected 1 choice, got %d", len(resp.Choices))
	}
	choice := resp.Choices[0]
	if choice.Message.Role != "assistant" {
		t.Errorf("expected role 'assistant', got %q", choice.Message.Role)
	}
	if got, want := derefString(choice.Message.Content), "Hello! How can I assist you today? 🌍"; got != want {
		t.Errorf("expected content %q, got %q", want, got)
	}
	if choice.FinishReason != "stop" {
		t.Errorf("expected finish_reason 'stop', got %q", choice.FinishReason)
	}
}

func derefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
