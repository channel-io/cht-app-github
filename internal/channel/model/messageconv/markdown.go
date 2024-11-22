package messageconv

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/text"

	"github.com/channel-io/cht-app-github/internal/channel/model"
)

type GithubMarkdownConverter struct {
	source []byte
	// managerMap is map from `githubID` to Manager
	managerMap map[string]model.Manager
	// accountRegex is Regexp for github account ID
	accountRegex *regexp.Regexp
	// issueURLRegex is Regexp for github issue URL
	// https://github.com/channel-io/ch-inhouse-frontend/pull/1776
	// https://github.com/channel-io/ch-inhouse-frontend/issues/1776
	issueURLRegex *regexp.Regexp
	// compareURLRegex is Regexp for github compare URL
	// https://github.com/channel-io/ch-inhouse-frontend/compare/ch-homepage-v1.10.6...ch-homepage-v1.10.7
	compareURLRegex *regexp.Regexp
}

func FromGithubMarkdown(
	source []byte,
	managerMap map[string]model.Manager,
) *GithubMarkdownConverter {
	return &GithubMarkdownConverter{
		source:          source,
		managerMap:      managerMap,
		accountRegex:    regexp.MustCompile(`@[a-zA-Z0-9_-]+`),
		issueURLRegex:   regexp.MustCompile("https?://github.com/[^/]+/[^/]+/(?:issues|pull)/([0-9]+)"),
		compareURLRegex: regexp.MustCompile("https?://github.com/[^/]+/[^/]+/compare/([^/]+)"),
	}
}

func (r *GithubMarkdownConverter) Convert() []model.MessageBlock {
	gm := goldmark.New(
		goldmark.WithExtensions(extension.GFM),
	)
	rd := text.NewReader(r.source)
	root := gm.Parser().Parse(rd)
	blocks := make([]model.MessageBlock, 0, root.ChildCount())
	for child := root.FirstChild(); child != nil; child = child.NextSibling() {
		blocks = append(blocks, r.buildMessageBlock(child))
	}

	return blocks
}

// buildMessageBlock build a single block for a leaf node
func (r *GithubMarkdownConverter) buildMessageBlock(node ast.Node) model.MessageBlock {
	switch node.Kind() {
	case ast.KindList:
		blocks := make([]model.MessageBlock, 0, node.ChildCount())
		for child := node.FirstChild(); child != nil; child = child.NextSibling() {
			text := r.buildListText(child)
			blocks = append(blocks, model.NewTextBlock(text))
		}
		return model.NewBulletsBlock(blocks)

	case ast.KindCodeBlock:
		text := r.buildText(node, false)
		return model.NewCodeBlock(text, nil)

	case ast.KindFencedCodeBlock:
		// Note: Language는 채널톡 블록 v1.0 문서 상으로 Hidden Spec으로 되어 있어서 사용하지 않기로 결정.
		// b := node.(*ast.FencedCodeBlock)
		// lang := string(b.Language(r.source))
		text := r.buildText(node, false)
		return model.NewCodeBlock(text, nil)
	}

	return model.NewTextBlock(
		r.buildText(node, true),
	)
}

func (r *GithubMarkdownConverter) buildText(node ast.Node, escape bool) string {
	var buf bytes.Buffer
	r.buildTextAux(&buf, node, escape)
	return buf.String()
}

