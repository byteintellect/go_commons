package db

import (
	"github.com/byteintellect/go_commons"
	"context"
	_ "github.com/elastic/go-elasticsearch/v7/esapi"
)

type BaseNoSQLRepo interface {
	BaseRepository
	ExactSearch(ctx context.Context, key string, value interface{}) (error, []go_commons.Base)
	RangeSearch(ctx context.Context, key string, start, end interface{}) (error, []go_commons.Base)
	TextSearch(ctx context.Context, value string) (error, []go_commons.Base)
	IndexMappings(ctx context.Context) error
}
