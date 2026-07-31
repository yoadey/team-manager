package jobs

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPushPayloadForNotification_TruncatesLongBody(t *testing.T) {
	t.Parallel()

	longQuestion := strings.Repeat("a", maxPushBodyRunes+50)
	payload := pushPayloadForNotification(NotificationArgs{Type: "poll", Title: &longQuestion})

	assert.Equal(t, maxPushBodyRunes+1, len([]rune(payload.Body)), "truncated body plus the ellipsis rune")
	assert.True(t, strings.HasSuffix(payload.Body, "…"))
	assert.Equal(t, strings.Repeat("a", maxPushBodyRunes), strings.TrimSuffix(payload.Body, "…"))
}

func TestPushPayloadForNotification_LeavesShortBodyUnchanged(t *testing.T) {
	t.Parallel()

	question := "Wer kommt zum Training?"
	payload := pushPayloadForNotification(NotificationArgs{Type: "poll", Title: &question})

	assert.Equal(t, question, payload.Body)
}

func TestPushPayloadForNotification_TruncatesMultiByteRunesWithoutSplitting(t *testing.T) {
	t.Parallel()

	longNote := strings.Repeat("ü", maxPushBodyRunes+10)
	payload := pushPayloadForNotification(NotificationArgs{Type: "news", Note: &longNote})

	runes := []rune(payload.Body)
	assert.Equal(t, maxPushBodyRunes+1, len(runes), "truncated body plus the ellipsis rune")
	assert.Equal(t, '…', runes[len(runes)-1])
}
