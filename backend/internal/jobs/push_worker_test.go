package jobs_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	promtestutil "github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yoadey/team-manager/backend/internal/jobs"
	"github.com/yoadey/team-manager/backend/internal/metrics"
	"github.com/yoadey/team-manager/backend/internal/push"
)

type mockPusher struct {
	sendFn func(ctx context.Context, sub push.Subscription, payload push.Payload) error
}

func (m *mockPusher) Send(ctx context.Context, sub push.Subscription, payload push.Payload) error {
	return m.sendFn(ctx, sub, payload)
}

type mockDeleter struct {
	deleteByIDFn func(ctx context.Context, id uuid.UUID) error
}

func (m *mockDeleter) DeleteByID(ctx context.Context, id uuid.UUID) error {
	return m.deleteByIDFn(ctx, id)
}

func TestPushDeliveryWorker_Work_Success(t *testing.T) {
	t.Parallel()

	before := promtestutil.ToFloat64(metrics.PushDeliverySuccess)

	var gotSub push.Subscription
	pusher := &mockPusher{sendFn: func(_ context.Context, sub push.Subscription, _ push.Payload) error {
		gotSub = sub
		return nil
	}}
	deleter := &mockDeleter{deleteByIDFn: func(context.Context, uuid.UUID) error {
		t.Fatal("must not delete the subscription on a successful send")
		return nil
	}}

	worker := jobs.NewPushDeliveryWorker(pusher, deleter)
	subID := uuid.New()
	job := &river.Job[jobs.PushDeliveryArgs]{
		JobRow: &rivertype.JobRow{ID: 1},
		Args: jobs.PushDeliveryArgs{
			SubscriptionID: subID,
			Endpoint:       "https://push.example/abc",
			P256dh:         "p256dh",
			AuthKey:        "auth",
			Title:          "Neues Training",
			Body:           "Details",
		},
	}

	require.NoError(t, worker.Work(context.Background(), job))
	assert.Equal(t, "https://push.example/abc", gotSub.Endpoint)
	assert.Equal(t, before+1, promtestutil.ToFloat64(metrics.PushDeliverySuccess))
}

func TestPushDeliveryWorker_Work_PrunesGoneSubscription(t *testing.T) {
	t.Parallel()

	before := promtestutil.ToFloat64(metrics.PushSubscriptionsPruned)

	pusher := &mockPusher{sendFn: func(context.Context, push.Subscription, push.Payload) error {
		return push.ErrGone
	}}
	var deletedID uuid.UUID
	deleter := &mockDeleter{deleteByIDFn: func(_ context.Context, id uuid.UUID) error {
		deletedID = id
		return nil
	}}

	worker := jobs.NewPushDeliveryWorker(pusher, deleter)
	subID := uuid.New()
	job := &river.Job[jobs.PushDeliveryArgs]{
		JobRow: &rivertype.JobRow{ID: 2},
		Args:   jobs.PushDeliveryArgs{SubscriptionID: subID},
	}

	require.NoError(t, worker.Work(context.Background(), job), "a gone subscription is handled, not an error to retry")
	assert.Equal(t, subID, deletedID)
	assert.Equal(t, before+1, promtestutil.ToFloat64(metrics.PushSubscriptionsPruned))
}

func TestPushDeliveryWorker_Work_TransientFailureIsRetried(t *testing.T) {
	t.Parallel()

	before := promtestutil.ToFloat64(metrics.PushDeliveryFailures)

	wantErr := errors.New("push service unavailable")
	pusher := &mockPusher{sendFn: func(context.Context, push.Subscription, push.Payload) error {
		return wantErr
	}}
	deleter := &mockDeleter{deleteByIDFn: func(context.Context, uuid.UUID) error {
		t.Fatal("must not prune the subscription on a transient failure")
		return nil
	}}

	worker := jobs.NewPushDeliveryWorker(pusher, deleter)
	job := &river.Job[jobs.PushDeliveryArgs]{
		JobRow: &rivertype.JobRow{ID: 3},
		Args:   jobs.PushDeliveryArgs{SubscriptionID: uuid.New()},
	}

	err := worker.Work(context.Background(), job)
	require.Error(t, err, "a transient failure must be returned so River retries")
	assert.Equal(t, before+1, promtestutil.ToFloat64(metrics.PushDeliveryFailures))
}
