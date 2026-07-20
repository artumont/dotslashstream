package redis

import (
	"context"
	"time"

	"github.com/hibiken/asynq"
)

// AsynqClientAdapter adapts hibiken/asynq.Client to our interface
type AsynqClientAdapter struct {
	client *asynq.Client
}

func NewAsyncqClient(redisAddr string) *AsynqClientAdapter {
	return &AsynqClientAdapter{
		client: asynq.NewClient(asynq.RedisClientOpt{Addr: redisAddr}),
	}
}

func (a *AsynqClientAdapter) Enqueue(ctx context.Context, t *Task, delay time.Duration) error {
	asynqTask := asynq.NewTask(t.Type, t.Payload)

	var opts []asynq.Option
	if delay > 0 {
		opts = append(opts, asynq.ProcessIn(delay))
	}

	_, err := a.client.EnqueueContext(ctx, asynqTask, opts...)
	return err
}

func (a *AsynqClientAdapter) Close() error {
	return a.client.Close()
}
