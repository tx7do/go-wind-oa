package count

import (
	"regexp"
	"unicode"
	"unicode/utf8"
)

// 预编译正则（全局只编译一次，提升性能）
var (
	htmlTagRegex = regexp.MustCompile(`<[^>]+>`)
	spaceRegex   = regexp.MustCompile(`\s+`)
)

// ContentCounter 内容字数统计器
type ContentCounter struct {
	HTMLContent string // 原始 HTML 内容
	PlainText   string // 去除 HTML 后的纯文本
}

// NewContentCounter 初始化统计器
func NewContentCounter(htmlContent string) *ContentCounter {
	plainText := htmlTagRegex.ReplaceAllString(htmlContent, "")
	return &ContentCounter{
		HTMLContent: htmlContent,
		PlainText:   plainText,
	}
}

// RawChars 统计纯字符数（含所有字符）
func (c *ContentCounter) RawChars() int {
	return utf8.RuneCountInString(c.PlainText)
}

// ValidChars 统计有效字符数（排除空白字符）
func (c *ContentCounter) ValidChars() int {
	validText := spaceRegex.ReplaceAllString(c.PlainText, "")
	return utf8.RuneCountInString(validText)
}

// CNWords 按中文规则统计（中文=2字符，英文=1字符）
func (c *ContentCounter) CNWords() int {
	total := 0
	for _, r := range c.PlainText {
		if unicode.Is(unicode.Han, r) {
			total += 2
		} else if r != ' ' {
			total += 1
		}
	}
	return total
}

func (c *ContentCounter) MultiLangWords(lang string) int {
	switch lang {
	case "zh-CN":
		return c.CNWords()
	case "en-US", "ja-JP":
		return c.ValidChars() // 英文/日文按纯字符统计
	default:
		return c.RawChars()
	}
}

// --------------- 使用示例 ---------------
// func main() {
// 	htmlContent := `<p>Vue3 <strong>暗黑模式</strong> 教程 🔥</p>`
// 	counter := NewContentCounter(htmlContent)
// 	fmt.Println("纯字符数：", counter.RawChars())    // 9
// 	fmt.Println("有效字符数：", counter.ValidChars()) // 9
// 	fmt.Println("中文规则字数：", counter.CNWords())   // 17
// }
