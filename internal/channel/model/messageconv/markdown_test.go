package messageconv

import (
	"embed"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/channel-io/cht-app-github/internal/channel/model"
)

//go:embed testdata/*
var testdata embed.FS

func TestFromMarkdown_Convert_List(t *testing.T) {
	t.Parallel()
	md, err := testdata.ReadFile("testdata/list.md")
	assert.NoError(t, err)

	actual := FromGithubMarkdown(md, nil).Convert()
	expected := []model.MessageBlock{
		model.NewTextBlock("bullet list"),
		model.NewBulletsBlock(
			[]model.MessageBlock{
				model.NewTextBlock("first"),
				model.NewTextBlock("second"),
				model.NewTextBlock("third"),
			},
		),

		model.NewTextBlock("bullet list 2"),
		model.NewBulletsBlock(
			[]model.MessageBlock{
				model.NewTextBlock("first"),
				model.NewTextBlock("second"),
				model.NewTextBlock("third"),
			},
		),

		// Ordered List is also coverted to bullet list
		model.NewTextBlock("ordered list"),
		model.NewBulletsBlock(
			[]model.MessageBlock{
				model.NewTextBlock("first"),
				model.NewTextBlock("second"),
				model.NewTextBlock("third"),
			},
		),

		model.NewTextBlock("ordered list 2"),
		model.NewBulletsBlock(
			[]model.MessageBlock{
				model.NewTextBlock("first"),
				model.NewTextBlock("second"),
				model.NewTextBlock("third"),
			},
		),
	}
	assert.Equal(t, expected, actual)
}

func TestFromMarkdown_Convert_Heading(t *testing.T) {
	t.Parallel()
	md, err := testdata.ReadFile("testdata/heading.md")
	assert.NoError(t, err)

	actual := FromGithubMarkdown(md, nil).Convert()
	expected := []model.MessageBlock{
		// heading is interpreted to bold
		model.NewTextBlock("<b>head 1</b>"),
		model.NewTextBlock("title"),

		model.NewTextBlock("<b>head 2</b>"),
		model.NewTextBlock("subject"),

		model.NewTextBlock("<b>head 3</b>"),
		model.NewTextBlock("content"),
	}
	assert.Equal(t, expected, actual)
}

func TestFromMarkdown_Convert_Link(t *testing.T) {
	t.Parallel()
	md, err := testdata.ReadFile("testdata/link.md")
	assert.NoError(t, err)

	actual := FromGithubMarkdown(md, nil).Convert()
	expected := []model.MessageBlock{
		model.NewTextBlock("link - <link type=\"url\" value=\"https://channel.io\">channel talk</link>\nauto-link - <link type=\"url\" value=\"https://channel.io\">https://channel.io</link>"),
		model.NewTextBlock("nested link"),
		model.NewBulletsBlock(
			[]model.MessageBlock{
				model.NewTextBlock("first <link type=\"url\" value=\"https://channel.io\">channel talk</link>"),
			},
		),
	}
	assert.Equal(t, expected, actual)
}

func TestFromMarkdown_Convert_Emphasize(t *testing.T) {
	t.Parallel()
	md, err := testdata.ReadFile("testdata/emphasize.md")
	assert.NoError(t, err)

	actual := FromGithubMarkdown(md, nil).Convert()
	expected := []model.MessageBlock{
		model.NewTextBlock("<b>bold</b> <i>italic</i>"),
	}
	assert.Equal(t, expected, actual)
}

func TestFromMarkdown_Convert_CodeBlock(t *testing.T) {
	t.Parallel()
	md, err := testdata.ReadFile("testdata/codeblock.md")
	assert.NoError(t, err)

	actual := FromGithubMarkdown(md, nil).Convert()

	expected := []model.MessageBlock{
		model.NewTextBlock("code block"),
		model.NewCodeBlock("<html>\n  <body>\n  </body>\n</html>\n", nil),
		model.NewTextBlock("fenced code block"),
		model.NewCodeBlock("func main() {\n    fmt.Println(\"Hello World!\")\n}\n", nil),
	}
	assert.Equal(t, expected, actual)
}

func TestFromMarkdown_Convert_Mention(t *testing.T) {
	t.Parallel()
	md, err := testdata.ReadFile("testdata/mention.md")
	assert.NoError(t, err)

	managerMap := map[string]model.Manager{
		"claud": {
			Name: "클로드",
			ID:   "12345",
		},
	}

	actual := FromGithubMarkdown(md, managerMap).Convert()

	expected := []model.MessageBlock{
		model.NewTextBlock("<b>title</b>"),
		model.NewTextBlock("Major reviewer: <link type=\"manager\" value=\"12345\">클로드</link>"),
		model.NewTextBlock("CC: @nobody"),
		model.NewTextBlock("content"),
	}
	assert.Equal(t, expected, actual)
}

