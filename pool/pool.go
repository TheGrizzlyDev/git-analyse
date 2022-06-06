package pool

import "context"

type Pool struct {
	jobs  int
	queue chan struct{}
}

func NewPool(size int) *Pool {
	return &Pool{
		jobs:  size,
		queue: make(chan struct{}, size),
	}
}

func (p *Pool) Enqueue(ctx context.Context, job func()) {
	ctx, cancel := context.WithCancel(ctx)
	p.queue <- struct{}{}
	go func() {
		<- ctx.Done()
		<- p.queue
	}()
	go func() {
		job()
		cancel()
	}()
}

func ForEach[T any](ctx context.Context, p *Pool, values []T, consumer func(T)) {
	for i := 0; i < len(values); i++ {
		value := values[i]
		p.Enqueue(ctx, func() {
			consumer(value)
		})
	}
}
