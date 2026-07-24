package jobs

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/riverqueue/river"
	"go.opentelemetry.io/otel/codes"

	"github.com/yoadey/team-manager/backend/internal/metrics"
	"github.com/yoadey/team-manager/backend/internal/push"
)

// PushDeliveryArgs carries everything needed to deliver a single Web Push
// notification -- the subscription's own fields are denormalized onto the
// job (rather than re-queried by SubscriptionID at delivery time) so a
// subscription deleted between enqueue and delivery doesn't turn into a
// lookup-miss error; the send is simply attempted against the endpoint as it
// was at enqueue time, and fails (ErrGone) the same way a genuinely stale
// endpoint would.
type PushDeliveryArgs struct {
	SubscriptionID uuid.UUID `json:"subscription_id"`
	Endpoint       string    `json:"endpoint"`
	P256dh         string    `json:"p256dh"`
	AuthKey        string    `json:"auth_key"`
	Title          string    `json:"title"`
	Body           string    `json:"body"`
	URL            string    `json:"url,omitempty"`
}

// Kind implements river.JobArgs.
func (PushDeliveryArgs) Kind() string { return "push_delivery" }

// pushSubscriptionDeleter is satisfied by *push.Repository.
type pushSubscriptionDeleter interface {
	DeleteByID(ctx context.Context, id uuid.UUID) error
}

// PushDeliveryWorker sends a single Web Push notification and prunes its
// subscription if the push service reports it gone.
type PushDeliveryWorker struct {
	river.WorkerDefaults[PushDeliveryArgs]
	pusher push.Pusher
	repo   pushSubscriptionDeleter
}

// NewPushDeliveryWorker constructs a PushDeliveryWorker.
func NewPushDeliveryWorker(pusher push.Pusher, repo pushSubscriptionDeleter) *PushDeliveryWorker {
	return &PushDeliveryWorker{pusher: pusher, repo: repo}
}

const pushDeliveryTimeout = 15 * time.Second

// Work sends the push notification described by job.Args. A push.ErrGone
// response prunes the subscription and is treated as handled (no retry --
// retrying a permanently gone endpoint is pointless). Any other failure is
// returned so River's built-in retry/backoff applies.
func (w *PushDeliveryWorker) Work(ctx context.Context, job *river.Job[PushDeliveryArgs]) (err error) {
	ctx, span := tracer.Start(ctx, "push_delivery.work")
	defer func() {
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		}
		span.End()
	}()

	ctx, cancel := context.WithTimeout(ctx, pushDeliveryTimeout)
	defer cancel()

	a := job.Args
	sendErr := w.pusher.Send(ctx, push.Subscription{
		Endpoint: a.Endpoint,
		P256dh:   a.P256dh,
		AuthKey:  a.AuthKey,
	}, push.Payload{Title: a.Title, Body: a.Body, URL: a.URL})

	if sendErr == nil {
		metrics.PushDeliverySuccess.Inc()
		return nil
	}

	if errors.Is(sendErr, push.ErrGone) {
		if delErr := w.repo.DeleteByID(ctx, a.SubscriptionID); delErr != nil {
			return fmt.Errorf("jobs.PushDeliveryWorker: prune gone subscription: %w", delErr)
		}
		metrics.PushSubscriptionsPruned.Inc()
		return nil
	}

	metrics.PushDeliveryFailures.Inc()
	return fmt.Errorf("jobs.PushDeliveryWorker: %w", sendErr)
}
