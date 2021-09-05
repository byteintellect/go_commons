package cache

import (
	"github.com/byteintellect/go_commons"
	"time"
)

type BaseCache interface {
	Put(base go_commons.Base) error
	Get(externalId string) (go_commons.Base, error)
	MultiGet(externalIds []string) ([]go_commons.Base, error)
	Delete(externalId string) error
	MultiDelete(externalIds []string) error
	PutWithTtl(base go_commons.Base, duration time.Duration) error
	DeleteAll() error
	Health() error
}
