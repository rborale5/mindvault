package config

import (
	"encoding/json"
	"os"
	"strconv"
)

type Config struct {
	ServerPort         int
	N8NWebhookBaseURL  string
	AlexaSkillID       string
	AlexaVerifyRequests bool
	DefaultAgent       string
	IntentAgentMap     map[string]string
}

func Load() (*Config, error) {
	port, _ := strconv.Atoi(getEnv("SERVER_PORT", "8080"))

	verify, _ := strconv.ParseBool(getEnv("ALEXA_VERIFY_REQUESTS", "false"))

	intentMap := make(map[string]string)
	if raw := getEnv("INTENT_AGENT_MAP", "{}"); raw != "" {
		if err := json.Unmarshal([]byte(raw), &intentMap); err != nil {
			return nil, err
		}
	}

	return &Config{
		ServerPort:          port,
		N8NWebhookBaseURL:   getEnv("N8N_WEBHOOK_BASE_URL", "http://localhost:5678"),
		AlexaSkillID:        getEnv("ALEXA_SKILL_ID", ""),
		AlexaVerifyRequests: verify,
		DefaultAgent:        getEnv("DEFAULT_AGENT", "general-assistant"),
		IntentAgentMap:      intentMap,
	}, nil
}

// AgentForIntent returns the n8n agent path for a given Alexa intent name.
// Falls back to DefaultAgent if no mapping exists.
func (c *Config) AgentForIntent(intentName string) string {
	if agent, ok := c.IntentAgentMap[intentName]; ok {
		return agent
	}
	return c.DefaultAgent
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
