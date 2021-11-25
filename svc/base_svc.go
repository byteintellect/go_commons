package svc

import (
	"context"
	"github.com/byteintellect/go_commons/db"
	"github.com/byteintellect/go_commons/entity"
)

type BaseSvc struct {
	Persistence db.BaseRepository
}

func (b *BaseSvc) Init(repo db.BaseRepository) {
	b.Persistence = repo
}

func (b *BaseSvc) FindById(ctx context.Context, id uint64) (error, entity.Base) {
	return b.Persistence.GetById(ctx, id)
}

func (b *BaseSvc) FindByExternalId(ctx context.Context, id string) (error, entity.Base) {
	return b.Persistence.GetByExternalId(ctx, id)
}

func (b *BaseSvc) MultiGetByExternalId(ctx context.Context, ids []string) (error, []entity.Base) {
	return b.Persistence.MultiGetByExternalId(ctx, ids)
}

func (b *BaseSvc) Create(ctx context.Context, base entity.Base) (error, entity.Base) {
	return b.Persistence.Create(ctx, base)
}

func (b *BaseSvc) Update(ctx context.Context, id string, base entity.Base) (error, entity.Base) {
	return b.Persistence.Update(ctx, id, base)
}

func (b *BaseSvc) GetPersistence() db.BaseRepository {
	return b.Persistence
}

func NewBaseSvc(persistence db.BaseRepository) BaseSvc {
	return BaseSvc{
		Persistence: persistence,
	}
}
