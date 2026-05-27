package redis

import (
	"context"
	"github.com/zloevil/jet"
	"time"
)

// Lock applies distributed lock
// unlockId allows only process issuing lock to unlock it
func (r *Redis) Lock(ctx context.Context, key, unlockId string, ttl time.Duration) (bool, error) {
	l := r.l().C(ctx).Mth("lock").F(jet.KV{"key": key}).Dbg()
	// SetNX sets key value if it's not exists
	// it returns true, if value is set and false otherwise
	res := r.Instance.SetNX(ctx, key, unlockId, ttl)
	if res.Err() != nil {
		return false, res.Err()
	}
	l.Dbg("locked: ", res.Val())
	return res.Val(), nil
}

// UnLock unlocks
func (r *Redis) UnLock(ctx context.Context, key, unlockId string) (bool, error) {
	r.l().C(ctx).Mth("unlock").F(jet.KV{"key": key}).Dbg()
	// allows checking and unlocking as one hit
	res := r.Instance.Eval(ctx, `
		if redis.call("GET", KEYS[1]) == ARGV[1] then
			return redis.call("DEL", KEYS[1])
		else
			return 0
		end`, []string{key}, unlockId)
	if res.Err() != nil {
		return false, res.Err()
	}
	cnt, _ := res.Int()
	return cnt > 0, nil

}
