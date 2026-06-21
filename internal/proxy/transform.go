package proxy

const (
	mutationRewriteModel     = "rewrite_model"
	mutationRewriteReasoning = "rewrite_reasoning"
)

var reasoningPaths = map[string]struct{}{
	"/v1/responses":        {},
	"/responses":           {},
	"/v1/chat/completions": {},
	"/chat/completions":    {},
}

func NormalizePayload(payload map[string]any, path string, config Config) []string {
	mutations := []string{}
	model, _ := payload["model"].(string)
	_, isAlias := config.ModelAliases[model]

	if isAlias {
		// Claude Code uses [1m] for local context budgeting and Codex uses the
		// catalog slug. Ollama Cloud receives the registered model name.
		payload["model"] = config.ModelTarget
		mutations = append(mutations, mutationRewriteModel)
	}

	if _, ok := reasoningPaths[path]; ok && (isAlias || payload["model"] == config.ModelTarget) {
		if InjectDefaultReasoning(payload, path, config.DefaultReasoningEffort) {
			mutations = append(mutations, mutationRewriteReasoning)
		}
		if RewriteReasoning(payload, path, config.ReasoningMap) {
			if !hasMutation(mutations, mutationRewriteReasoning) {
				mutations = append(mutations, mutationRewriteReasoning)
			}
		}
	}

	return mutations
}

func hasMutation(mutations []string, value string) bool {
	for _, mutation := range mutations {
		if mutation == value {
			return true
		}
	}
	return false
}

func InjectDefaultReasoning(payload map[string]any, path string, effort string) bool {
	if effort == "" {
		return false
	}
	switch path {
	case "/v1/responses", "/responses":
		if reasoning, ok := payload["reasoning"].(map[string]any); ok {
			_, hasEffort := reasoning["effort"].(string)
			if hasEffort {
				return false
			}
			reasoning["effort"] = effort
			return true
		}
		payload["reasoning"] = map[string]any{"effort": effort}
		return true
	default:
		if _, ok := payload["reasoning_effort"].(string); ok {
			return false
		}
		payload["reasoning_effort"] = effort
		return true
	}
}

func RewriteReasoning(payload map[string]any, path string, reasoningMap map[string]string) bool {
	switch path {
	case "/v1/responses", "/responses":
		reasoning, ok := payload["reasoning"].(map[string]any)
		if !ok {
			return false
		}
		effort, ok := reasoning["effort"].(string)
		if !ok {
			return false
		}
		mapped, ok := reasoningMap[effort]
		if !ok || mapped == effort {
			return false
		}
		reasoning["effort"] = mapped
		return true
	default:
		effort, ok := payload["reasoning_effort"].(string)
		if !ok {
			return false
		}
		mapped, ok := reasoningMap[effort]
		if !ok || mapped == effort {
			return false
		}
		payload["reasoning_effort"] = mapped
		return true
	}
}
