package summary

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"
)

// AISummaryRequest AI摘要请求参数
type AISummaryRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	Temperature float64   `json:"temperature"`
	MaxTokens   int       `json:"max_tokens"`
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// AISummaryResponse AI摘要响应
type AISummaryResponse struct {
	Choices []Choice `json:"choices"`
}

type Choice struct {
	Message Message `json:"message"`
}

// GenerateSummaryByAI AI生成摘要
// apiKey: AI平台API Key
// content: 原始HTML内容（或剥离标签后的纯文本）
// lang: 语言（zh-CN/en-US）
// maxLength: 摘要最大字符数
func GenerateSummaryByAI(apiKey, content, lang string, maxLength int) (string, error) {
	// 步骤1：先剥离HTML标签，减少Token消耗
	plainText := htmlTagRegex.ReplaceAllString(content, "")
	plainText = spaceRegex.ReplaceAllString(plainText, " ")
	plainText = strings.TrimSpace(plainText)

	// 步骤2：构建AI提示词（Prompt）
	prompt := fmt.Sprintf(`请为以下文章生成简洁的摘要，要求：
1. 语言：%s
2. 长度：不超过%d个字符
3. 内容：提炼核心观点，不要简单截断
4. 格式：纯文本，无markdown，无多余标点

文章内容：%s`, lang, maxLength, plainText)

	// 步骤3：调用OpenAI API
	requestBody := AISummaryRequest{
		Model:       "gpt-3.5-turbo",
		Temperature: 0.3, // 低随机性，保证摘要精准
		MaxTokens:   200, // 预留足够Token生成摘要
		Messages: []Message{
			{Role: "user", Content: prompt},
		},
	}

	// 序列化请求体
	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("序列化请求体失败：%v", err)
	}

	// 发送请求
	req, err := http.NewRequest("POST", "https://api.openai.com/v1/chat/completions", bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", fmt.Errorf("创建请求失败：%v", err)
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	// 发送请求（设置超时）
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("调用AI API失败：%v", err)
	}
	defer resp.Body.Close()

	// 解析响应
	var aiResp AISummaryResponse
	if err := json.NewDecoder(resp.Body).Decode(&aiResp); err != nil {
		return "", fmt.Errorf("解析AI响应失败：%v", err)
	}

	// 提取摘要内容
	if len(aiResp.Choices) == 0 {
		return "", fmt.Errorf("AI未返回摘要")
	}
	summary := strings.TrimSpace(aiResp.Choices[0].Message.Content)

	// 兜底：如果AI生成的摘要过长，截断
	if utf8.RuneCountInString(summary) > maxLength {
		summary = truncateByLength(summary, maxLength) + "..."
	}

	return summary, nil
}

// 使用示例
// func main() {
// 	apiKey := "your-openai-api-key"
// 	htmlContent := `<p>Vue3 暗黑模式教程是前端开发的重要知识点。通过CSS变量和Tiptap编辑器，可快速适配暗黑模式，提升用户体验。该方案已适配Vben Admin，支持多语言切换，性能比UEditor提升80%。</p>`
// 	summary, err := GenerateSummaryByAI(apiKey, htmlContent, "zh-CN", 100)
// 	if err != nil {
// 		fmt.Println("生成失败：", err)
// 		return
// 	}
// 	fmt.Println("AI摘要：", summary)
// 	// 输出示例：Vue3暗黑模式教程聚焦前端开发，介绍通过CSS变量和Tiptap编辑器适配暗黑模式的方法，该方案适配Vben Admin，支持多语言，性能较UEditor提升80%。
// }
