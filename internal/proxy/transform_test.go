package proxy

import (
	"reflect"
	"testing"
	"time"
)

func testConfig(t *testing.T) Config {
	t.Helper()
	config, err := NewConfig(ConfigInput{
		Upstream:               "http://127.0.0.1:11434",
		ModelAliases:           []string{"glm-5.2", "glm-5.2:cloud[1m]"},
		ModelTarget:            "glm-5.2:cloud",
		ReasoningMaps:          []string{"xhigh=max"},
		DefaultReasoningEffort: "",
		Timeout:                10 * time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	return config
}

func TestNormalizePayloadInjectsDefaultReasoningWhenMissing(t *testing.T) {
	config := testConfig(t)
	config.DefaultReasoningEffort = "max"
	payload := map[string]any{"model": "glm-5.2"}

	mutations := NormalizePayload(payload, "/v1/responses", config)

	if !reflect.DeepEqual(mutations, []string{"rewrite_model", "rewrite_reasoning"}) {
		t.Fatalf("mutations = %#v", mutations)
	}
	if payload["reasoning"].(map[string]any)["effort"] != "max" {
		t.Fatalf("reasoning effort = %v", payload["reasoning"].(map[string]any)["effort"])
	}
}

func TestNormalizePayloadDoesNotOverrideExplicitHighWithDefault(t *testing.T) {
	config := testConfig(t)
	config.DefaultReasoningEffort = "max"
	payload := map[string]any{
		"model":     "glm-5.2",
		"reasoning": map[string]any{"effort": "high"},
	}

	mutations := NormalizePayload(payload, "/v1/responses", config)

	if !reflect.DeepEqual(mutations, []string{"rewrite_model"}) {
		t.Fatalf("mutations = %#v", mutations)
	}
	if payload["reasoning"].(map[string]any)["effort"] != "high" {
		t.Fatalf("reasoning effort = %v", payload["reasoning"].(map[string]any)["effort"])
	}
}

func TestNormalizePayloadRewritesCodexModelAndXhighReasoning(t *testing.T) {
	payload := map[string]any{
		"model":     "glm-5.2",
		"reasoning": map[string]any{"effort": "xhigh"},
	}

	mutations := NormalizePayload(payload, "/v1/responses", testConfig(t))

	if !reflect.DeepEqual(mutations, []string{"rewrite_model", "rewrite_reasoning"}) {
		t.Fatalf("mutations = %#v", mutations)
	}
	if payload["model"] != "glm-5.2:cloud" {
		t.Fatalf("model = %v", payload["model"])
	}
	if payload["reasoning"].(map[string]any)["effort"] != "max" {
		t.Fatalf("reasoning effort = %v", payload["reasoning"].(map[string]any)["effort"])
	}
}

func TestNormalizePayloadKeepsHighReasoningSelectable(t *testing.T) {
	payload := map[string]any{
		"model":     "glm-5.2",
		"reasoning": map[string]any{"effort": "high"},
	}

	mutations := NormalizePayload(payload, "/v1/responses", testConfig(t))

	if !reflect.DeepEqual(mutations, []string{"rewrite_model"}) {
		t.Fatalf("mutations = %#v", mutations)
	}
	if payload["reasoning"].(map[string]any)["effort"] != "high" {
		t.Fatalf("reasoning effort = %v", payload["reasoning"].(map[string]any)["effort"])
	}
}

func TestNormalizePayloadRewritesClaudeDisplayModelWithoutReasoning(t *testing.T) {
	payload := map[string]any{"model": "glm-5.2:cloud[1m]"}

	mutations := NormalizePayload(payload, "/v1/messages", testConfig(t))

	if !reflect.DeepEqual(mutations, []string{"rewrite_model"}) {
		t.Fatalf("mutations = %#v", mutations)
	}
	if payload["model"] != "glm-5.2:cloud" {
		t.Fatalf("model = %v", payload["model"])
	}
}

func TestNormalizePayloadRewritesChatCompletionsXhighReasoning(t *testing.T) {
	payload := map[string]any{
		"model":            "glm-5.2",
		"reasoning_effort": "xhigh",
	}

	mutations := NormalizePayload(payload, "/v1/chat/completions", testConfig(t))

	if !reflect.DeepEqual(mutations, []string{"rewrite_model", "rewrite_reasoning"}) {
		t.Fatalf("mutations = %#v", mutations)
	}
	if payload["reasoning_effort"] != "max" {
		t.Fatalf("reasoning effort = %v", payload["reasoning_effort"])
	}
}
