package controller

import (
	"github.com/dgraph-io/ristretto"
	"github.com/eko/gocache/lib/v4/cache"
	ristretto_store "github.com/eko/gocache/store/ristretto/v4"
)

var (
	globalCache *cache.Cache[any]
)

func init() {
	if globalCache == nil {
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
		globalCache = cache.New[any](ristrettoStore)
	}
}
