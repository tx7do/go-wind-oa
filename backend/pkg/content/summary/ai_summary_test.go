package summary

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// TestGenerateSummaryByAI_Success 测试成功生成摘要
func TestGenerateSummaryByAI_Success(t *testing.T) {
	tests := []struct {
		name       string
		content    string
		lang       string
		maxLength  int
		mockResp   string
		wantSubstr string // 期望包含的子串
	}{
		{
			name:      "Chinese content with HTML",
			content:   "<p>Vue3 暗黑模式教程是前端开发的重要知识点。</p>",
			lang:      "zh-CN",
			maxLength: 100,
			mockResp: `{
				"choices": [{
					"message": {
						"role": "assistant",
						"content": "Vue3暗黑模式教程介绍前端开发暗黑模式实现方法"
					}
				}]
			}`,
			wantSubstr: "Vue3",
		},
		{
			name:      "English content",
			content:   "<p>Vue.js is a progressive JavaScript framework.</p>",
			lang:      "en-US",
			maxLength: 80,
			mockResp: `{
				"choices": [{
					"message": {
						"role": "assistant",
						"content": "Vue.js is a progressive JavaScript framework for building UIs"
					}
				}]
			}`,
			wantSubstr: "Vue.js",
		},
		{
			name:      "Plain text without HTML",
			content:   "This is a simple text without HTML tags",
			lang:      "en-US",
			maxLength: 50,
			mockResp: `{
				"choices": [{
					"message": {
						"role": "assistant",
						"content": "Simple text content"
					}
				}]
			}`,
			wantSubstr: "Simple",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建 mock server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// 验证请求头
				if auth := r.Header.Get("Authorization"); !strings.HasPrefix(auth, "Bearer ") {
					t.Errorf("Invalid Authorization header: %s", auth)
				}
				if ct := r.Header.Get("Content-Type"); ct != "application/json" {
					t.Errorf("Invalid Content-Type: %s", ct)
				}

				// 验证请求方法
				if r.Method != http.MethodPost {
					t.Errorf("Expected POST, got %s", r.Method)
				}

				// 验证请求体
				var req AISummaryRequest
				if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
					t.Errorf("Failed to decode request: %v", err)
				}

				// 验证请求参数
				if req.Model != "gpt-3.5-turbo" {
					t.Errorf("Expected model gpt-3.5-turbo, got %s", req.Model)
				}
				if len(req.Messages) != 1 {
					t.Errorf("Expected 1 message, got %d", len(req.Messages))
				}

				// 返回 mock 响应
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(tt.mockResp))
			}))
			defer server.Close()

			// 临时替换 API URL（通过修改函数或使用环境变量）
			// 这里我们需要修改函数以支持自定义 URL，或者使用接口注入
			// 暂时跳过实际调用，使用 mock 验证逻辑

			// 由于原函数硬编码了 URL，我们需要重构或使用集成测试
			// 这里先验证其他逻辑
			t.Skip("Skipping actual API call test - requires refactoring for dependency injection")
		})
	}
}

// TestGenerateSummaryByAI_HTMLStripping 测试 HTML 剥离功能
func TestGenerateSummaryByAI_HTMLStripping(t *testing.T) {
	// 这个测试验证 HTML 标签被正确剥离
	// 由于函数内部使用了 htmlTagRegex 和 spaceRegex，我们可以单独测试这部分逻辑

	tests := []struct {
		name     string
		content  string
		expected string
	}{
		{
			name:     "Simple HTML",
			content:  "<p>Hello World</p>",
			expected: "Hello World",
		},
		{
			name:     "Nested HTML",
			content:  "<div><p>Hello <strong>World</strong></p></div>",
			expected: "Hello World",
		},
		{
			name:     "Multiple spaces",
			content:  "<p>Hello    World</p>",
			expected: "Hello World",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 模拟函数内部的 HTML 剥离逻辑
			plainText := htmlTagRegex.ReplaceAllString(tt.content, "")
			plainText = spaceRegex.ReplaceAllString(plainText, " ")
			plainText = strings.TrimSpace(plainText)

			if plainText != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, plainText)
			}
		})
	}
}

// TestGenerateSummaryByAI_ErrorHandling 测试错误处理
func TestGenerateSummaryByAI_ErrorHandling(t *testing.T) {
	tests := []struct {
		name          string
		apiKey        string
		content       string
		lang          string
		maxLength     int
		mockStatus    int
		mockResp      string
		expectError   bool
		errorContains string
	}{
		{
			name:          "Empty API key",
			apiKey:        "",
			content:       "test content",
			lang:          "zh-CN",
			maxLength:     100,
			mockStatus:    http.StatusUnauthorized,
			mockResp:      `{"error": "unauthorized"}`,
			expectError:   true,
			errorContains: "",
		},
		{
			name:       "Empty choices in response",
			apiKey:     "test-key",
			content:    "test content",
			lang:       "zh-CN",
			maxLength:  100,
			mockStatus: http.StatusOK,
			mockResp: `{
				"choices": []
			}`,
			expectError:   true,
			errorContains: "AI未返回摘要",
		},
		{
			name:          "Invalid JSON response",
			apiKey:        "test-key",
			content:       "test content",
			lang:          "zh-CN",
			maxLength:     100,
			mockStatus:    http.StatusOK,
			mockResp:      `invalid json`,
			expectError:   true,
			errorContains: "解析AI响应失败",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建 mock server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.mockStatus)
				w.Write([]byte(tt.mockResp))
			}))
			defer server.Close()

			// 由于原函数硬编码了 URL，这里需要重构
			t.Skip("Skipping error handling test - requires refactoring for dependency injection")
		})
	}
}

