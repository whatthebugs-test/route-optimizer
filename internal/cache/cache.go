// Package cache memoises solved plans in Redis so dispatchers refreshing the
// board don't re-run the solver for an unchanged stop set.
package cache

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/kestrelfreight/route-optimizer/internal/solver"
)

const ttl = 15 * time.Minute

type Cache struct {
	rdb *redis.Client
}

func New(url string) (*Cache, error) {
	opts, err := redis.ParseURL(url)
	if err != nil {
		return nil, err
	}
	return &Cache{rdb: redis.NewClient(opts)}, nil
}

// Key hashes everything that affects the solution.
func Key(terminal string, stops []solver.Stop, opts solver.Options) string {
	h := sha256.New()
	_ = json.NewEncoder(h).Encode(struct {
		T string
		S []solver.Stop
		O solver.Options
	}{terminal, stops, opts})
	return "routeplan:" + hex.EncodeToString(h.Sum(nil))[:32]
}

func (c *Cache) Get(ctx context.Context, key string) ([]solver.Route, bool) {
	raw, err := c.rdb.Get(ctx, key).Bytes()
	if err != nil {
		return nil, false
	}
	var routes []solver.Route
	if json.Unmarshal(raw, &routes) != nil {
		return nil, false
	}
	return routes, true
}

func (c *Cache) Put(ctx context.Context, key string, routes []solver.Route) error {
	raw, err := json.Marshal(routes)
	if err != nil {
		return err
	}
	return c.rdb.Set(ctx, key, raw, ttl).Err()
}

func (c *Cache) Ping(ctx context.Context) error {
	if c == nil {
		return errors.New("cache disabled")
	}
	return c.rdb.Ping(ctx).Err()
}
