package model

import (
	"fmt"
	"strings"
)

// ANTLRString := (Pattern | String)+
// Pattern     := Emoji | Mention | Variable | Bold | Italic | InlineLink

// Emoji     := :{EmojiName}:
// EmojiName := RegExp(/^[-+_0-9a-zA-Z]+$/)
func Emoji(name string) string {
	return fmt.Sprintf(":%s:", name)
}

// MentionType := "manager" | "team"
type MentionType string

const (
	MentionTypeManager MentionType = "manager"
	MentionTypeTeam    MentionType = "team"
)

// Mention     := <link type="{MentionType}" value="{MentionId}">{MentionName}</link>
// MentionId   := EscapedString
// MentionName := {ANTLRString}
func Mention(mt MentionType, id string, mentionName string) string {
	return fmt.Sprintf("<link type=\"%s\" value=\"%s\">%s</link>", mt, EscapedString(id), mentionName)
}

// Bold := <b>{ANTLRString}</b>
func Bold(s string) string {
	return fmt.Sprintf("<b>%s</b>", s)
}

// Italic := <i>{ANTLRString}</i>
func Italic(s string) string {
	return fmt.Sprintf("<i>%s</i>", s)
}

// InlineLink     := <link type="url" value="{InlineLinkHref}">{ANTLRString}</link>
// InlineLinkHref := EscapedString
func InlineLink(href string, s string) string {
	return fmt.Sprintf("<link type=\"url\" value=\"%s\">%s</link>", EscapedString(href), s)
}

// Variable    := ${{VariableKey}} | ${{VariableKey}|{VariableAlt}}
// VariableKey := RegExp(/^\w+(?:\.[^<>.\s|$]+)*$/) | ""
// VariableAlt := RegExp(/^[^\s}]*[^\v}]+[^\s}]*$/) | ""
func Variable(key, alt string) string {
	if alt == "" {
		return fmt.Sprintf("${%s}", key)
	}
	return fmt.Sprintf("${%s|%s}", key, alt)
}

var escapeMap = map[rune]string{
	'"': "&quot;",
	'&': "&amp;",
	'<': "&lt;",
	'>': "&gt;",
}

// EscapedString := String - ("\"" | "&" | "<" | ">")
func EscapedString(s string) string {
	var b strings.Builder
	for _, c := range s {
		if escaped, ok := escapeMap[c]; ok {
			b.WriteString(escaped)
		} else {
			b.WriteRune(c)
		}
	}
	return b.String()
}
