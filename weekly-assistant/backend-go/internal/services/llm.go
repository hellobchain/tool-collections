package services

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/hellobchain/weekly-assistant/internal/config"
	"github.com/hellobchain/weekly-assistant/internal/constants"
)

type LLMService struct {
	client  *http.Client
	apiKey  string
	baseURL string
	model   string
}

type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatRequest struct {
	Model       string        `json:"model"`
	Messages    []ChatMessage `json:"messages"`
	Temperature float64       `json:"temperature"`
	MaxTokens   int           `json:"max_tokens"`
	Stream      bool          `json:"stream"`
}

type ChatChoice struct {
	Message struct {
		Content string `json:"content"`
	} `json:"message"`
}

type ChatResponse struct {
	Choices []ChatChoice `json:"choices"`
	Error   *struct {
		Message string `json:"message"`
	} `json:"error"`
}

func NewLLMService() *LLMService {
	return &LLMService{
		client: &http.Client{
			Timeout: 60 * time.Second,
		},
		apiKey:  config.AppConfig.DeepSeekAPIKey,
		baseURL: config.AppConfig.DeepSeekBaseURL,
		model:   config.AppConfig.DeepSeekModel,
	}
}

// newStreamClient 流式请求专用客户端，不设超时（由调用方控制）
func newStreamClient() *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			ResponseHeaderTimeout: 30 * time.Second,
		},
	}
}

// GenerateDraftStream 支持自定义提示词 流式生成周报草稿，通过ch逐块发送内容
func (s *LLMService) GenerateDraftStream(systemPrompt, userPrompt string, ch chan<- string) {
	defer close(ch)
	reqBody := ChatRequest{
		Model: s.model,
		Messages: []ChatMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
		Temperature: 0.3,
		MaxTokens:   config.AppConfig.DeepSeekMaxTokens,
		Stream:      true,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		ch <- s.generateFallback(nil, nil)
		return
	}

	log.Println("LLM流式请求:", string(jsonData))

	req, err := http.NewRequest("POST", s.baseURL+"/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		ch <- s.generateFallback(nil, nil)
		return
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.apiKey)

	resp, err := newStreamClient().Do(req)
	if err != nil {
		log.Println("stream request error:", err)
		ch <- s.generateFallback(nil, nil)
		return
	}
	defer resp.Body.Close()

	reader := bufio.NewReader(resp.Body)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				log.Println("stream EOF normal exit")
				break
			} else {
				log.Println("read error:", err)
				errMsg := fmt.Sprintf("LLM调用失败: %v", err)
				ch <- errMsg
				break
			}
		}
		if line == "" || !strings.HasPrefix(line, "data: ") {
			continue
		}
		log.Println("stream:", line)
		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			log.Println("done")
			break
		}
		var streamResp struct {
			Choices []struct {
				Delta struct {
					Content string `json:"content"`
				} `json:"delta"`
			} `json:"choices"`
		}
		if err := json.Unmarshal([]byte(data), &streamResp); err != nil {
			continue
		}
		if len(streamResp.Choices) > 0 {
			content := streamResp.Choices[0].Delta.Content
			contentJsonByte, err := json.Marshal(content)
			if err != nil {
				log.Println("json marshal error:", err)
				ch <- content
			} else {
				ch <- string(contentJsonByte)
			}
		}
	}
}

// GenerateDraftWithPrompt 支持自定义提示词
func (s *LLMService) GenerateDraftWithPrompt(systemPrompt, userPrompt string) (string, error) {
	reqBody := ChatRequest{
		Model: s.model,
		Messages: []ChatMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
		Temperature: 0.3,
		MaxTokens:   config.AppConfig.DeepSeekMaxTokens,
		Stream:      false,
	}

	return s.doChatRequest(reqBody)
}

// 抽取公共方法
func (s *LLMService) doChatRequest(reqBody ChatRequest) (string, error) {
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		log.Printf("LLM请求序列化失败: %v", err)
		return "", err
	}

	log.Println("LLM请求:", string(jsonData))

	req, err := http.NewRequest("POST", s.baseURL+"/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		log.Printf("创建LLM请求失败: %v", err)
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.apiKey)

	resp, err := s.client.Do(req)
	if err != nil {
		log.Printf("LLM调用失败: %v", err)
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("读取LLM响应失败: %v", err)
		return "", err
	}

	var chatResp ChatResponse
	if err := json.Unmarshal(body, &chatResp); err != nil {
		log.Printf("解析LLM响应失败: %v", err)
		return "", err
	}

	if chatResp.Error != nil {
		log.Printf("LLM返回错误: %s", chatResp.Error.Message)
		return "", fmt.Errorf("LLM API error: %s", chatResp.Error.Message)
	}

	if len(chatResp.Choices) > 0 {
		return chatResp.Choices[0].Message.Content, nil
	}

	return "", fmt.Errorf("LLM返回结果为空")
}

