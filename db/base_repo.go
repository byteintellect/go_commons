package db

import (
	"context"
	"github.com/byteintellect/go_commons/entity"
)

type BaseRepository interface {
	GetById(ctx context.Context, id uint64) (error, entity.Base)
	GetByExternalId(ctx context.Context, externalId string) (error, entity.Base)
	MultiGetByExternalId(ctx context.Context, externalIds []string) (error, []entity.Base)
	Create(ctx context.Context, base entity.Base) (error, entity.Base)
	Update(ctx context.Context, externalId string, updatedBase entity.Base) (error, entity.Base)
	Search(ctx context.Context, params map[string]string) (error, []entity.Base)
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
