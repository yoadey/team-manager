package push_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yoadey/team-manager/backend/internal/push"
)

func TestFakePusher_RecordsLastSend(t *testing.T) {
	t.Parallel()

	p := push.NewFakePusher(nil)
	sub := push.Subscription{Endpoint: "https://push.example/abc", P256dh: "p", AuthKey: "a"}
	payload := push.Payload{Title: "Neues Training", Body: "Details"}

	require.NoError(t, p.Send(context.Background(), sub, payload))

	endpoint, gotPayload := p.LastSent()
	assert.Equal(t, sub.Endpoint, endpoint)
	assert.Equal(t, payload, gotPayload)
	assert.Equal(t, 1, p.SentCount())

	require.NoError(t, p.Send(context.Background(), sub, payload))
	assert.Equal(t, 2, p.SentCount())
}
