package db

import (
	"github.com/byteintellect/go_commons"
	"context"
)

type BaseRepository interface {
	GetById(ctx context.Context, id uint64) (error, go_commons.Base)
	GetByExternalId(ctx context.Context, externalId string) (error, go_commons.Base)
	MultiGetByExternalId(ctx context.Context, externalIds []string) (error, []go_commons.Base)
	Create(ctx context.Context, base go_commons.Base) (error, go_commons.Base)
	Update(ctx context.Context, externalId string, updatedBase go_commons.Base) (error, go_commons.Base)
	Search(ctx context.Context, params map[string]string) (error, []go_commons.Base)
	GetDb() interface{}
}

type BaseDao struct {
	BaseRepository
}

func NewBaseGORMDao(opts ...GORMRepositoryOption) BaseDao {
	return BaseDao{
		NewGORMRepository(opts...),
	}
}