func (r *GithubMarkdownConverter) buildTextAux(buf *bytes.Buffer, node ast.Node, escape bool) {
	// Note: supported blocks for ChannelTalk
	switch node.Kind() {
	case ast.KindEmphasis:
		b := node.(*ast.Emphasis)
		text := r.buildRawText(node, escape)

		if b.Level == 1 {
			buf.WriteString(model.Italic(text))
		} else {
			buf.WriteString(model.Bold(text))
		}

	case ast.KindHeading:
		text := r.buildRawText(node, escape)
		buf.WriteString(model.Bold(text))

	case ast.KindAutoLink:
		b := node.(*ast.AutoLink)
		url := string(b.URL(r.source))
		if ref, ok := r.getGithubReferenceString(url); ok {
			buf.WriteString(model.InlineLink(url, ref))
		} else {
			buf.WriteString(model.InlineLink(url, url))
		}

	case ast.KindLink:
		b := node.(*ast.Link)
		url := string(b.Destination)
		value := string(b.Text(r.source))
		buf.WriteString(model.InlineLink(url, value))

	case ast.KindCodeSpan:
		buf.WriteRune('`')
		buf.WriteString(r.buildRawText(node, escape))
		buf.WriteRune('`')

	case ast.KindText:
		b := node.(*ast.Text)
		if b.SoftLineBreak() || b.HardLineBreak() {
			buf.Write([]byte("\n"))
			return
		}
		buf.WriteString(r.buildRawText(node, escape))

	default:
		if node.Type() == ast.TypeBlock && !node.IsRaw() {
			for child := node.FirstChild(); child != nil; child = child.NextSibling() {
				r.buildTextAux(buf, child, escape)
			}
			return
		}

		buf.WriteString(r.buildRawText(node, escape))
	}
}

func (r *GithubMarkdownConverter) getGithubReferenceString(url string) (n string, ok bool) {
	if match := r.issueURLRegex.FindStringSubmatch(url); len(match) == 2 {
		return fmt.Sprintf("#%s", match[1]), true
	}

	if match := r.compareURLRegex.FindStringSubmatch(url); len(match) == 2 {
		return match[1], true
	}

	return "", false
}

// buildRawText returns raw text content of given node
func (r *GithubMarkdownConverter) buildRawText(node ast.Node, escape bool) string {
	text := string(r.buildRawTextAux(node))
	if escape {
		text = model.EscapedString(text)
	}

	return r.replaceManagerMention(text)
}

func (r *GithubMarkdownConverter) buildRawTextAux(node ast.Node) []byte {
	if node.Type() == ast.TypeInline {
		return node.Text(r.source)
	}

	// Note: case ast.TypeBlock
	var buf bytes.Buffer

	if node.IsRaw() {
		lines := node.Lines()
		for i := 0; i < lines.Len(); i++ {
			l := lines.At(i)
			buf.Write(l.Value(r.source))
		}
		return buf.Bytes()
	}

	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		buf.Write(r.buildRawTextAux(child))
	}

	return buf.Bytes()
}

func (r *GithubMarkdownConverter) replaceManagerMention(text string) string {
	matches := r.accountRegex.FindAllString(text, -1)
	for _, match := range matches {
		githubUsername := string(match)[1:]
		if manager, ok := r.managerMap[strings.ToLower(githubUsername)]; ok {
			mention := model.Mention(model.MentionTypeManager, manager.ID, manager.Name)
			text = strings.ReplaceAll(text, match, mention)
		}
	}
	return text
}

// buildListText는 Nested List를 텍스트로 렌더링한다.
func (r *GithubMarkdownConverter) buildListText(node ast.Node) string {
	var buf bytes.Buffer
	r.buildListTextAux(&buf, node, 0)
	return buf.String()
}

func (r *GithubMarkdownConverter) buildListTextAux(buf *bytes.Buffer, node ast.Node, offset int) {
	switch node.Kind() {
	case ast.KindList:
		l := node.(*ast.List)
		for child := node.FirstChild(); child != nil; child = child.NextSibling() {
			for i := 0; i < offset; i += 1 {
				buf.WriteRune(' ')
			}
			buf.WriteByte(l.Marker)
			r.buildListTextAux(buf, child, offset)

			if child != node.LastChild() {
				buf.WriteRune('\n')
			}
		}

	case ast.KindListItem:
		li := node.(*ast.ListItem)
		for child := node.FirstChild(); child != nil; child = child.NextSibling() {
			if child.IsRaw() {
				buf.WriteString(r.buildText(node, true))
			} else {
				r.buildListTextAux(buf, child, offset+li.Offset)
			}

			if child != node.LastChild() {
				buf.WriteRune('\n')
			}
		}

	default:
		buf.WriteString(r.buildText(node, true))
	}
}
