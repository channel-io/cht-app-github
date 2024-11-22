package appstore

import (
	"context"
	"encoding/json"
)

func (c *Client) WriteGroupMessage(
	ctx context.Context,
	req *WriteGroupMessageRequest,
) (*WriteGroupMessageResponse, error) {
	token, err := c.getAccessToken(ctx, req.ChannelID)
	if err != nil {
		return nil, err
	}

	r, err := c.invokeNativeFunction(ctx, token, req)
	if err != nil {
		return nil, err
	}

	var res WriteGroupMessageResponse
	if err := json.Unmarshal(r, &res); err != nil {
		return nil, err
	}

	return &res, nil
}

func (c *Client) SearchManagers(ctx context.Context, req *SearchManagersRequest) (*SearchManagersResponse, error) {
	token, err := c.getAccessToken(ctx, req.ChannelID)
	if err != nil {
		return nil, err
	}

	r, err := c.invokeNativeFunction(ctx, token, req)
	if err != nil {
		return nil, err
	}

	var res SearchManagersResponse
	if err := json.Unmarshal(r, &res); err != nil {
		return nil, err
	}

	return &res, nil
}

func (c *Client) GetManager(ctx context.Context, req *GetManagerRequest) (*GetManagerResponse, error) {
	token, err := c.getAccessToken(ctx, req.ChannelID)
	if err != nil {
		return nil, err
	}

	r, err := c.invokeNativeFunction(ctx, token, req)
	if err != nil {
		return nil, err
	}

	var res GetManagerResponse
	if err := json.Unmarshal(r, &res); err != nil {
		return nil, err
	}

	return &res, nil
}
