package model

import (
	"context"
	"github.com/dgraph-io/ristretto"
	"github.com/eko/gocache/lib/v4/cache"
	"github.com/eko/gocache/lib/v4/store"
	ristretto_store "github.com/eko/gocache/store/ristretto/v4"
)

var cacheManager *cache.Cache[any]

func init() {
	CacheManager()
}

func CacheManager() *cache.Cache[any] {
	if cacheManager != nil {
		return cacheManager
	}

	// https://github.com/dgraph-io/ristretto
	ristrettoCache, err := ristretto.NewCache(&ristretto.Config{
		NumCounters: 10000,
		MaxCost:     (1 << 30) / 2, // 512MB???
		BufferItems: 64,
		// NumCounters: 1e7,     // number of keys to track frequency of (10M).
		// MaxCost:     1 << 30, // maximum cost of cache (1GB).
		// BufferItems: 64,      // number of keys per Get buffer.
	})
	if err != nil {
		panic(err)
	}
	ristrettoStore := ristretto_store.NewRistretto(ristrettoCache)

	cacheManager = cache.New[any](ristrettoStore)

	return cacheManager
}

func WithCache(key string, cacheOption store.Option, compute func() interface{}) interface{} {
	var tmpCache = CacheManager()
	resp, err := tmpCache.Get(context.Background(), key)
	if err == nil && resp != nil {
		return resp
	}
	resp = compute()
	_ = tmpCache.Set(context.Background(), key, resp, cacheOption)
	return resp
}
