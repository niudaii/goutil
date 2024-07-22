package hotsearch

import (
	"context"
	"github.com/go-redis/redis/v8"
	"time"
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
	key := "search_keywords"
	// 增加关键词的计数
	if err := r.client.ZIncrBy(r.ctx, key, 1, keyword).Err(); err != nil {
		return err
	}
	// 更新 key 的过期时间为一周
	return r.client.Expire(r.ctx, key, 7*24*time.Hour).Err()
}

func (r *RedisClient) GetTopKeywords(limit int) ([]redis.Z, error) {
	// 获取前 N 个热门关键词
	return r.client.ZRevRangeWithScores(r.ctx, "search_keywords", 0, int64(limit-1)).Result()
}