// generateFallback 降级方案
func (s *LLMService) generateFallback(fragments []map[string]interface{}, carryover []map[string]interface{}) string {
	lines := []string{"### 本周关键产出"}

	count := 0
	for _, f := range fragments {
		if count >= 5 {
			break
		}
		if content, ok := f["content"].(string); ok {
			lines = append(lines, fmt.Sprintf("- %s", content))
			count++
		}
	}

	if len(carryover) > 0 {
		if content, ok := carryover[0]["content"].(string); ok {
			carryoverLine := fmt.Sprintf("承接上周遗留：%s", content)
			// 在 "### 本周关键产出" 之后插入继承行
			result := []string{lines[0], carryoverLine}
			result = append(result, lines[1:]...)
			lines = result
		}
	}

	lines = append(lines, "", "### 常规事项与协作")
	lines = append(lines, "- （暂无详细记录，请手动补充）")
	lines = append(lines, "", "### 风险与下周计划")
	lines = append(lines, "- （请手动补充）")

	var result strings.Builder
	for _, line := range lines {
		result.WriteString(line + "\n")
	}
	return result.String()
}

// ExtractNextWeekPlan 调用 LLM 从周报内容中提取下周计划（最多3条）
func (s *LLMService) ExtractNextWeekPlan(weekStart, content string) []map[string]interface{} {
	items := []map[string]interface{}{}
	systemPrompt := constants.DefaultNextWeekPlanSystemPrompt
	userPrompt := fmt.Sprintf(constants.DefaultNextWeekPlanUserPrompt, content)
	reqBody := ChatRequest{
		Model: s.model,
		Messages: []ChatMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
		Temperature: 0.1,
		MaxTokens:   config.AppConfig.DeepSeekMaxTokens,
		Stream:      false,
	}

	result, err := s.doChatRequest(reqBody)
	if err != nil {
		log.Printf("LLM请求失败: %v", err)
		return items
	}

	// 尝试解析JSON数组
	result = strings.TrimSpace(result)
	// 去掉可能的markdown代码块标记
	result = strings.TrimPrefix(result, "```json")
	result = strings.TrimPrefix(result, "```")
	result = strings.TrimSuffix(result, "```")
	result = strings.TrimSpace(result)

	var parsed []map[string]interface{}
	if err := json.Unmarshal([]byte(result), &parsed); err != nil {
		log.Printf("ExtractNextWeekPlan 解析LLM返回失败: %v, 原始返回: %s", err, result)
		return items
	}

	seen := make(map[string]bool)
	for _, item := range parsed {
		if len(items) >= 3 {
			break
		}
		content, ok := item["content"].(string)
		if !ok || content == "" {
			continue
		}
		cleaned := cleanString(content)
		if cleaned == "" || seen[cleaned] {
			continue
		}
		seen[cleaned] = true
		items = append(items, map[string]interface{}{
			"content":  cleaned,
			"id":       uuid.NewString(),
			"fromWeek": weekStart,
		})
	}
	return items
}

func cleanString(s string) string {
	// 去掉"可能""大概"等弱化词（整词匹配，不拆字）
	re := regexp.MustCompile(`(可能|大概|似乎)`)
	result := re.ReplaceAllString(s, "")
	// 去掉首尾空白
	result = regexp.MustCompile(`^\s+|\s+$`).ReplaceAllString(result, "")
	return result
}

// GenerateSummary 根据多篇周报内容生成阶段性总结
func (s *LLMService) GenerateSummary(periodLabel string, reports []map[string]string) (string, error) {
	var sb strings.Builder
	for i, r := range reports {
		sb.WriteString(fmt.Sprintf("=== 第%d篇：%s ===\n", i+1, r["week_start"]))
		sb.WriteString(r["content"])
		sb.WriteString("\n\n")
	}
	systemPrompt := fmt.Sprintf(constants.DefaultSummarySystemPrompt, periodLabel)
	userPrompt := fmt.Sprintf(constants.DefaultSummaryUserPrompt, periodLabel, sb.String(), periodLabel)
	reqBody := ChatRequest{
		Model: s.model,
		Messages: []ChatMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
		Temperature: 0.3,
		MaxTokens:   config.AppConfig.DeepSeekMaxTokens,
		Stream:      false,
	}

	return s.doChatRequest(reqBody)
}