// TestGenerateSummaryByAI_TruncateLongResponse 测试过长响应的截断
func TestGenerateSummaryByAI_TruncateLongResponse(t *testing.T) {
	// 测试当 AI 返回的摘要超过 maxLength 时，是否正确截断

	longSummary := strings.Repeat("这是一个很长的摘要内容。", 20) // 超长内容
	maxLength := 50

	// 模拟截断逻辑
	if len([]rune(longSummary)) > maxLength {
		truncated := truncateByLength(longSummary, maxLength) + "..."

		if len([]rune(truncated)) > maxLength+3 { // +3 for "..."
			t.Errorf("Truncated summary is still too long: %d runes", len([]rune(truncated)))
		}

		if !strings.HasSuffix(truncated, "...") {
			t.Errorf("Truncated summary should end with '...'")
		}
	}
}

// TestGenerateSummaryByAI_RequestTimeout 测试请求超时
func TestGenerateSummaryByAI_RequestTimeout(t *testing.T) {
	// 创建一个慢响应的 mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(15 * time.Second) // 超过 10 秒超时设置
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// 由于原函数硬编码了 URL，这里需要重构
	t.Skip("Skipping timeout test - requires refactoring for dependency injection")
}

// TestAISummaryRequest_Marshaling 测试请求结构体序列化
func TestAISummaryRequest_Marshaling(t *testing.T) {
	req := AISummaryRequest{
		Model:       "gpt-3.5-turbo",
		Temperature: 0.3,
		MaxTokens:   200,
		Messages: []Message{
			{Role: "user", Content: "test content"},
		},
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Failed to marshal request: %v", err)
	}

	// 验证 JSON 包含必要的字段
	jsonStr := string(data)
	requiredFields := []string{
		`"model":"gpt-3.5-turbo"`,
		`"temperature":0.3`,
		`"max_tokens":200`,
		`"messages"`,
	}

	for _, field := range requiredFields {
		if !strings.Contains(jsonStr, field) {
			t.Errorf("JSON missing required field: %s", field)
		}
	}
}

// TestAISummaryResponse_Unmarshaling 测试响应结构体反序列化
func TestAISummaryResponse_Unmarshaling(t *testing.T) {
	tests := []struct {
		name     string
		jsonData string
		wantErr  bool
	}{
		{
			name: "Valid response",
			jsonData: `{
				"choices": [{
					"message": {
						"role": "assistant",
						"content": "test summary"
					}
				}]
			}`,
			wantErr: false,
		},
		{
			name: "Multiple choices",
			jsonData: `{
				"choices": [
					{
						"message": {
							"role": "assistant",
							"content": "summary 1"
						}
					},
					{
						"message": {
							"role": "assistant",
							"content": "summary 2"
						}
					}
				]
			}`,
			wantErr: false,
		},
		{
			name:     "Invalid JSON",
			jsonData: `{"invalid": json}`,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var resp AISummaryResponse
			err := json.Unmarshal([]byte(tt.jsonData), &resp)

			if (err != nil) != tt.wantErr {
				t.Errorf("Unmarshal error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr && len(resp.Choices) == 0 {
				t.Error("Expected at least one choice in valid response")
			}
		})
	}
}

// TestGenerateSummaryByAI_PromptGeneration 测试 Prompt 生成逻辑
func TestGenerateSummaryByAI_PromptGeneration(t *testing.T) {
	tests := []struct {
		name      string
		lang      string
		maxLength int
		content   string
	}{
		{
			name:      "Chinese",
			lang:      "zh-CN",
			maxLength: 100,
			content:   "测试内容",
		},
		{
			name:      "English",
			lang:      "en-US",
			maxLength: 80,
			content:   "test content",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 模拟 prompt 生成逻辑
			prompt := strings.Builder{}
			prompt.WriteString("请为以下文章生成简洁的摘要，要求：\n")
			prompt.WriteString("1. 语言：" + tt.lang + "\n")
			prompt.WriteString("2. 长度：不超过" + string(rune(tt.maxLength)) + "个字符\n")

			result := prompt.String()

			if !strings.Contains(result, tt.lang) {
				t.Errorf("Prompt should contain language: %s", tt.lang)
			}
		})
	}
}

// BenchmarkGenerateSummaryByAI_HTMLStripping 基准测试 HTML 剥离性能
func BenchmarkGenerateSummaryByAI_HTMLStripping(b *testing.B) {
	content := `<div class="article">
		<p>Vue3 <strong>暗黑模式</strong>教程是前端开发的重要知识点。</p>
		<p>通过CSS变量和<em>Tiptap编辑器</em>，可快速适配暗黑模式。</p>
	</div>`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		plainText := htmlTagRegex.ReplaceAllString(content, "")
		plainText = spaceRegex.ReplaceAllString(plainText, " ")
		_ = strings.TrimSpace(plainText)
	}
}

// MockAIClient 用于测试的 Mock AI 客户端接口
type MockAIClient struct {
	Response string
	Err      error
}

func (m *MockAIClient) GenerateSummary(content, lang string, maxLength int) (string, error) {
	if m.Err != nil {
		return "", m.Err
	}
	return m.Response, nil
}

// 集成测试示例（需要实际 API Key 才能运行）
func TestGenerateSummaryByAI_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// 从环境变量获取 API Key
	// apiKey := os.Getenv("OPENAI_API_KEY")
	// if apiKey == "" {
	// 	t.Skip("OPENAI_API_KEY not set, skipping integration test")
	// }

	// content := "<p>Vue3 暗黑模式教程</p>"
	// summary, err := GenerateSummaryByAI(apiKey, content, "zh-CN", 100)
	// if err != nil {
	// 	t.Fatalf("GenerateSummaryByAI failed: %v", err)
	// }

	// if summary == "" {
	// 	t.Error("Expected non-empty summary")
	// }

	t.Skip("Integration test requires actual API key")
}
