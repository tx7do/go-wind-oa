package summary

import (
	"regexp"
	"strings"
	"unicode/utf8"
)

// 预编译正则
var (
	htmlTagRegex     = regexp.MustCompile(`<[^>]+>`)    // 匹配HTML标签
	spaceRegex       = regexp.MustCompile(`\s+`)        // 匹配多余空格/换行
	sentenceEndRegex = regexp.MustCompile(`[。！？；.!?;]`) // 匹配句子结束标点
)

// GenerateSummaryByRule 规则化生成摘要（非AI）
// content: 原始HTML内容
// maxLength: 摘要最大字符数（如150）
// bySentence: 是否按句子截断（true=更完整，false=直接截断）
func GenerateSummaryByRule(content string, maxLength int, bySentence bool) string {
	// 步骤1：剥离HTML标签，转为纯文本
	plainText := htmlTagRegex.ReplaceAllString(content, "")
	// 步骤2：去除多余空格/换行，合并为单行
	plainText = spaceRegex.ReplaceAllString(plainText, " ")
	plainText = strings.TrimSpace(plainText)

	// 步骤3：无内容则返回默认值
	if plainText == "" {
		return "暂无摘要"
	}

	// 步骤4：按规则截断
	var summary string
	if bySentence {
		// 方案A：按句子截断（更自然，优先取完整句子）
		summary = truncateBySentence(plainText, maxLength)
	} else {
		// 方案B：直接截断（性能更高，适合短摘要）
		summary = truncateByLength(plainText, maxLength)
	}

	// 步骤5：补充省略号（如果截断了）
	if utf8.RuneCountInString(summary) < utf8.RuneCountInString(plainText) {
		summary += "..."
	}

	return summary
}

// 按字符长度截断（简单直接）
func truncateByLength(text string, maxLength int) string {
	if maxLength <= 0 {
		return ""
	}
	runes := []rune(text)
	if len(runes) <= maxLength {
		return text
	}
	return string(runes[:maxLength])
}

// 按句子截断（优先取完整句子）
func truncateBySentence(text string, maxLength int) string {
	runes := []rune(text)
	// 1. 如果文本长度不足，直接返回
	if len(runes) <= maxLength {
		return text
	}

	// 2. 截取前maxLength个字符，再找最后一个句子结束标点
	truncated := string(runes[:maxLength])
	lastEndIdx := sentenceEndRegex.FindAllStringIndex(truncated, -1)
	if len(lastEndIdx) == 0 {
		// 无结束标点，直接返回截断内容
		return truncated
	}

	// 3. 取到最后一个完整句子
	lastIdx := lastEndIdx[len(lastEndIdx)-1][1]
	return truncated[:lastIdx]
}

// 使用示例
// func main() {
// 	htmlContent := `<p>Vue3 暗黑模式教程是前端开发的重要知识点。通过CSS变量和Tiptap编辑器，可快速适配暗黑模式，提升用户体验。</p>`
// 	// 按句子截断，最大150字符
// 	summary := GenerateSummaryByRule(htmlContent, 50, true)
// 	fmt.Println(summary) // 输出：Vue3 暗黑模式教程是前端开发的重要知识点。通过CSS变量和Tiptap编辑器，可快速适配暗黑模式，提升用户体验。
// }
