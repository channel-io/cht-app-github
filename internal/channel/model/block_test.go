package model

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

const sampleGoCode = `
package main

import "fmt"

func main() {
  fmt.Println("Hello, World!")
}`

func TestBlock_MarshalJSON_By_Unmarshal(t *testing.T) {
	tests := []struct {
		name     string
		block    MessageBlock
		expected map[string]interface{}
	}{
		{
			name: "simple text block",
			block: MessageBlock{
				Type: BlockTypeText,
				Text: Text{
					Value: "hello world",
				},
			},
			expected: map[string]interface{}{
				"type":  "text",
				"value": "hello world",
			},
		},
		{
			name: "simple code block",
			block: MessageBlock{
				Type: BlockTypeCode,
				Code: Code{
					Language: stringPtr("go"),
					Value:    sampleGoCode,
				},
			},
			expected: map[string]interface{}{
				"type":     "code",
				"language": "go",
				"value":    sampleGoCode,
			},
		},
		{
			name: "simple bullets block",
			block: MessageBlock{
				Type: BlockTypeBullets,
				Bullets: Bullets{
					Blocks: []MessageBlock{
						{
							Type: BlockTypeText,
							Text: Text{
								Value: "lorem ipsum",
							},
						},
						{
							Type: BlockTypeText,
							Text: Text{
								Value: "dolor sit amet",
							},
						},
					},
				},
			},
			expected: map[string]interface{}{
				"type": "bullets",
				"blocks": []interface{}{
					map[string]interface{}{
						"type":  "text",
						"value": "lorem ipsum",
					},
					map[string]interface{}{
						"type":  "text",
						"value": "dolor sit amet",
					},
				},
			},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			actual, err := marshalJSONWithoutEscapeHTML(tc.block)
			assert.NoError(t, err)

			var m map[string]interface{}
			err = json.Unmarshal(actual, &m)
			assert.NoError(t, err)

			assert.Equal(t, tc.expected, m)
		})
	}
}

func TestBlocks_MarshalJSON_By_Unmarshal(t *testing.T) {
	type message struct {
		Blocks []MessageBlock `json:"blocks"`
	}

	msg := message{
		Blocks: []MessageBlock{
			{
				Type: BlockTypeText,
				Text: Text{
					Value: "This is " + Bold("bold") + ", " + Italic("italic") + ", and " + Bold(Italic("bold+italic")),
				},
			},
			{
				Type: BlockTypeText,
				Text: Text{
					Mention(MentionTypeManager, "managerId_goes_here", "@username"),
				},
			},
			{
				Type: BlockTypeText,
				Text: Text{
					Value: "This is a url <link type=\"url\">https://channel.io</link>",
				},
			},
			{
				Type: BlockTypeText,
				Text: Text{
					Value: "This is a link " + InlineLink("https://channel.io", "Channel"),
				},
			},
			{
				Type: BlockTypeCode,
				Code: Code{
					Language: nil,
					Value:    "<script>ChannelIO('boot')</script>",
				},
			},
			{
				Type: BlockTypeBullets,
				Bullets: Bullets{
					Blocks: []MessageBlock{
						{
							Type: BlockTypeText,
							Text: Text{
								Value: "Bulleted text goes here",
							},
						},
						{
							Type: BlockTypeText,
							Text: Text{
								Value: "Next bulleted text goes here",
							},
						},
					},
				},
			},
		},
	}

	actual, err := marshalJSONWithoutEscapeHTML(msg)
	assert.NoError(t, err)

	var m map[string]interface{}
	err = json.Unmarshal(actual, &m)
	assert.NoError(t, err)

	// Sample from OpenAPI doc: https://api-doc.channel.io
	expected := map[string]interface{}{
		"blocks": []interface{}{
			map[string]interface{}{
				"type":  "text",
				"value": "This is <b>bold</b>, <i>italic</i>, and <b><i>bold+italic</i></b>",
			},
			map[string]interface{}{
				"type":  "text",
				"value": "<link type=\"manager\" value=\"managerId_goes_here\">@username</link>",
			},
			map[string]interface{}{
				"type":  "text",
				"value": "This is a url <link type=\"url\">https://channel.io</link>",
			},
			map[string]interface{}{
				"type":  "text",
				"value": "This is a link <link type=\"url\" value=\"https://channel.io\">Channel</link>",
			},
			map[string]interface{}{
				"type":  "code",
				"value": "<script>ChannelIO('boot')</script>",
			},
			map[string]interface{}{
				"type": "bullets",
				"blocks": []interface{}{
					map[string]interface{}{
						"type":  "text",
						"value": "Bulleted text goes here",
					},
					map[string]interface{}{
						"type":  "text",
						"value": "Next bulleted text goes here",
					},
				},
			},
		},
	}

	assert.Equal(t, expected, m)
}

func stringPtr(s string) *string {
	return &s
}
