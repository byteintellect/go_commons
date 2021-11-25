package db

import (
	"context"
	"github.com/byteintellect/go_commons/entity"
	_ "github.com/elastic/go-elasticsearch/v7/esapi"
)

type BaseNoSQLRepo interface {
	BaseRepository
	ExactSearch(ctx context.Context, key string, value interface{}) (error, []entity.Base)
	RangeSearch(ctx context.Context, key string, start, end interface{}) (error, []entity.Base)
	TextSearch(ctx context.Context, value string) (error, []entity.Base)
	IndexMappings(ctx context.Context) error
}
