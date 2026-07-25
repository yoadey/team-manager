package push_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yoadey/team-manager/backend/internal/push"
)

func TestNewWebPusher_RequiresVAPIDKeys(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		cfg  push.VAPIDConfig
	}{
		{"both empty", push.VAPIDConfig{}},
		{"missing private key", push.VAPIDConfig{PublicKey: "pub", Subject: "mailto:ops@example.com"}},
		{"missing public key", push.VAPIDConfig{PrivateKey: "priv", Subject: "mailto:ops@example.com"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			_, err := push.NewWebPusher(c.cfg)
			require.Error(t, err)
			assert.ErrorIs(t, err, push.ErrVAPIDKeysRequired)
		})
	}
}

func TestNewWebPusher_ValidConfig(t *testing.T) {
	t.Parallel()

	p, err := push.NewWebPusher(push.VAPIDConfig{
		PublicKey:  "pub",
		PrivateKey: "priv",
		Subject:    "mailto:ops@example.com",
	})
	require.NoError(t, err)
	assert.NotNil(t, p)
}
