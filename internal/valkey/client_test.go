package valkey

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/channel-io/cht-app-github/internal/config"
)

func TestRepositoryEnabled(t *testing.T) {
	require.False(t, NewRepository(nil).Enabled())
}

func TestNewUniversalOptions(t *testing.T) {
	cfg := new(config.Config)
	cfg.Valkey.URL = "localhost:6379"
	cfg.Valkey.ClusterMode = true

	options := NewUniversalOptions(cfg)

	require.Equal(t, []string{"localhost:6379"}, options.Addrs)
	require.True(t, options.IsClusterMode)
}
