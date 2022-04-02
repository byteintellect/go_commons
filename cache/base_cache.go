package cache

import (
	"context"
	"github.com/byteintellect/go_commons/entity"
	"time"
)

type BaseCache interface {
	Put(ctx context.Context, base entity.Base) error
	Get(ctx context.Context, externalId string) (entity.Base, error)
	MultiGet(ctx context.Context, externalIds []string) ([]entity.Base, error)
	Delete(ctx context.Context, externalId string) error
	MultiDelete(ctx context.Context, externalIds []string) error
	PutWithTtl(ctx context.Context, base entity.Base, duration time.Duration) error
	DeleteAll(ctx context.Context) error
	Health(ctx context.Context) error
}
