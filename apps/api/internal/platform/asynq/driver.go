package redis

import (
	"context"
	"time"

	"github.com/artumont/dotslashstream/internal/platform"
	"github.com/hibiken/asynq"
)

type AsynqClientAdapterDriver struct {
	client *asynq.Client
}

var _ platform.QueueClient = (*AsynqClientAdapterDriver)(nil)

func New(redisAddr string) *AsynqClientAdapterDriver {
	return &AsynqClientAdapterDriver{
		client: asynq.NewClient(asynq.RedisClientOpt{Addr: redisAddr}),
	}
}

func (a *AsynqClientAdapterDriver) Enqueue(ctx context.Context, t *platform.Task, delay time.Duration) error {
	asynqTask := asynq.NewTask(t.Type, t.Payload)

	var opts []asynq.Option
	if delay > 0 {
		opts = append(opts, asynq.ProcessIn(delay))
	}

	_, err := a.client.EnqueueContext(ctx, asynqTask, opts...)
	return err
}

func (a *AsynqClientAdapterDriver) Close() error {
	return a.client.Close()
}
