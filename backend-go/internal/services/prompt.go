package services

import (
	"encoding/json"
	"strings"
)

// PromptEngine 提示词引擎
type PromptEngine struct{}

func NewPromptEngine() *PromptEngine {
	return &PromptEngine{}
}

// RenderPrompt 渲染提示词（替换占位符）
func (e *PromptEngine) RenderPrompt(template string, fragments interface{}, carryover interface{}, narrativeType string) string {
	result := template

	// 替换占位符
	fragmentsJSON := toJSON(fragments)
	carryoverJSON := toJSON(carryover)

	result = strings.ReplaceAll(result, "{fragments}", fragmentsJSON)
	result = strings.ReplaceAll(result, "{carryover}", carryoverJSON)
	result = strings.ReplaceAll(result, "{narrative_type}", narrativeType)

	return result
}

// BuildMessages 构建完整的LLM消息
func (e *PromptEngine) BuildMessages(systemPrompt, userPrompt string) []ChatMessage {
	return []ChatMessage{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userPrompt},
	}
}

func toJSON(v interface{}) string {
	b, err := json.Marshal(v)
	if err != nil {
		return "[]"
	}
	return string(b)
}
