package appstore

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/channel-io/cht-app-github/internal/channel/model"
)

type nativeFuntcionParams interface {
	Method() string
}

type nativeFunctionRequest struct {
	Method string               `json:"method"`
	Params nativeFuntcionParams `json:"params"`
}

type nativeFunctionResponse struct {
	Result json.RawMessage           `json:"result"`
	Error  nativeFunctionErrorDetail `json:"error"`
}

type nativeFunctionErrorDetail struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

type WriteGroupMessageRequest struct {
	ChannelID     string          `json:"channelId"`
	GroupID       string          `json:"groupId"`
	RootMessageID string          `json:"rootMessageId"`
	Broadcast     bool            `json:"broadcast"`
	DTO           GroupMessageDTO `json:"dto"`
}

func (w *WriteGroupMessageRequest) Method() string {
	return "writeGroupMessage"
}

type GroupMessageDTO struct {
	Blocks    []model.MessageBlock `json:"blocks"`
	RequestId string               `json:"requestId"`
	BotName   string               `json:"botName"`
}

type WriteGroupMessageResponse struct {
	Message struct {
		ID string `json:"id"`
	} `json:"message"`
}

type SearchManagersRequest struct {
	ChannelID  string     `json:"channelId"`
	Pagination Pagination `json:"pagination"`
}

func (s *SearchManagersRequest) Method() string {
	return "searchManagers"
}

type Pagination struct {
	SortOrder int    `json:"sortOrder"`
	Since     string `json:"since"`
	Limit     int    `json:"limit"`
}

type GetManagerRequest struct {
	ChannelID string `json:"channelId"`
	ManagerID string `json:"managerId"`
}

func (s *GetManagerRequest) Method() string {
	return "getManager"
}

const (
	SortOrderAsc  = 1
	SortOrderDesc = 2
	SortOrderBoth = 3
)

type SearchManagersResponse struct {
	Managers []ManagerDTO `json:"managers"`
	Next     string       `json:"next"`
}

type GetManagerResponse struct {
	Manager ManagerDTO `json:"manager"`
}

type ManagerDTO struct {
	ID      string     `json:"id"`
	Name    string     `json:"name"`
	Email   *string    `json:"email"`
	Profile profileDTO `json:"profile"`
}

func (m *ManagerDTO) ToModel() model.Manager {
	return model.Manager{
		ID:                 m.ID,
		Name:               m.Name,
		Email:              m.Email,
		GithubUsername:     m.Profile.GithubUsername,
		GithubOrganization: m.Profile.GithubOrganization,
	}
}

type profileDTO struct {
	GithubUsername     *string `json:"github-username"`
	GithubOrganization *string `json:"github-organization"`
}

type errorResponse struct {
	Type     string        `json:"type"`
	Status   int           `json:"status"`
	Language string        `json:"language"`
	Errors   []errorDetail `json:"errors"`
}

func (e errorResponse) Error() string {
	var msg string
	if len(e.Errors) > 0 {
		msg = e.Errors[0].Message
	}
	return fmt.Sprintf("%d %s: %s", e.Status, e.Type, msg)
}

type errorDetail struct {
	Message string `json:"message"`
}

type issueTokenRequest struct {
	Secret    string  `json:"secret"`
	ChannelId *string `json:"channelId"`
}

func (r *issueTokenRequest) Method() string {
	return "issueToken"
}

type issueTokenResponse struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
	ExpiresIn    int64  `json:"expiresIn"`

	Expiry time.Time
}

type refreshIssueTokenRequest struct {
	RefreshToken string `json:"refreshToken"`
}

func (r *refreshIssueTokenRequest) Method() string {
	return "issueToken"
}
