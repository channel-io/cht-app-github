package appstore

import "encoding/json"

const (
	AppCallerType     = "app"
	ManagerCallerType = "manager"
	UserCallerType    = "user"
)

type Context struct {
	Caller  Caller  `json:"caller"`
	Channel Channel `json:"channel"`
}

type Channel struct {
	ID string `json:"id"`
}

type Caller struct {
	Type string `json:"type"`
	ID   string `json:"id"`
}

type Chat struct {
	Type string `json:"type"`
	ID   string `json:"id"`
}

type Trigger struct {
	Type       string            `json:"type"`
	Attributes map[string]string `json:"attributes"`
}

type CommandParams struct {
	Chat    Chat            `json:"chat"`
	Trigger Trigger         `json:"trigger"`
	Input   json.RawMessage `json:"input"` // function params; slice or object
}

type FunctionResult struct {
	Type       string          `json:"type"`
	Attributes json.RawMessage `json:"attributes"`
}

type JsonFunctionRequest struct {
	Method  string        `json:"method"`
	Params  CommandParams `json:"params"`
	Context Context       `json:"context"`
}

type JsonFunctionResponse struct {
	Error  *Error          `json:"error,omitempty"`
	Result json.RawMessage `json:"result,omitempty"`
}

type Error struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}
