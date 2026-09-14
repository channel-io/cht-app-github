package ingress

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

var ErrMissingDeliveryID = errors.New("missing github delivery id")

type PacketQueue interface {
	Enqueue(ctx context.Context, packet WebhookPacket) (EnqueueResult, error)
}

type ValKeyPacketQueueConfig struct {
	StreamKey      string
	DedupKeyPrefix string
	DedupTTL       time.Duration
}

type ValKeyPacketQueue struct {
	client redis.UniversalClient
	config ValKeyPacketQueueConfig
}

func NewValKeyPacketQueue(client redis.UniversalClient, config ValKeyPacketQueueConfig) *ValKeyPacketQueue {
	if config.StreamKey == "" {
		config.StreamKey = "stream:{github-events}:ingress"
	}
	if config.DedupKeyPrefix == "" {
		config.DedupKeyPrefix = "dedup:{github-events}:delivery:"
	}
	if config.DedupTTL == 0 {
		config.DedupTTL = 7 * 24 * time.Hour
	}

	return &ValKeyPacketQueue{
		client: client,
		config: config,
	}
}

func (q *ValKeyPacketQueue) Enqueue(ctx context.Context, packet WebhookPacket) (EnqueueResult, error) {
	if packet.DeliveryID == "" {
		return EnqueueResult{}, ErrMissingDeliveryID
	}

	payload, err := json.Marshal(packet)
	if err != nil {
		return EnqueueResult{}, fmt.Errorf("marshal webhook packet: %w", err)
	}

	result, err := atomicEnqueueScript.Run(
		ctx,
		q.client,
		[]string{q.dedupKey(packet.DeliveryID), q.config.StreamKey},
		strconv.FormatInt(int64(q.config.DedupTTL/time.Second), 10),
		string(payload),
	).Result()
	if err != nil {
		return EnqueueResult{}, fmt.Errorf("valkey enqueue packet: %w", err)
	}

	return parseAtomicEnqueueResult(result)
}

func (q *ValKeyPacketQueue) dedupKey(deliveryID string) string {
	return q.config.DedupKeyPrefix + deliveryID
}

var atomicEnqueueScript = redis.NewScript(`
local dedupKey = KEYS[1]
local streamKey = KEYS[2]
local ttlSeconds = tonumber(ARGV[1])
local packet = ARGV[2]

if redis.call("EXISTS", dedupKey) == 1 then
  local existing = redis.call("GET", dedupKey)
  if existing == false then
    existing = ""
  end
  return {1, existing}
end

local streamEntryID = redis.call("XADD", streamKey, "*", "packet", packet)
redis.call("SET", dedupKey, streamEntryID, "EX", ttlSeconds)
return {0, streamEntryID}
`)

func parseAtomicEnqueueResult(raw any) (EnqueueResult, error) {
	values, ok := raw.([]any)
	if !ok || len(values) != 2 {
		return EnqueueResult{}, fmt.Errorf("unexpected valkey enqueue result: %v", raw)
	}

	duplicateFlag, ok := values[0].(int64)
	if !ok {
		return EnqueueResult{}, fmt.Errorf("unexpected valkey duplicate flag: %v", values[0])
	}

	streamEntryID, ok := values[1].(string)
	if !ok {
		return EnqueueResult{}, fmt.Errorf("unexpected valkey stream id: %v", values[1])
	}

	return EnqueueResult{
		Duplicate:     duplicateFlag == 1,
		StreamEntryID: streamEntryID,
	}, nil
}
