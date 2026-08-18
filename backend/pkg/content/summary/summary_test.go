package summary

import (
	"strings"
	"testing"
)

// TestGenerateSummaryByRule_BasicHTMLStripping 测试基本的 HTML 标签剥离功能
func TestGenerateSummaryByRule_BasicHTMLStripping(t *testing.T) {
	tests := []struct {
		name       string
		content    string
		maxLength  int
		bySentence bool
		expected   string
	}{
		{
			name:       "simple HTML paragraph",
			content:    "<p>Hello World</p>",
			maxLength:  20,
			bySentence: false,
			expected:   "Hello World",
		},
		{
			name:       "multiple HTML tags",
			content:    "<div><p>Test</p><span>Content</span></div>",
			maxLength:  20,
			bySentence: false,
			expected:   "TestContent",
		},
		{
			name:       "nested HTML tags",
			content:    "<div><p>This is <strong>bold</strong> text</p></div>",
			maxLength:  50,
			bySentence: false,
			expected:   "This is bold text",
		},
		{
			name:       "HTML with attributes",
			content:    `<p class="content" id="main">Hello</p>`,
			maxLength:  20,
			bySentence: false,
			expected:   "Hello",
		},
		{
			name:       "self-closing tags",
			content:    "Text<br/>More text<hr/>Final",
			maxLength:  30,
			bySentence: false,
			expected:   "TextMore textFinal",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GenerateSummaryByRule(tt.content, tt.maxLength, tt.bySentence)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

// TestGenerateSummaryByRule_WhitespaceHandling 测试空格和换行的处理
func TestGenerateSummaryByRule_WhitespaceHandling(t *testing.T) {
	tests := []struct {
		name       string
		content    string
		maxLength  int
		bySentence bool
		expected   string
	}{
		{
			name:       "multiple spaces",
			content:    "Hello    World    Test",
			maxLength:  20,
			bySentence: false,
			expected:   "Hello World Test",
		},
		{
			name:       "newlines and tabs",
			content:    "Hello\n\nWorld\t\tTest",
			maxLength:  30,
			bySentence: false,
			expected:   "Hello World Test",
		},
		{
			name:       "leading and trailing spaces",
			content:    "   Hello World   ",
			maxLength:  20,
			bySentence: false,
			expected:   "Hello World",
		},
		{
			name:       "HTML with whitespace",
			content:    "<p>  Hello   \n  World  </p>",
			maxLength:  20,
			bySentence: false,
			expected:   "Hello World",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GenerateSummaryByRule(tt.content, tt.maxLength, tt.bySentence)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

// TestGenerateSummaryByRule_TruncateByLength 测试按长度截断功能
func TestGenerateSummaryByRule_TruncateByLength(t *testing.T) {
	tests := []struct {
		name       string
		content    string
		maxLength  int
		bySentence bool
		expected   string
	}{
		{
			name:       "no truncation needed",
			content:    "Hello",
			maxLength:  10,
			bySentence: false,
			expected:   "Hello",
		},
		{
			name:       "exact length match",
			content:    "Hello",
			maxLength:  5,
			bySentence: false,
			expected:   "Hello",
		},
		{
			name:       "simple truncation",
			content:    "Hello World Test",
			maxLength:  5,
			bySentence: false,
			expected:   "Hello...",
		},
		{
			name:       "truncate to middle",
			content:    "This is a long text",
			maxLength:  7,
			bySentence: false,
			expected:   "This is...",
		},
		{
			name:       "truncate empty string",
			content:    "",
			maxLength:  10,
			bySentence: false,
			expected:   "暂无摘要",
		},
		{
			name:       "truncate whitespace only",
			content:    "   \n\t  ",
			maxLength:  10,
			bySentence: false,
			expected:   "暂无摘要",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GenerateSummaryByRule(tt.content, tt.maxLength, tt.bySentence)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

// TestGenerateSummaryByRule_TruncateBySentence 测试按句子截断功能
func TestGenerateSummaryByRule_TruncateBySentence(t *testing.T) {
	tests := []struct {
		name       string
		content    string
		maxLength  int
		bySentence bool
		expected   string
	}{
		{
			name:       "single sentence no truncation",
			content:    "This is a sentence.",
			maxLength:  50,
			bySentence: true,
			expected:   "This is a sentence.",
		},
		{
			name:       "multiple sentences - truncate at period",
			content:    "First sentence. Second sentence. Third sentence.",
			maxLength:  20,
			bySentence: true,
			expected:   "First sentence....",
		},
		{
			name:       "Chinese punctuation",
			content:    "这是第一句。这是第二句。这是第三句。",
			maxLength:  15,
			bySentence: true,
			expected:   "这是第一句。这是第二句。...",
		},
		{
			name:       "mixed Chinese and English punctuation",
			content:    "English sentence. 中文句子。More text!",
			maxLength:  20,
			bySentence: true,
			expected:   "English sentence....",
		},
		{
			name:       "no punctuation in text",
			content:    "This is a long text without any punctuation marks",
			maxLength:  10,
			bySentence: true,
			expected:   "This is a ...",
		},
		{
			name:       "exclamation mark as sentence end",
			content:    "What is this! This is another sentence.",
			maxLength:  15,
			bySentence: true,
			expected:   "What is this!...",
		},
		{
			name:       "question mark as sentence end",
			content:    "Why? Because this is important!",
			maxLength:  10,
			bySentence: true,
			expected:   "Why?...",
		},
		{
			name:       "semicolon as sentence end",
			content:    "First part; Second part.",
			maxLength:  12,
			bySentence: true,
			expected:   "First part;...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GenerateSummaryByRule(tt.content, tt.maxLength, tt.bySentence)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

// TestGenerateSummaryByRule_ChineseContent 测试中文内容处理
func TestGenerateSummaryByRule_ChineseContent(t *testing.T) {
	tests := []struct {
		name       string
		content    string
		maxLength  int
		bySentence bool
		expected   string
	}{
		{
			name:       "simple Chinese text",
			content:    "<p>Vue3 暗黑模式教程是前端开发的重要知识点。</p>",
			maxLength:  50,
			bySentence: false,
			expected:   "Vue3 暗黑模式教程是前端开发的重要知识点。",
		},
		{
			name:       "Chinese truncation by length",
			content:    "这是一个很长的中文文本，需要被截断以生成摘要。",
			maxLength:  10,
			bySentence: false,
			expected:   "这是一个很长的中文文...",
		},
		{
			name:       "Chinese truncation by sentence",
			content:    "第一句话。第二句话。第三句话。",
			maxLength:  10,
			bySentence: true,
			expected:   "第一句话。第二句话。...",
		},
		{
			name:       "mixed content with HTML",
			content:    "<p>Vue3 <strong>暗黑模式</strong>教程。通过 CSS 变量实现。</p>",
			maxLength:  20,
			bySentence: true,
			expected:   "Vue3 暗黑模式教程。...",
		},
		{
			name:       "Chinese punctuation only",
			content:    "一二三四五，六七八九十。",
			maxLength:  15,
			bySentence: false,
			expected:   "一二三四五，六七八九十。",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GenerateSummaryByRule(tt.content, tt.maxLength, tt.bySentence)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

// TestGenerateSummaryByRule_EdgeCases 测试边界情况
func TestGenerateSummaryByRule_EdgeCases(t *testing.T) {
	tests := []struct {
		name       string
		content    string
		maxLength  int
		bySentence bool
		expected   string
	}{
		{
			name:       "zero max length",
			content:    "Hello World",
			maxLength:  0,
			bySentence: false,
			expected:   "...",
		},
		{
			name:       "negative max length",
			content:    "Hello World",
			maxLength:  -1,
			bySentence: false,
			expected:   "...",
		},
		{
			name:       "max length 1",
			content:    "Hello World",
			maxLength:  1,
			bySentence: false,
			expected:   "H...",
		},
		{
			name:       "only HTML tags",
			content:    "<p></p><div></div><span></span>",
			maxLength:  50,
			bySentence: false,
			expected:   "暂无摘要",
		},
		{
			name:       "special characters",
			content:    "Hello & goodbye @ test # symbol $ end",
			maxLength:  50,
			bySentence: false,
			expected:   "Hello & goodbye @ test # symbol $ end",
		},
		{
			name:       "emoji content",
			content:    "Hello 👋 World 🌍 Test 🎉",
			maxLength:  50,
			bySentence: false,
			expected:   "Hello 👋 World 🌍 Test 🎉",
		},
		{
			name:       "very long content",
			content:    "a" + strings.Repeat("b", 10000),
			maxLength:  100,
			bySentence: false,
			expected:   "a" + strings.Repeat("b", 99) + "...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GenerateSummaryByRule(tt.content, tt.maxLength, tt.bySentence)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

// TestGenerateSummaryByRule_ComplexHTML 测试复杂 HTML 结构
func TestGenerateSummaryByRule_ComplexHTML(t *testing.T) {
	tests := []struct {
		name       string
		content    string
		maxLength  int
		bySentence bool
		expected   string
	}{
		{
			name: "article with multiple paragraphs",
			content: `
				<article>
					<p>First paragraph with <strong>bold text</strong>.</p>
					<p>Second paragraph with <em>italic text</em>.</p>
					<p>Third paragraph.</p>
				</article>
			`,
			maxLength:  50,
			bySentence: true,
			expected:   "First paragraph with bold text....",
		},
		{
			name: "nested formatting",
			content: `
				<div class="content">
					<p>This is <strong><em>very important</em></strong> content.</p>
				</div>
			`,
			maxLength:  50,
			bySentence: false,
			expected:   "This is very important content.",
		},
		{
			name: "content with links",
			content: `
				<p>Check out <a href="https://example.com">this link</a> for more info.</p>
			`,
			maxLength:  30,
			bySentence: true,
			expected:   "Check out this link for more i...",
		},
		{
			name: "content with code blocks",
			content: `
				<p>Here is some code:</p>
				<pre><code>const x = 1;</code></pre>
				<p>And more text.</p>
			`,
			maxLength:  30,
			bySentence: false,
			expected:   "Here is some code: const x = 1...",
		},
		{
			name: "content with line breaks",
			content: `
				<p>Line 1<br/>Line 2<br/>Line 3</p>
			`,
			maxLength:  20,
			bySentence: false,
			expected:   "Line 1Line 2Line 3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GenerateSummaryByRule(tt.content, tt.maxLength, tt.bySentence)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

// TestGenerateSummaryByRule_RealWorldExamples 测试真实世界的例子
func TestGenerateSummaryByRule_RealWorldExamples(t *testing.T) {
	tests := []struct {
		name       string
		content    string
		maxLength  int
		bySentence bool
		checkFunc  func(string) bool
	}{
		{
			name: "blog post excerpt",
			content: `
				<article>
					<p>Vue.js is a progressive JavaScript framework for building user interfaces. 
					It is designed from the ground up to be incrementally adoptable. 
					The core library is focused on the view layer only, and is easy to pick up and integrate with other libraries or existing projects.</p>
				</article>
			`,
			maxLength:  150,
			bySentence: false,
			checkFunc: func(s string) bool {
				return len(s) > 0 && len(s) <= 160 && !strings.Contains(s, "<")
			},
		},
		{
			name: "product description",
			content: `
				<div class="product">
					<p><strong>Premium Keyboard</strong> with mechanical switches. Features:</p>
					<ul>
						<li>RGB backlighting</li>
						<li>Wireless connectivity</li>
						<li>Ergonomic design</li>
					</ul>
					<p>Available now for $99.99</p>
				</div>
			`,
			maxLength:  100,
			bySentence: true,
			checkFunc: func(s string) bool {
				return strings.HasPrefix(s, "Premium Keyboard") || strings.HasPrefix(s, "Product")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GenerateSummaryByRule(tt.content, tt.maxLength, tt.bySentence)
			if !tt.checkFunc(result) {
				t.Errorf("result %q does not pass validation", result)
			}
		})
	}
}

// BenchmarkGenerateSummaryByRule_ByLength 基准测试：按长度截断
func BenchmarkGenerateSummaryByRule_ByLength(b *testing.B) {
	content := `
		<article>
			<p>Vue.js is a progressive JavaScript framework. It provides data-reactive components with a simple and flexible API.</p>
			<p>Unlike some frameworks, Vue.js is designed to scale up as well as scale down.</p>
		</article>
	`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		GenerateSummaryByRule(content, 100, false)
	}
}

// BenchmarkGenerateSummaryByRule_BySentence 基准测试：按句子截断
func BenchmarkGenerateSummaryByRule_BySentence(b *testing.B) {
	content := `
		<article>
			<p>Vue.js is a progressive JavaScript framework. It provides data-reactive components with a simple and flexible API.</p>
			<p>Unlike some frameworks, Vue.js is designed to scale up as well as scale down.</p>
		</article>
	`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		GenerateSummaryByRule(content, 100, true)
	}
}

// BenchmarkGenerateSummaryByRule_LargeContent 基准测试：大型内容
func BenchmarkGenerateSummaryByRule_LargeContent(b *testing.B) {
	// 生成 10KB 的测试内容
	largeContent := `<article>` + strings.Repeat(
		`<p>This is a paragraph with some content that will be repeated many times to create a large document for testing performance.</p>`,
		100,
	) + `</article>`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		GenerateSummaryByRule(largeContent, 150, false)
	}
}

// 测试辅助函数
func TestTruncateByLength(t *testing.T) {
	tests := []struct {
		name      string
		text      string
		maxLength int
		expected  string
	}{
		{
			name:      "no truncation",
			text:      "Hello",
			maxLength: 10,
			expected:  "Hello",
		},
		{
			name:      "exact length",
			text:      "Hello",
			maxLength: 5,
			expected:  "Hello",
		},
		{
			name:      "truncation needed",
			text:      "Hello World",
			maxLength: 5,
			expected:  "Hello",
		},
		{
			name:      "Chinese characters",
			text:      "你好世界测试",
			maxLength: 3,
			expected:  "你好世",
		},
		{
			name:      "emoji",
			text:      "Hello 👋 World 🌍",
			maxLength: 8,
			expected:  "Hello 👋 ",
		},
		{
			name:      "zero length",
			text:      "Hello",
			maxLength: 0,
			expected:  "",
		},
		{
			name:      "negative length",
			text:      "Hello",
			maxLength: -5,
			expected:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := truncateByLength(tt.text, tt.maxLength)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

// TestTruncateBySentence 测试按句子截断的辅助函数
func TestTruncateBySentence(t *testing.T) {
	tests := []struct {
		name      string
		text      string
		maxLength int
		expected  string
	}{
		{
			name:      "single sentence",
			text:      "This is a sentence.",
			maxLength: 50,
			expected:  "This is a sentence.",
		},
		{
			name:      "multiple sentences",
			text:      "First. Second. Third.",
			maxLength: 10,
			expected:  "First.",
		},
		{
			name:      "Chinese sentences",
			text:      "第一句。第二句。",
			maxLength: 10,
			expected:  "第一句。第二句。",
		},
		{
			name:      "no punctuation",
			text:      "No punctuation here",
			maxLength: 10,
			expected:  "No punctua",
		},
		{
			name:      "multiple punctuation types",
			text:      "What? Why! Because.",
			maxLength: 10,
			expected:  "What? Why!",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := truncateBySentence(tt.text, tt.maxLength)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}
