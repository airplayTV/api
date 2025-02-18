package controller

import (
	"github.com/airplayTV/api/model"
	"github.com/airplayTV/api/util"
	"github.com/dgraph-io/ristretto"
	"github.com/eko/gocache/lib/v4/cache"
	redis_store "github.com/eko/gocache/store/redis/v4"
	ristretto_store "github.com/eko/gocache/store/ristretto/v4"
	"github.com/redis/go-redis/v9"
	"github.com/spf13/viper"
	"log"
)

var (
	globalCache  *cache.Cache[any]
	cmsApiConfig []model.CmsApiConfig
)

func init() {
	viper.AddConfigPath(".")    // optionally look for config in the working directory
	err := viper.ReadInConfig() // Find and read the config file
	if err != nil {             // Handle errors reading the config file
		log.Panicln("[ConfigError]", err.Error())
		return
	}
	if err = viper.UnmarshalKey("source", &cmsApiConfig); err != nil {
		log.Println("[ConfigUnmarshalError]", err.Error())
		return
	}

	initRistrettoCache()
	//initRedisCache()// 这玩意还需要给模型实现（implement encoding.BinaryMarshaler）！！！

	initHttpHeader()

}

func initRistrettoCache() {
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

func initRedisCache() {
	if globalCache == nil {
		redisStore := redis_store.NewRedis(redis.NewClient(&redis.Options{
			Addr: "127.0.0.1:6379",
		}))
		globalCache = cache.New[any](redisStore)
	}
}

func initHttpHeader() {
	for _, h := range sourceMap {
		header, err := util.LoadHttpHeader(h.Handler.Name())
		if err != nil {
			continue
		}
		if err = h.Handler.UpdateHeader(header); err != nil {
			log.Println("[设置http异常]", h.Handler.Name(), err.Error())
		}
	}
}