func TestFromMarkdown_Convert_Overall(t *testing.T) {
	t.Parallel()
	md, err := testdata.ReadFile("testdata/sample.md")
	assert.NoError(t, err)

	actual := FromGithubMarkdown(md, nil).Convert()

	expected := []model.MessageBlock{
		model.NewTextBlock("<b>head 1</b>"),

		model.NewTextBlock("<b>head 2</b>"),
		model.NewBulletsBlock(
			[]model.MessageBlock{
				model.NewTextBlock("list 1"),
				model.NewTextBlock("list 2"),
				model.NewTextBlock("list 3"),
			},
		),

		model.NewTextBlock("<b>head 3</b>"),
		model.NewBulletsBlock(
			[]model.MessageBlock{
				model.NewTextBlock("list 4"),
				model.NewTextBlock("list 5"),
				model.NewTextBlock("list 6"),
			},
		),

		model.NewTextBlock("<b>head 4</b>"), model.NewBulletsBlock(
			[]model.MessageBlock{
				model.NewTextBlock("list 1\n  *list 2\n  *list 3\n    *list 4\n    *list 5\n      *list 6\n      *list 7\n  *list 8"),
				model.NewTextBlock("list 9"),
			},
		),

		model.NewTextBlock("<b>head 5</b>"),
		model.NewBulletsBlock(
			[]model.MessageBlock{
				model.NewTextBlock("ordered list 1"),
				model.NewTextBlock("ordered list 2"),
				model.NewTextBlock("ordered list 3"),
			},
		),

		model.NewTextBlock("<b>head 6</b>"),
		model.NewTextBlock("go to <link type=\"url\" value=\"https://channel.io\">channel talk</link> `page`"),

		model.NewCodeBlock("code block \nsuch code block - much wow\n", nil),
	}
	assert.Equal(t, expected, actual)
}

func TestFromMarkdown_Convert_Issue_Link(t *testing.T) {
	t.Parallel()
	md, err := testdata.ReadFile("testdata/github_link.md")
	assert.NoError(t, err)

	actual := FromGithubMarkdown(md, nil).Convert()
	expected := []model.MessageBlock{
		model.NewTextBlock("<b>What's Changed</b>"),
		model.NewBulletsBlock(
			[]model.MessageBlock{
				model.NewTextBlock("미국 가이드 메인페이지 문구 및 에셋 변경 by @choichoigang in <link type=\"url\" value=\"https://github.com/channel-io/ch-inhouse-frontend/pull/1776\">#1776</link>"),
				model.NewTextBlock("bezier v2 대응 by @igy95 in <link type=\"url\" value=\"https://github.com/channel-io/ch-inhouse-frontend/issues/1778\">#1778</link>"),
				model.NewTextBlock("Revert &quot;bezier v2 대응&quot; by @igy95 in <link type=\"url\" value=\"https://github.com/channel-io/ch-inhouse-frontend/pull/1779\">#1779</link>"),
				model.NewTextBlock("bezier v2 대응 by @igy95 in <link type=\"url\" value=\"https://github.com/channel-io/ch-inhouse-frontend/pull/1780\">#1780</link>"),
				model.NewTextBlock("exp &gt;&gt; main by @choichoigang in <link type=\"url\" value=\"https://github.com/channel-io/ch-inhouse-frontend/pull/1783\">#1783</link>"),
			},
		),
		model.NewTextBlock("<b>Full Changelog</b>: <link type=\"url\" value=\"https://github.com/channel-io/ch-inhouse-frontend/compare/ch-homepage-v1.10.6...ch-homepage-v1.10.7\">ch-homepage-v1.10.6...ch-homepage-v1.10.7</link>"),
	}
	assert.Equal(t, expected, actual)
}

func TestReleaseWithNewLines(t *testing.T) {
	content := "line-1 \r\nline-2 \r\nline-3\r\n"
	md := []byte(content)

	actual := FromGithubMarkdown(md, nil).Convert()
	expected := []model.MessageBlock{
		model.NewTextBlock("line-1 \nline-2 \nline-3"),
	}
	assert.Equal(t, expected, actual)
}

func TestFromMarkdown_Convert_Regression_Escape(t *testing.T) {
	t.Parallel()
	md, err := testdata.ReadFile("testdata/regression_escape.md")
	assert.NoError(t, err)

	actual := FromGithubMarkdown(md, nil).Convert()
	expected := []model.MessageBlock{
		model.NewTextBlock("<b>What's Changed</b>"),
		model.NewBulletsBlock(
			[]model.MessageBlock{
				model.NewTextBlock("[web-2133] 워크플로우 가격 안내 팔로업 by @nabi-chan in <link type=\"url\" value=\"https://github.com/channel-io/ch-inhouse-frontend/pull/1923\">#1923</link>"),
				model.NewTextBlock("main &lt;&lt; exp by @nabi-chan in <link type=\"url\" value=\"https://github.com/channel-io/ch-inhouse-frontend/pull/1924\">#1924</link>"),
			},
		),
		model.NewTextBlock("<b>Full Changelog</b>: <link type=\"url\" value=\"https://github.com/channel-io/ch-inhouse-frontend/compare/ch-homepage-v1.11.12...ch-homepage-v1.11.13\">ch-homepage-v1.11.12...ch-homepage-v1.11.13</link>"),
	}
	assert.Equal(t, expected, actual)
}
