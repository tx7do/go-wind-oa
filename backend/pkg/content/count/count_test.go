package count

import (
	"strings"
	"testing"
)

// TestNewContentCounter 测试初始化
func TestNewContentCounter(t *testing.T) {
	tests := []struct {
		name        string
		htmlContent string
		wantHTML    string
		wantPlain   string
	}{
		{
			name:        "simple HTML",
			htmlContent: "<p>Hello World</p>",
			wantHTML:    "<p>Hello World</p>",
			wantPlain:   "Hello World",
		},
		{
			name:        "nested HTML",
			htmlContent: "<div><p>Test <strong>Content</strong></p></div>",
			wantHTML:    "<div><p>Test <strong>Content</strong></p></div>",
			wantPlain:   "Test Content",
		},
		{
			name:        "empty content",
			htmlContent: "",
			wantHTML:    "",
			wantPlain:   "",
		},
		{
			name:        "only HTML tags",
			htmlContent: "<div></div><p></p>",
			wantHTML:    "<div></div><p></p>",
			wantPlain:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			counter := NewContentCounter(tt.htmlContent)

			if counter.HTMLContent != tt.wantHTML {
				t.Errorf("HTMLContent = %q, want %q", counter.HTMLContent, tt.wantHTML)
			}

			if counter.PlainText != tt.wantPlain {
				t.Errorf("PlainText = %q, want %q", counter.PlainText, tt.wantPlain)
			}
		})
	}
}

// TestContentCounter_RawChars 测试纯字符数统计
func TestContentCounter_RawChars(t *testing.T) {
	tests := []struct {
		name        string
		htmlContent string
		want        int
	}{
		{
			name:        "simple text",
			htmlContent: "<p>Hello</p>",
			want:        5,
		},
		{
			name:        "text with spaces",
			htmlContent: "<p>Hello World</p>",
			want:        11, // "Hello World" = 11 characters including space
		},
		{
			name:        "Chinese text",
			htmlContent: "<p>你好世界</p>",
			want:        4,
		},
		{
			name:        "mixed Chinese and English",
			htmlContent: "<p>Vue3 暗黑模式</p>",
			want:        9, // "Vue3 暗黑模式" = 9 characters
		},
		{
			name:        "emoji",
			htmlContent: "<p>Hello 🔥</p>",
			want:        7, // "Hello 🔥" = 7 characters (emoji counts as 1)
		},
		{
			name:        "empty",
			htmlContent: "<p></p>",
			want:        0,
		},
		{
			name:        "multiple spaces",
			htmlContent: "<p>Hello    World</p>",
			want:        14, // "Hello    World" after HTML removal, spaceRegex doesn't run in RawChars
		},
		{
			name:        "newlines and tabs",
			htmlContent: "<p>Hello\n\tWorld</p>",
			want:        12, // "Hello\n\tWorld" = 12 characters
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			counter := NewContentCounter(tt.htmlContent)
			got := counter.RawChars()

			if got != tt.want {
				t.Errorf("RawChars() = %d, want %d (content: %q)", got, tt.want, counter.PlainText)
			}
		})
	}
}

// TestContentCounter_ValidChars 测试有效字符数统计（排除空白）
func TestContentCounter_ValidChars(t *testing.T) {
	tests := []struct {
		name        string
		htmlContent string
		want        int
	}{
		{
			name:        "no whitespace",
			htmlContent: "<p>Hello</p>",
			want:        5,
		},
		{
			name:        "with single space",
			htmlContent: "<p>Hello World</p>",
			want:        10, // "HelloWorld" = 10 (space removed)
		},
		{
			name:        "multiple spaces",
			htmlContent: "<p>Hello    World</p>",
			want:        10, // "HelloWorld" = 10
		},
		{
			name:        "Chinese text",
			htmlContent: "<p>你好 世界</p>",
			want:        4, // "你好世界" = 4
		},
		{
			name:        "mixed with spaces",
			htmlContent: "<p>Vue3 暗黑 模式</p>",
			want:        8, // "Vue3暗黑模式" = 8
		},
		{
			name:        "newlines and tabs",
			htmlContent: "<p>Hello\n\t\r World</p>",
			want:        10, // "HelloWorld" = 10
		},
		{
			name:        "only whitespace",
			htmlContent: "<p>   \n\t  </p>",
			want:        0,
		},
		{
			name:        "emoji",
			htmlContent: "<p>Test 🔥 Content</p>",
			want:        12, // "Test🔥Content" = 12
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			counter := NewContentCounter(tt.htmlContent)
			got := counter.ValidChars()

			if got != tt.want {
				t.Errorf("ValidChars() = %d, want %d (plain: %q)", got, tt.want, counter.PlainText)
			}
		})
	}
}

