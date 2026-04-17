package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// ApprovalSuggestionRequest 审批建议请求
type ApprovalSuggestionRequest struct {
	InstanceTitle string         `json:"instance_title"`
	WorkflowType  string         `json:"workflow_type"`
	StepName      string         `json:"step_name"`
	Context       map[string]any `json:"context,omitempty"`
	History       []ApprovalHistory `json:"history,omitempty"`
}

// ApprovalHistory 审批历史记录
type ApprovalHistory struct {
	ApproverID string `json:"approver_id"`
	Status     string `json:"status"`
	Comment    string `json:"comment"`
}

// LLMService LLM 服务接口
type LLMService interface {
	GenerateApprovalSuggestion(ctx context.Context, req *ApprovalSuggestionRequest) (string, error)
}

// llmService LLM 服务实现
type llmService struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

// NewLLMService 创建 LLM 服务实例
func NewLLMService(baseURL, apiKey string, timeout time.Duration) LLMService {
	return &llmService{
		baseURL: baseURL,
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

// openAIRequest OpenAI 兼容接口请求体
type openAIRequest struct {
	Model    string                   `json:"model"`
	Messages []openAIMessage          `json:"messages"`
}

// openAIMessage OpenAI 消息
type openAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// openAIResponse OpenAI 兼容接口响应体
type openAIResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

// GenerateApprovalSuggestion 生成审批建议
func (s *llmService) GenerateApprovalSuggestion(ctx context.Context, req *ApprovalSuggestionRequest) (string, error) {
	// 构建系统提示词
	systemPrompt := buildSystemPrompt(req)

	// 构建请求体
	reqBody := openAIRequest{
		Model: "gpt-4",
		Messages: []openAIMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: fmt.Sprintf("请为以下审批提供建议：\n标题：%s\n工作流类型：%s\n当前步骤：%s", req.InstanceTitle, req.WorkflowType, req.StepName)},
		},
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("序列化请求失败: %w", err)
	}

	// 创建 HTTP 请求
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, s.baseURL+"/v1/chat/completions", bytes.NewReader(bodyBytes))
	if err != nil {
		return "", fmt.Errorf("创建请求失败: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+s.apiKey)

	// 发送请求
	resp, err := s.httpClient.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("请求 LLM 服务失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("LLM 服务返回错误: HTTP %d, %s", resp.StatusCode, string(body))
	}

	// 解析响应
	var openAIResp openAIResponse
	if err := json.NewDecoder(resp.Body).Decode(&openAIResp); err != nil {
		return "", fmt.Errorf("解析 LLM 响应失败: %w", err)
	}

	if len(openAIResp.Choices) == 0 {
		return "", fmt.Errorf("LLM 响应中没有可用的建议")
	}

	return openAIResp.Choices[0].Message.Content, nil
}

// buildSystemPrompt 构建系统提示词
func buildSystemPrompt(req *ApprovalSuggestionRequest) string {
	prompt := "你是一个审批助手，请根据审批信息提供简洁的审批建议。"

	if req.WorkflowType != "" {
		prompt += fmt.Sprintf("\n工作流类型：%s", req.WorkflowType)
	}

	if len(req.History) > 0 {
		prompt += "\n\n历史审批记录："
		for _, h := range req.History {
			prompt += fmt.Sprintf("\n- 审批人 %s：%s（%s）", h.ApproverID, h.Status, h.Comment)
		}
	}

	if len(req.Context) > 0 {
		prompt += "\n\n审批上下文数据："
		for k, v := range req.Context {
			prompt += fmt.Sprintf("\n- %s: %v", k, v)
		}
	}

	return prompt
}
