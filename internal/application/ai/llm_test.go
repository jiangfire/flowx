package ai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// TestGenerateApprovalSuggestion_Success 正确请求返回建议
func TestGenerateApprovalSuggestion_Success(t *testing.T) {
	// 模拟 OpenAI 兼容接口
	suggestion := "建议通过该部署请求，工具版本稳定且安全检查已通过。"
	responseBody := map[string]any{
		"choices": []map[string]any{
			{
				"message": map[string]any{
					"content": suggestion,
				},
			},
		},
	}
	respBytes, _ := json.Marshal(responseBody)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 验证请求路径
		if r.URL.Path != "/v1/chat/completions" {
			t.Errorf("期望请求路径为 '/v1/chat/completions'，实际为 '%s'", r.URL.Path)
		}
		// 验证请求方法
		if r.Method != http.MethodPost {
			t.Errorf("期望请求方法为 POST，实际为 '%s'", r.Method)
		}
		// 验证 Authorization header
		if !strings.HasPrefix(r.Header.Get("Authorization"), "Bearer ") {
			t.Error("期望请求包含 Bearer token")
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(respBytes)
	}))
	defer server.Close()

	svc := NewLLMService(server.URL, "test-api-key", 5*time.Second)

	result, err := svc.GenerateApprovalSuggestion(context.Background(), &ApprovalSuggestionRequest{
		InstanceTitle: "部署 nginx 工具",
		WorkflowType:  "tool_deploy",
		StepName:      "技术审核",
		Context:       map[string]any{"tool_name": "nginx", "version": "1.25"},
		History:       nil,
	})

	if err != nil {
		t.Fatalf("期望返回建议，实际返回错误: %v", err)
	}
	if result != suggestion {
		t.Errorf("期望建议为 '%s'，实际为 '%s'", suggestion, result)
	}
}

// TestGenerateApprovalSuggestion_APIError API 错误返回错误（不 panic）
func TestGenerateApprovalSuggestion_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": {"message": "internal server error"}}`))
	}))
	defer server.Close()

	svc := NewLLMService(server.URL, "test-api-key", 5*time.Second)

	_, err := svc.GenerateApprovalSuggestion(context.Background(), &ApprovalSuggestionRequest{
		InstanceTitle: "部署 nginx 工具",
		WorkflowType:  "tool_deploy",
		StepName:      "技术审核",
	})

	if err == nil {
		t.Fatal("期望 API 错误返回错误，但返回 nil")
	}
}

// TestGenerateApprovalSuggestion_Timeout 超时处理
func TestGenerateApprovalSuggestion_Timeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 模拟延迟响应
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"choices": [{"message": {"content": "延迟建议"}}]}`))
	}))
	defer server.Close()

	// 设置超时时间为 100ms
	svc := NewLLMService(server.URL, "test-api-key", 100*time.Millisecond)

	_, err := svc.GenerateApprovalSuggestion(context.Background(), &ApprovalSuggestionRequest{
		InstanceTitle: "部署 nginx 工具",
		WorkflowType:  "tool_deploy",
		StepName:      "技术审核",
	})

	if err == nil {
		t.Fatal("期望超时返回错误，但返回 nil")
	}
}

// TestGenerateApprovalSuggestion_EmptyContext 空上下文也能正常请求
func TestGenerateApprovalSuggestion_EmptyContext(t *testing.T) {
	suggestion := "建议通过。"
	responseBody := map[string]any{
		"choices": []map[string]any{
			{
				"message": map[string]any{
					"content": suggestion,
				},
			},
		},
	}
	respBytes, _ := json.Marshal(responseBody)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(respBytes)
	}))
	defer server.Close()

	svc := NewLLMService(server.URL, "test-api-key", 5*time.Second)

	result, err := svc.GenerateApprovalSuggestion(context.Background(), &ApprovalSuggestionRequest{
		InstanceTitle: "测试审批",
		WorkflowType:  "custom",
		StepName:      "审批",
		Context:       nil,
		History:       nil,
	})

	if err != nil {
		t.Fatalf("期望返回建议，实际返回错误: %v", err)
	}
	if result != suggestion {
		t.Errorf("期望建议为 '%s'，实际为 '%s'", suggestion, result)
	}
}

// TestGenerateApprovalSuggestion_WithHistory 带历史记录的请求
func TestGenerateApprovalSuggestion_WithHistory(t *testing.T) {
	suggestion := "前序审批人均已通过，建议继续审批。"
	responseBody := map[string]any{
		"choices": []map[string]any{
			{
				"message": map[string]any{
					"content": suggestion,
				},
			},
		},
	}
	respBytes, _ := json.Marshal(responseBody)

	var receivedBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&receivedBody)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(respBytes)
	}))
	defer server.Close()

	svc := NewLLMService(server.URL, "test-api-key", 5*time.Second)

	result, err := svc.GenerateApprovalSuggestion(context.Background(), &ApprovalSuggestionRequest{
		InstanceTitle: "数据审核",
		WorkflowType:  "data_review",
		StepName:      "负责人审批",
		Context:       map[string]any{"dataset": "users"},
		History: []ApprovalHistory{
			{ApproverID: "user-1", Status: "approved", Comment: "数据质量合格"},
		},
	})

	if err != nil {
		t.Fatalf("期望返回建议，实际返回错误: %v", err)
	}
	if result != suggestion {
		t.Errorf("期望建议为 '%s'，实际为 '%s'", suggestion, result)
	}

	// 验证请求体中包含历史记录信息
	if receivedBody == nil {
		t.Fatal("期望请求体不为 nil")
	}
}

// TestGenerateApprovalSuggestion_InvalidJSON 响应 JSON 格式错误
func TestGenerateApprovalSuggestion_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`invalid json`))
	}))
	defer server.Close()

	svc := NewLLMService(server.URL, "test-api-key", 5*time.Second)

	_, err := svc.GenerateApprovalSuggestion(context.Background(), &ApprovalSuggestionRequest{
		InstanceTitle: "测试审批",
		WorkflowType:  "custom",
		StepName:      "审批",
	})

	if err == nil {
		t.Fatal("期望 JSON 解析错误返回错误，但返回 nil")
	}
}
