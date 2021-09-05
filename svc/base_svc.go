package svc

import (
	"context"
	"github.com/byteintellect/go_commons"
	"github.com/byteintellect/go_commons/db"
)

type BaseSvc struct {
	Persistence db.BaseRepository
}

func (b *BaseSvc) Init(repo db.BaseRepository) {
	b.Persistence = repo
}

func (b *BaseSvc) FindById(ctx context.Context, id uint64) (error, go_commons.Base) {
	return b.Persistence.GetById(ctx, id)
}

func (b *BaseSvc) FindByExternalId(ctx context.Context, id string) (error, go_commons.Base) {
	return b.Persistence.GetByExternalId(ctx, id)
}

func (b *BaseSvc) MultiGetByExternalId(ctx context.Context, ids []string) (error, []go_commons.Base) {
	return b.Persistence.MultiGetByExternalId(ctx, ids)
}

func (b *BaseSvc) Create(ctx context.Context, base go_commons.Base) (error, go_commons.Base) {
	return b.Persistence.Create(ctx, base)
}

func (b *BaseSvc) Update(ctx context.Context, id string, base go_commons.Base) (error, go_commons.Base) {
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
