package proxy

import (
	"fmt"
	"net/url"
	"strings"
	"time"
)

type ConfigInput struct {
	Upstream               string
	ModelAliases           []string
	ModelTarget            string
	ReasoningMaps          []string
	DefaultReasoningEffort string
	Timeout                time.Duration
}

type Config struct {
	UpstreamURL            *url.URL
	ModelAliases           map[string]struct{}
	ModelTarget            string
	ReasoningMap           map[string]string
	DefaultReasoningEffort string
	Timeout                time.Duration
}

func NewConfig(input ConfigInput) (Config, error) {
	upstream, err := url.Parse(input.Upstream)
	if err != nil || upstream.Scheme == "" || upstream.Host == "" {
		return Config{}, fmt.Errorf("upstream must be an http(s) URL with a host")
	}
	if upstream.Scheme != "http" && upstream.Scheme != "https" {
		return Config{}, fmt.Errorf("unsupported upstream scheme %q", upstream.Scheme)
	}
	if input.ModelTarget == "" {
		return Config{}, fmt.Errorf("model target must not be empty")
	}

	aliases := make(map[string]struct{}, len(input.ModelAliases))
	for _, alias := range input.ModelAliases {
		if alias == "" {
			return Config{}, fmt.Errorf("model aliases must not contain empty values")
		}
		aliases[alias] = struct{}{}
	}

	reasoningMap, err := ParseReasoningMaps(input.ReasoningMaps)
	if err != nil {
		return Config{}, err
	}
	timeout := input.Timeout
	if timeout <= 0 {
		timeout = 3000 * time.Second
	}

	return Config{
		UpstreamURL:            upstream,
		ModelAliases:           aliases,
		ModelTarget:            input.ModelTarget,
		ReasoningMap:           reasoningMap,
		DefaultReasoningEffort: strings.TrimSpace(input.DefaultReasoningEffort),
		Timeout:                timeout,
	}, nil
}

func ParseReasoningMaps(values []string) (map[string]string, error) {
	result := map[string]string{}
	for _, value := range values {
		source, target, ok := strings.Cut(value, "=")
		source = strings.TrimSpace(source)
		target = strings.TrimSpace(target)
		if !ok || source == "" || target == "" {
			return nil, fmt.Errorf("invalid reasoning map %q; expected FROM=TO", value)
		}
		result[source] = target
	}
	return result, nil
}
