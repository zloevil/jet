package redis

import (
	"context"
	"encoding/json"
	"github.com/redis/go-redis/v9"
	"github.com/zloevil/jet"
	"strconv"
)

type PriorityQueue[T any] interface {
	// Push pushes item to queue with the given priority
	Push(ctx context.Context, item T, priority int) error
	// PopManyAndRemove pops requested number of items according to priority and remove them atomically
	PopManyAndRemove(ctx context.Context, numItems int) ([]T, error)
	// Pop pops requested number of items
	Pop(ctx context.Context, numItems int) ([]T, error)
}

type priorityQueue[T any] struct {
	queue string
	redis *Redis
}

func NewPriorityQueue[T any](queue string, redis *Redis) PriorityQueue[T] {
	return &priorityQueue[T]{
		queue: queue,
		redis: redis,
	}
}

type qItem[T any] struct {
	Id  string `json:"id"`
	Val T      `json:"val"`
}

func (p *priorityQueue[T]) Push(ctx context.Context, item T, priority int) error {

	itemJs, _ := json.Marshal(&qItem[T]{jet.NewRandString(), item})

	err := p.redis.Instance.ZAdd(ctx, p.queue, redis.Z{Score: float64(priority), Member: itemJs}).Err()
	if err != nil {
		return ErrRedisPriorityQueuePushErr(ctx, err)
	}

	return nil

}

func (p *priorityQueue[T]) PopManyAndRemove(ctx context.Context, numItems int) ([]T, error) {

	stop := numItems - 1

	if stop <= 0 {
		stop = 1
	}

	rs := p.redis.Instance.Eval(ctx, `
        local items = redis.call('ZRANGE', KEYS[1], ARGV[1], ARGV[2])
        if #items > 0 then
            redis.call('ZREM', KEYS[1], unpack(items))
        end
        return items`, []string{p.queue}, strconv.Itoa(0), strconv.Itoa(stop))
	if rs.Err() != nil {
		return nil, ErrRedisPriorityQueuePopRemoveErr(ctx, rs.Err())
	}

	res, err := rs.Result()
	if err != nil {
		return nil, ErrRedisPriorityQueuePopRemoveErr(ctx, err)
	}
	itemsAny, ok := res.([]any)
	if !ok {
		return nil, ErrRedisPriorityQueuePopRemoveErr(ctx, nil)
	}
	items := jet.Map(itemsAny, func(i any) string { return i.(string) })

	return jet.Map(items, func(item string) T {
		var res *qItem[T]
		_ = json.Unmarshal([]byte(item), &res)
		return res.Val
	}), nil
}

func (p *priorityQueue[T]) Pop(ctx context.Context, numItems int) ([]T, error) {

	stop := int64(numItems - 1)

	if stop <= 0 {
		stop = 1
	}

	rs := p.redis.Instance.ZRange(ctx, p.queue, 0, stop)
	if rs.Err() != nil {
		return nil, ErrRedisPriorityQueuePopErr(ctx, rs.Err())
	}

	res, err := rs.Result()
	if err != nil {
		return nil, ErrRedisPriorityQueuePopErr(ctx, err)
	}

	return jet.Map(res, func(item string) T {
		var res *qItem[T]
		_ = json.Unmarshal([]byte(item), &res)
		return res.Val
	}), nil

}
