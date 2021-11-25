package cache

import (
	"github.com/byteintellect/go_commons/entity"
	"time"
)

type BaseCache interface {
	Put(base entity.Base) error
	Get(externalId string) (entity.Base, error)
	MultiGet(externalIds []string) ([]entity.Base, error)
	Delete(externalId string) error
	MultiDelete(externalIds []string) error
	PutWithTtl(base entity.Base, duration time.Duration) error
	DeleteAll() error
	Health() error
}
