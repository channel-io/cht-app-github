package appstore

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/pkg/errors"
)

func NewClient(baseURL, secret string) *Client {
	return &Client{
		secret: secret,
		rest:   resty.New().SetBaseURL(baseURL),
		cache:  new(sync.Map),
	}
}

type Client struct {
	secret string
	rest   *resty.Client
	cache  *sync.Map
}

func (c *Client) issueToken(ctx context.Context, channelID *string) (*issueTokenResponse, error) {
	funcRes, err := c.invokeNativeFunction(ctx, "", &issueTokenRequest{
		Secret:    c.secret,
		ChannelId: channelID,
	})
	if err != nil {
		return nil, err
	}

	var res issueTokenResponse
	if err = json.Unmarshal(funcRes, &res); err != nil {
		return nil, err
	}

	res.Expiry = time.Now().Add(time.Duration(res.ExpiresIn)*time.Second - 20*time.Second) // with buffer
	return &res, nil
}

func (c *Client) getAccessToken(ctx context.Context, channelID string) (string, error) {
	// 1. from local cache
	cacheKey := fmt.Sprintf("auth:%s", channelID)
	token, ok := c.cache.Load(cacheKey)

	if ok {
		token := token.(issueTokenResponse)
		if time.Now().Before(token.Expiry) {
			return token.AccessToken, nil
		}

		// 2. refresh token
		res, err := c.refreshIssueToken(ctx, token.RefreshToken)
		if err == nil {
			c.cache.Store(cacheKey, *res)
			return res.AccessToken, nil
		}
	}

	// 3. issue token
	res, err := c.issueToken(ctx, &channelID)
	if err != nil {
		return "", err
	}

	c.cache.Store(cacheKey, *res)
	return res.AccessToken, nil
}

func (c *Client) refreshIssueToken(ctx context.Context, refreshToken string) (*issueTokenResponse, error) {
	funcRes, err := c.invokeNativeFunction(ctx, "", &refreshIssueTokenRequest{
		RefreshToken: refreshToken,
	})
	if err != nil {
		return nil, err
	}

	var res issueTokenResponse
	if err = json.Unmarshal(funcRes, &res); err != nil {
		return nil, err
	}

	res.Expiry = time.Now().Add(time.Duration(res.ExpiresIn)*time.Second - 20*time.Second) // with buffer
	return &res, nil
}

const nativeFunctionPath = "/general/v1/native/functions"

func (c *Client) invokeNativeFunction(
	ctx context.Context,
	accessToken string,
	params nativeFuntcionParams,
) (json.RawMessage, error) {
	body := nativeFunctionRequest{
		Method: params.Method(),
		Params: params,
	}

	var res nativeFunctionResponse
	var apiErr errorResponse
	r := c.rest.R().
		SetContext(ctx).
		SetHeader("Content-Type", "application/json").
		SetHeader("x-access-token", accessToken).
		SetBody(body).
		SetResult(&res).
		SetError(&apiErr)

	resp, err := r.Put(nativeFunctionPath)
	if err != nil {
		return nil, err
	}

	if err != nil {
		return nil, err
	}

	if resp.IsError() {
		return nil, apiErr
	}

	if res.Error.Type != "" {
		return nil, errors.Errorf("%s: %s", res.Error.Type, res.Error.Message)
	}

	return res.Result, nil
}
