package aitools

import (
	"encoding/json"
	"fmt"
	"nofrillz/internal/config"
	"regexp"
	"sort"
	"strings"
)

// Option IDs are stable user preferences. Concrete provider model names may be
// upgraded independently and are recorded on each generated variant.
type Option struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Provider  string `json:"provider"`
	Model     string `json:"model"`
	Available bool   `json:"available"`
	Tools     Tools  `json:"-"`
}
type Registry struct{ options map[string]Option }

func NewRegistry(options []Option) (*Registry, error) {
	r := &Registry{options: map[string]Option{}}
	for _, o := range options {
		if !regexp.MustCompile(`^[a-z][a-z0-9_-]{0,63}$`).MatchString(o.ID) || o.Model == "" {
			return nil, fmt.Errorf("invalid model option configuration")
		}
		if _, ok := r.options[o.ID]; ok {
			return nil, fmt.Errorf("duplicate model option %s", o.ID)
		}
		o.Available = o.Tools != nil
		r.options[o.ID] = o
	}
	return r, nil
}
func (r *Registry) Options() []Option {
	items := make([]Option, 0, len(r.options))
	for _, o := range r.options {
		items = append(items, o)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].ID < items[j].ID })
	return items
}
func (r *Registry) Get(id string) (Option, bool) { o, ok := r.options[id]; return o, ok }
func NewRegistryFromConfig(c *config.Config) (*Registry, error) {
	base := c.AIToolsConfig()
	defaults := []Option{{ID: "openai", Name: "OpenAI", Provider: "openai", Model: base.OpenAI.Model},
		{ID: "claude", Name: "Claude", Provider: "anthropic", Model: valueOr(c.Viper.GetString("ai_tools.anthropic.model"), "configure-model")},
		{ID: "grok", Name: "Grok", Provider: "xai", Model: valueOr(c.Viper.GetString("ai_tools.xai.model"), "configure-model")}}
	if strings.EqualFold(base.Provider, "mock") {
		defaults = append(defaults, Option{ID: "mock", Name: "Mock fixtures (development only)", Provider: "mock", Model: "mock"})
	}
	// Entries extend the catalog or override an existing stable ID. Credentials
	// remain in provider configuration and are never part of this public catalog.
	if raw := c.Viper.GetString("ai_tools.model_options_json"); raw != "" {
		var configured []Option
		if err := json.Unmarshal([]byte(raw), &configured); err != nil {
			return nil, fmt.Errorf("invalid ai_tools.model_options_json")
		}
		seen := map[string]bool{}
		for _, option := range configured {
			if seen[option.ID] {
				return nil, fmt.Errorf("duplicate configured model option %s", option.ID)
			}
			seen[option.ID] = true
			found := false
			for i := range defaults {
				if defaults[i].ID == option.ID {
					defaults[i] = option
					found = true
					break
				}
			}
			if !found {
				defaults = append(defaults, option)
			}
		}
	}
	for i := range defaults {
		o := &defaults[i]
		var err error
		switch o.Provider {
		case "openai":
			cfg := base.OpenAI
			cfg.Model = o.Model
			if cfg.APIKey != "" {
				o.Tools, err = NewOpenAI(&cfg)
			}
		case "anthropic":
			key := c.Viper.GetString("ai_tools.anthropic.api_key")
			if key != "" && o.Model != "configure-model" {
				o.Tools = NewAnthropic(key, valueOr(c.Viper.GetString("ai_tools.anthropic.base_url"), "https://api.anthropic.com/v1"), o.Model)
			}
		case "xai":
			key := c.Viper.GetString("ai_tools.xai.api_key")
			if key != "" && o.Model != "configure-model" {
				o.Tools = NewChatCompletions(key, valueOr(c.Viper.GetString("ai_tools.xai.base_url"), "https://api.x.ai/v1"), o.Model)
			}
		case "mock":
			if strings.EqualFold(base.Provider, "mock") {
				o.Tools = NewMockTools()
			}
		default:
			return nil, fmt.Errorf("unsupported provider for model option %s", o.ID)
		}
		if err != nil {
			return nil, err
		}
	}
	return NewRegistry(defaults)
}
func valueOr(s, fallback string) string {
	if strings.TrimSpace(s) == "" {
		return fallback
	}
	return s
}
