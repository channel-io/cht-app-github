package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEmoji(t *testing.T) {
	t.Parallel()
	name := "man_bowing"
	assert.Equal(t, ":man_bowing:", Emoji(name))
}

func TestMention(t *testing.T) {
	tests := []struct {
		mentionType MentionType
		mentionID   string
		mentionName string
		expected    string
	}{
		{
			mentionType: MentionTypeManager,
			mentionID:   "mention-1",
			mentionName: "@claud",
			expected:    "<link type=\"manager\" value=\"mention-1\">@claud</link>",
		},
		{
			mentionType: MentionTypeTeam,
			mentionID:   "mention-2",
			mentionName: "@devops",
			expected:    "<link type=\"team\" value=\"mention-2\">@devops</link>",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run("TestMention", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.expected, Mention(tc.mentionType, tc.mentionID, tc.mentionName))
		})
	}
}

func TestBold(t *testing.T) {
	t.Parallel()
	s := "bold message"
	assert.Equal(t, "<b>bold message</b>", Bold(s))
}

func TestItalic(t *testing.T) {
	t.Parallel()
	s := "italic message"
	assert.Equal(t, "<i>italic message</i>", Italic(s))
}

func TestInlineLink(t *testing.T) {
	t.Parallel()
	href := "https://channel.io"
	s := "channel"
	assert.Equal(
		t,
		"<link type=\"url\" value=\"https://channel.io\">channel</link>",
		InlineLink(href, s),
	)
}

func TestVariable(t *testing.T) {
	tests := []struct {
		key      string
		alt      string
		expected string
	}{
		{
			key:      "somekey",
			expected: `${somekey}`,
		},
		{
			key:      "somekey",
			alt:      "somealt",
			expected: `${somekey|somealt}`,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run("TestVariable", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.expected, Variable(tc.key, tc.alt))
		})
	}
}

func TestEscapedString(t *testing.T) {
	tests := []struct {
		s        string
		expected string
	}{
		{
			s:        "no escape",
			expected: "no escape",
		},
		{
			s:        `escape "quote"`,
			expected: "escape &quot;quote&quot;",
		},
		{
			s:        `escape && amp`,
			expected: "escape &amp;&amp; amp",
		},
		{
			s:        `escape << lt`,
			expected: "escape &lt;&lt; lt",
		},
		{
			s:        `escape >> gt`,
			expected: "escape &gt;&gt; gt",
		},
		{
			s:        `<sample type="hello">world & bye</sample>`,
			expected: "&lt;sample type=&quot;hello&quot;&gt;world &amp; bye&lt;/sample&gt;",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run("EscapedString", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.expected, EscapedString(tc.s))
		})
	}
}
