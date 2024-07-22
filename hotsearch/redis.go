package hotsearch

import (
	"context"
	"fmt"
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
	return &RedisClient{
		client: rdb,
		ctx:    ctx,
	}
}

//func (r *RedisClient) RecordKeyword(keyword string) error {
//	key := "search_keywords"
//	// 增加关键词的计数
//	if err := r.client.ZIncrBy(r.ctx, key, 1, keyword).Err(); err != nil {
//		return err
//	}
//	// 更新 key 的过期时间为一周
//	return r.client.Expire(r.ctx, key, 7*24*time.Hour).Err()
//}
//
//func (r *RedisClient) GetTopKeywords(limit int) ([]redis.Z, error) {
//	// 获取前 N 个热门关键词
//	return r.client.ZRevRangeWithScores(r.ctx, "search_keywords", 0, int64(limit-1)).Result()
//}

func (r *RedisClient) IncrementKeywordCount(keyword string) error {
	key := fmt.Sprintf("keyword:%s", keyword) // 为每个关键词创建独立的键
	sortedSetKey := "keywords:rank"           // Sorted Set 的键

	// 使用事务同时增加关键词计数和更新 Sorted Set
	_, err := r.client.TxPipelined(r.ctx, func(pipe redis.Pipeliner) error {
		// 增加关键词的计数
		pipe.Incr(r.ctx, key)
		// 更新 Sorted Set 中的计数
		pipe.ZIncrBy(r.ctx, sortedSetKey, 1, keyword)
		// 设置关键词的过期时间
		pipe.Expire(r.ctx, key, 7*24*time.Hour)
		return nil
	})
	if err != nil {
		return err
	}
	return nil
}

func (r *RedisClient) GetTopKeywords(limit int) ([]string, error) {
	sortedSetKey := "keywords:rank"
	// 获取得分最高的 limit 个关键词
	return r.client.ZRevRange(r.ctx, sortedSetKey, 0, int64(limit-1)).Result()
}

func (r *RedisClient) CleanUpSortedSet() error {
	sortedSetKey := "keywords:rank"

	// 获取 Sorted Set 中的所有关键词
	keywords, err := r.client.ZRange(r.ctx, sortedSetKey, 0, -1).Result()
	if err != nil {
		return err
	}

	for _, keyword := range keywords {
		key := fmt.Sprintf("keyword:%s", keyword)
		// 检查关键词的独立键是否存在
		var exists int64
		exists, err = r.client.Exists(r.ctx, key).Result()
		if err != nil {
			return err
		}
		// 如果独立键不存在，从 Sorted Set 中删除关键词
		if exists == 0 {
			_, err = r.client.ZRem(r.ctx, sortedSetKey, keyword).Result()
			if err != nil {
				return err
			}
		}
	}
	return nil
}
