package ingress

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func TestValKeyPacketQueueEnqueueOncePerDelivery(t *testing.T) {
	ctx := context.Background()
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	t.Cleanup(func() {
		require.NoError(t, client.Close())
	})

	queue := NewValKeyPacketQueue(client, ValKeyPacketQueueConfig{
		StreamKey:      "stream:github-events",
		DedupKeyPrefix: "dedup:delivery:",
		DedupTTL:       7 * 24 * time.Hour,
	})
	packet := WebhookPacket{
		DeliveryID: "delivery-1",
		EventName:  "issues",
		Action:     "opened",
		Org:        "channel-io",
		Repo:       "cht-app-github",
		Number:     1,
		ReceivedAt: time.Unix(10, 0).UTC(),
		Payload:    []byte(`{"action":"opened"}`),
	}

	first, err := queue.Enqueue(ctx, packet)
	require.NoError(t, err)
	require.False(t, first.Duplicate)
	require.NotEmpty(t, first.StreamEntryID)

	dedupValue, err := client.Get(ctx, "dedup:delivery:delivery-1").Result()
	require.NoError(t, err)
	require.Equal(t, first.StreamEntryID, dedupValue)

	streamEntries, err := client.XRange(ctx, "stream:github-events", "-", "+").Result()
	require.NoError(t, err)
	require.Len(t, streamEntries, 1)
	require.Equal(t, first.StreamEntryID, streamEntries[0].ID)
	require.Contains(t, streamEntries[0].Values, "packet")

	second, err := queue.Enqueue(ctx, packet)
	require.NoError(t, err)
	require.True(t, second.Duplicate)
	require.Equal(t, first.StreamEntryID, second.StreamEntryID)

	streamEntries, err = client.XRange(ctx, "stream:github-events", "-", "+").Result()
	require.NoError(t, err)
	require.Len(t, streamEntries, 1)
}

func TestValKeyPacketQueueDefaultsUseClusterSafeHashTag(t *testing.T) {
	queue := NewValKeyPacketQueue(nil, ValKeyPacketQueueConfig{})

	require.Equal(t, "stream:{github-events}:ingress", queue.config.StreamKey)
	require.Equal(t, "dedup:{github-events}:delivery:", queue.config.DedupKeyPrefix)
	require.Equal(t, 7*24*time.Hour, queue.config.DedupTTL)
}

func TestValKeyPacketQueueRejectsMissingDeliveryID(t *testing.T) {
	ctx := context.Background()
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	t.Cleanup(func() {
		require.NoError(t, client.Close())
	})

	queue := NewValKeyPacketQueue(client, ValKeyPacketQueueConfig{
		StreamKey:      "stream:github-events",
		DedupKeyPrefix: "dedup:delivery:",
		DedupTTL:       time.Hour,
	})

	_, err := queue.Enqueue(ctx, WebhookPacket{})
	require.Error(t, err)
	require.True(t, errors.Is(err, ErrMissingDeliveryID))
}