// TestContentCounter_CNWords 测试中文规则字数统计
func TestContentCounter_CNWords(t *testing.T) {
	tests := []struct {
		name        string
		htmlContent string
		want        int
	}{
		{
			name:        "pure English",
			htmlContent: "<p>Hello</p>",
			want:        5, // H=1, e=1, l=1, l=1, o=1
		},
		{
			name:        "English with space",
			htmlContent: "<p>Hello World</p>",
			want:        10, // 5 + 5, spaces ignored
		},
		{
			name:        "pure Chinese",
			htmlContent: "<p>你好</p>",
			want:        4, // 你=2, 好=2
		},
		{
			name:        "Chinese with space",
			htmlContent: "<p>你好 世界</p>",
			want:        8, // 你=2, 好=2, 世=2, 界=2
		},
		{
			name:        "mixed Chinese and English",
			htmlContent: "<p>Vue3 暗黑模式</p>",
			want:        12, // V=1, u=1, e=1, 3=1, 暗=2, 黑=2, 模=2, 式=2
		},
		{
			name:        "example from code",
			htmlContent: "<p>Vue3 <strong>暗黑模式</strong> 教程 🔥</p>",
			want:        17, // V=1, u=1, e=1, 3=1, 暗=2, 黑=2, 模=2, 式=2, 教=2, 程=2, 🔥=1
		},
		{
			name:        "empty",
			htmlContent: "<p></p>",
			want:        0,
		},
		{
			name:        "only spaces",
			htmlContent: "<p>   </p>",
			want:        0,
		},
		{
			name:        "numbers",
			htmlContent: "<p>12345</p>",
			want:        5, // 1=1, 2=1, 3=1, 4=1, 5=1
		},
		{
			name:        "punctuation",
			htmlContent: "<p>你好！世界。</p>",
			want:        10, // 你=2, 好=2, ！=1, 世=2, 界=2, 。=1
		},
		{
			name:        "emoji",
			htmlContent: "<p>测试🔥内容</p>",
			want:        9, // 测=2, 试=2, 🔥=1, 内=2, 容=2
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			counter := NewContentCounter(tt.htmlContent)
			got := counter.CNWords()

			if got != tt.want {
				t.Errorf("CNWords() = %d, want %d (plain: %q)", got, tt.want, counter.PlainText)
			}
		})
	}
}

// TestContentCounter_ComplexHTML 测试复杂 HTML 结构
func TestContentCounter_ComplexHTML(t *testing.T) {
	tests := []struct {
		name        string
		htmlContent string
		wantRaw     int
		wantValid   int
		wantCN      int
	}{
		{
			name: "article with multiple tags",
			htmlContent: `<article>
				<h1>标题</h1>
				<p>这是一段<strong>重要</strong>内容。</p>
			</article>`,
			wantRaw:   25, // 包含换行和制表符
			wantValid: 11, // "标题这是一段重要内容。"
			wantCN:    35, // 包含空格和换行符的字符数
		},
		{
			name: "nested formatting",
			htmlContent: `<div>
				<p>Hello <strong><em>World</em></strong></p>
			</div>`,
			wantRaw:   20, // 包含换行和制表符
			wantValid: 10, // "HelloWorld"
			wantCN:    19, // 包含空格、换行符
		},
		{
			name:        "code block",
			htmlContent: `<pre><code>const x = 1;</code></pre>`,
			wantRaw:     12, // "const x = 1;"
			wantValid:   9,  // "constx=1;"（空格被移除）
			wantCN:      9,  // 每个非空格字符 = 1
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			counter := NewContentCounter(tt.htmlContent)

			gotRaw := counter.RawChars()
			if gotRaw != tt.wantRaw {
				t.Errorf("RawChars() = %d, want %d", gotRaw, tt.wantRaw)
			}

			gotValid := counter.ValidChars()
			if gotValid != tt.wantValid {
				t.Errorf("ValidChars() = %d, want %d", gotValid, tt.wantValid)
			}

			gotCN := counter.CNWords()
			if gotCN != tt.wantCN {
				t.Errorf("CNWords() = %d, want %d", gotCN, tt.wantCN)
			}
		})
	}
}

