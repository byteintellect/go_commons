package cache

import (
	"context"
	"github.com/byteintellect/go_commons/entity"
	"github.com/go-redis/redis/extra/redisotel/v8"
	"github.com/go-redis/redis/v8"
	"github.com/sirupsen/logrus"
	traceSdk "go.opentelemetry.io/otel/sdk/trace"
	"time"
)

type RedisCache struct {
	*redis.Client
	logger        *logrus.Logger
	entityCreator entity.EntityCreator
}

func (r *RedisCache) Put(ctx context.Context, base entity.Base) error {
	cmd := r.Client.Set(ctx, base.GetExternalId(), base, 0)
	return cmd.Err()
}

func (r *RedisCache) Get(ctx context.Context, externalId string) (entity.Base, error) {
	cmd := r.Client.Get(ctx, externalId)
	if cmd.Err() != nil {
		return nil, cmd.Err()
	}
	entity := r.entityCreator()
	err := cmd.Scan(&entity)
	if err != nil {
		return nil, err
	}
	return entity, nil
}

func (r *RedisCache) MultiGet(ctx context.Context, externalIds []string) ([]entity.Base, error) {
	var result []entity.Base
	for _, externalId := range externalIds {
		base, err := r.Get(ctx, externalId)
		if err != nil {
			return nil, err
		}
		result = append(result, base)
	}
	return result, nil
}

func (r *RedisCache) Delete(ctx context.Context, externalId string) error {
	statusCmd := r.Client.Del(ctx, externalId)
	if statusCmd.Err() != nil {
		return statusCmd.Err()
	}
	return nil
}

func (r *RedisCache) MultiDelete(ctx context.Context, externalIds []string) error {
	statusCmd := r.Client.Del(ctx, externalIds...)
	if statusCmd.Err() != nil {
		return statusCmd.Err()
	}
	return nil
}

func (r *RedisCache) PutWithTtl(ctx context.Context, base entity.Base, duration time.Duration) error {
	statusCmd := r.Client.Set(ctx, base.GetExternalId(), base, duration)
	if statusCmd.Err() != nil {
		return statusCmd.Err()
	}
	return nil
}

func (r *RedisCache) DeleteAll(ctx context.Context) error {
	cmd := r.Client.FlushDB(ctx)
	if cmd.Err() != nil {
		return cmd.Err()
	}
	return nil
}

func (r *RedisCache) Health(ctx context.Context) error {
	pong, err := r.Client.Ping(ctx).Result()
	if err != nil {
		return err
	}
	r.logger.Infof("Health check ping response <%v>", pong)
	return nil
}

func NewRedisCache(
	addr string,
	password string,
	db uint,
	logger *logrus.Logger,
	entityCreator entity.EntityCreator,
	provider *traceSdk.TracerProvider) BaseCache {
	client := redis.NewClient(
		&redis.Options{
			Addr:     addr,
			Password: password,
			DB:       int(db),
		})
	client.AddHook(redisotel.NewTracingHook(redisotel.WithTracerProvider(provider)))
	return &RedisCache{
		client,
		logger,
		entityCreator,
	}
}
