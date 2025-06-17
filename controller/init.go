package controller

import (
	"github.com/airplayTV/api/model"
	"github.com/airplayTV/api/util"
	"github.com/eko/gocache/lib/v4/cache"
	redis_store "github.com/eko/gocache/store/redis/v4"
	"github.com/redis/go-redis/v9"
	"github.com/spf13/viper"
	"log"
)

var (
	globalCache   = util.NewCacheManager()
	cmsApiConfig  []model.CmsApiConfig
	sourceModeMap map[string][]string
)

func init() {
	viper.AddConfigPath(".")    // optionally look for config in the working directory
	err := viper.ReadInConfig() // Find and read the config file
	if err != nil {             // Handle errors reading the config file
		log.Panicln("[ConfigError]", err.Error())
		return
	}
	if err = viper.UnmarshalKey("source", &cmsApiConfig); err != nil {
		log.Println("[ConfigUnmarshalError.source]", err.Error())
		return
	}
	if err = viper.UnmarshalKey("mode", &sourceModeMap); err != nil {
		log.Println("[ConfigUnmarshalError.mode]", err.Error())
		return
	}

	initHttpHeader()
}

// initRedisCache()// 这玩意还需要给模型实现（implement encoding.BinaryMarshaler）！！！
func initRedisCache() {
	if globalCache == nil {
		redisStore := redis_store.NewRedis(redis.NewClient(&redis.Options{
			Addr: "127.0.0.1:6379",
		}))
		globalCache = cache.New[any](redisStore)
	}
}

func initHttpHeader() {
	for _, h := range model.AppSourceMap() {
		header, err := util.LoadHttpHeader(h.Handler.Name())
		if err != nil {
			continue
		}
		if err = h.Handler.UpdateHeader(header); err != nil {
			log.Println("[设置http异常]", h.Handler.Name(), err.Error())
		}
	}
}