// TestContentCounter_EdgeCases 测试边界情况
func TestContentCounter_EdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		htmlContent string
		wantRaw     int
		wantValid   int
		wantCN      int
	}{
		{
			name:        "empty string",
			htmlContent: "",
			wantRaw:     0,
			wantValid:   0,
			wantCN:      0,
		},
		{
			name:        "only HTML tags",
			htmlContent: "<div><p><span></span></p></div>",
			wantRaw:     0,
			wantValid:   0,
			wantCN:      0,
		},
		{
			name:        "only whitespace",
			htmlContent: "<p>   \n\t\r   </p>",
			wantRaw:     9,
			wantValid:   0,
			wantCN:      3, // 3个空格计数为3
		},
		{
			name:        "very long content",
			htmlContent: "<p>" + strings.Repeat("测试", 1000) + "</p>",
			wantRaw:     2000,
			wantValid:   2000,
			wantCN:      4000, // 测=2, 试=2, repeated 1000 times
		},
		{
			name:        "special characters",
			htmlContent: "<p>@#$%^&*()</p>",
			wantRaw:     9,
			wantValid:   9,
			wantCN:      9,
		},
		{
			name:        "HTML entities (not decoded)",
			htmlContent: "<p>&lt;div&gt;</p>",
			wantRaw:     11, // "&lt;div&gt;"
			wantValid:   11,
			wantCN:      11,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			counter := NewContentCounter(tt.htmlContent)

			gotRaw := counter.RawChars()
			if gotRaw != tt.wantRaw {
				t.Errorf("RawChars() = %d, want %d", gotRaw, tt.wantRaw)
			}

			gotValid := counter.ValidChars()
			if gotValid != tt.wantValid {
				t.Errorf("ValidChars() = %d, want %d", gotValid, tt.wantValid)
			}

			gotCN := counter.CNWords()
			if gotCN != tt.wantCN {
				t.Errorf("CNWords() = %d, want %d", gotCN, tt.wantCN)
			}
		})
	}
}

// TestContentCounter_UnicodeHandling 测试 Unicode 处理
func TestContentCounter_UnicodeHandling(t *testing.T) {
	tests := []struct {
		name        string
		htmlContent string
		wantRaw     int
		wantCN      int
		description string
	}{
		{
			name:        "emoji single",
			htmlContent: "<p>🔥</p>",
			wantRaw:     1,
			wantCN:      1,
			description: "Single emoji should count as 1 rune",
		},
		{
			name:        "emoji multiple",
			htmlContent: "<p>🔥🎉✨</p>",
			wantRaw:     3,
			wantCN:      3,
			description: "Multiple emojis",
		},
		{
			name:        "combined emoji",
			htmlContent: "<p>👨‍👩‍👧‍👦</p>", // Family emoji (combined)
			wantRaw:     7,                // This is a combined emoji, counts as multiple runes
			wantCN:      7,
			description: "Combined emoji",
		},
		{
			name:        "Japanese Hiragana",
			htmlContent: "<p>ひらがな</p>",
			wantRaw:     4,
			wantCN:      4, // Japanese is not Han, so 1 per character
			description: "Japanese Hiragana",
		},
		{
			name:        "Japanese Kanji",
			htmlContent: "<p>漢字</p>",
			wantRaw:     2,
			wantCN:      4, // Kanji are Han characters, so 2 per character
			description: "Japanese Kanji (Han characters)",
		},
		{
			name:        "Korean",
			htmlContent: "<p>한글</p>",
			wantRaw:     2,
			wantCN:      2, // Korean is not Han
			description: "Korean",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			counter := NewContentCounter(tt.htmlContent)

			gotRaw := counter.RawChars()
			if gotRaw != tt.wantRaw {
				t.Errorf("%s: RawChars() = %d, want %d", tt.description, gotRaw, tt.wantRaw)
			}

			gotCN := counter.CNWords()
			if gotCN != tt.wantCN {
				t.Errorf("%s: CNWords() = %d, want %d", tt.description, gotCN, tt.wantCN)
			}
		})
	}
}

