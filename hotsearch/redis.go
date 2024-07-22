package hotsearch

import (
	"context"
	"github.com/go-redis/redis/v8"
)

type RedisClient struct {
	client *redis.Client
	ctx    context.Context
}

func NewRedisClient(addr, password string, db int) *RedisClient {
	rdb := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})
	ctx := context.Background()
	return &RedisClient{client: rdb, ctx: ctx}
}

func (r *RedisClient) RecordKeyword(keyword string) error {
	// 增加关键词的计数
	return r.client.ZIncrBy(r.ctx, "search_keywords", 1, keyword).Err()
}

func (r *RedisClient) GetTopKeywords(limit int) ([]redis.Z, error) {
	// 获取前 N 个热门关键词
	return r.client.ZRevRangeWithScores(r.ctx, "search_keywords", 0, int64(limit-1)).Result()
}
