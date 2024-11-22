package model

import (
	"bytes"
	"encoding/json"

	"github.com/pkg/errors"
)

type BlockType string

const (
	BlockTypeText    BlockType = "text"
	BlockTypeCode    BlockType = "code"
	BlockTypeBullets BlockType = "bullets"
)

func NewTextBlock(s string) MessageBlock {
	return MessageBlock{
		Type: BlockTypeText,
		Text: Text{
			Value: s,
		},
	}
}

func NewCodeBlock(value string, language *string) MessageBlock {
	return MessageBlock{
		Type: BlockTypeCode,
		Code: Code{
			Language: language,
			Value:    value,
		},
	}
}

func NewBulletsBlock(textBlocks []MessageBlock) MessageBlock {
	return MessageBlock{
		Type: BlockTypeBullets,
		Bullets: Bullets{
			Blocks: textBlocks,
		},
	}
}

// Block := Text | Code | Bullets
type MessageBlock struct {
	Type BlockType

	Text    Text    // Populated if Type is Text
	Code    Code    // Populated if Type is Code
	Bullets Bullets // Populated if Type is Bullets
}

func (b MessageBlock) MarshalJSON() ([]byte, error) {
	m := map[string]interface{}{
		"type": b.Type,
	}
	switch b.Type {
	case BlockTypeText:
		m["value"] = b.Text.Value

	case BlockTypeCode:
		if b.Code.Language != nil {
			m["language"] = *b.Code.Language
		}
		m["value"] = b.Code.Value

	case BlockTypeBullets:
		m["blocks"] = b.Bullets.Blocks

	default:
		return nil, errors.New("unknown block type")
	}

	return marshalJSONWithoutEscapeHTML(m)
}

// Text := { type: "text", value: ANTLRString }
type Text struct {
	Value string
}

// Code := { type: "code", language: String | null, value: String }
type Code struct {
	Language *string
	Value    string
}

// Bullets := { type: "bullets", blocks: [Text] }
type Bullets struct {
	Blocks []MessageBlock
}

// Note: cannot use json.Marshaler as it always escapes HTML characters (<, >, &)
// e.g. "<b>" encodes to "\u003cb\u003e"
// https://pkg.go.dev/encoding/json#Marshal
func marshalJSONWithoutEscapeHTML(m any) ([]byte, error) {
	var buffer bytes.Buffer
	encoder := json.NewEncoder(&buffer)
	encoder.SetEscapeHTML(false)
	err := encoder.Encode(m)
	if err != nil {
		return nil, err
	}
	return bytes.TrimRight(buffer.Bytes(), "\n"), nil
}