// TestContentCounter_RealWorldExamples 测试真实场景
func TestContentCounter_RealWorldExamples(t *testing.T) {
	tests := []struct {
		name        string
		htmlContent string
		checkFunc   func(*ContentCounter) bool
		description string
	}{
		{
			name: "blog post",
			htmlContent: `<article>
				<h1>Vue3 暗黑模式教程</h1>
				<p>本文介绍如何在 Vue3 项目中实现暗黑模式。</p>
				<p>通过 CSS 变量和 Tiptap 编辑器，可快速适配暗黑模式。</p>
			</article>`,
			checkFunc: func(c *ContentCounter) bool {
				raw := c.RawChars()
				valid := c.ValidChars()
				cn := c.CNWords()

				// 验证三种统计方法都返回合理的值
				return raw > 0 && valid > 0 && cn > 0 && valid <= raw && cn >= valid
			},
			description: "Blog post should have positive counts",
		},
		{
			name: "product description",
			htmlContent: `<div class="product">
				<strong>Premium Keyboard</strong> - Mechanical switches, RGB lighting
			</div>`,
			checkFunc: func(c *ContentCounter) bool {
				// 英文为主，但包含换行制表符，所以 CN words 可能大于 valid chars
				cn := c.CNWords()
				valid := c.ValidChars()
				// 允许合理范围
				return cn >= valid && cn <= valid*2
			},
			description: "English product description",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			counter := NewContentCounter(tt.htmlContent)

			if !tt.checkFunc(counter) {
				t.Errorf("%s failed: raw=%d, valid=%d, cn=%d",
					tt.description,
					counter.RawChars(),
					counter.ValidChars(),
					counter.CNWords())
			}
		})
	}
}

// BenchmarkNewContentCounter 基准测试：初始化
func BenchmarkNewContentCounter(b *testing.B) {
	htmlContent := `<div class="article">
		<p>Vue3 <strong>暗黑模式</strong>教程是前端开发的重要知识点。</p>
		<p>通过CSS变量和<em>Tiptap编辑器</em>，可快速适配暗黑模式。</p>
	</div>`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = NewContentCounter(htmlContent)
	}
}

// BenchmarkRawChars 基准测试：纯字符数统计
func BenchmarkRawChars(b *testing.B) {
	counter := NewContentCounter(`<p>Vue3 暗黑模式教程</p>`)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = counter.RawChars()
	}
}

// BenchmarkValidChars 基准测试：有效字符数统计
func BenchmarkValidChars(b *testing.B) {
	counter := NewContentCounter(`<p>Vue3 暗黑模式教程</p>`)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = counter.ValidChars()
	}
}

// BenchmarkCNWords 基准测试：中文规则字数统计
func BenchmarkCNWords(b *testing.B) {
	counter := NewContentCounter(`<p>Vue3 暗黑模式教程</p>`)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = counter.CNWords()
	}
}

// BenchmarkAllMethods 基准测试：所有方法
func BenchmarkAllMethods(b *testing.B) {
	htmlContent := `<article>
		<h1>Vue3 暗黑模式教程</h1>
		<p>本文介绍如何在 Vue3 项目中实现暗黑模式。</p>
		<p>通过 CSS 变量和 Tiptap 编辑器，可快速适配暗黑模式。</p>
	</article>`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		counter := NewContentCounter(htmlContent)
		_ = counter.RawChars()
		_ = counter.ValidChars()
		_ = counter.CNWords()
	}
}

// BenchmarkLargeContent 基准测试：大文件
func BenchmarkLargeContent(b *testing.B) {
	// 生成 10KB 的测试内容
	largeContent := "<article>" + strings.Repeat(
		"<p>这是一段很长的文本内容，用于测试大文件的性能表现。This is a long text for performance testing.</p>",
		100,
	) + "</article>"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		counter := NewContentCounter(largeContent)
		_ = counter.RawChars()
		_ = counter.ValidChars()
		_ = counter.CNWords()
	}
}
