package db

import (
	"github.com/byteintellect/go_commons"
	"context"
	"database/sql"
	"errors"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type GORMRepositoryOption func(repository *GORMRepository)

type GORMRepository struct {
	db      *gorm.DB
	creator go_commons.EntityCreator
	logger  *logrus.Logger
}

func WithCreator(creator go_commons.EntityCreator) GORMRepositoryOption {
	return func(r *GORMRepository) {
		r.creator = creator
	}
}

func WithLogger(logger *logrus.Logger) GORMRepositoryOption {
	return func(r *GORMRepository) {
		r.logger = logger
	}
}

func WithDb(db *gorm.DB) GORMRepositoryOption {
	return func(r *GORMRepository) {
		r.db = db
	}
}

func (r *GORMRepository) GetDb() interface{} {
	// users will have to cast this to *gorm.Db
	return r.db
}

func NewGORMRepository(opts ...GORMRepositoryOption) *GORMRepository {
	repo := GORMRepository{}
	for _, opt := range opts {
		opt(&repo)
	}
	return &repo
}

func (r *GORMRepository) GetById(ctx context.Context, id uint64) (error, go_commons.Base) {
	entity := r.creator()
	if err := r.db.Table(string(entity.GetTable())).WithContext(ctx).Where("id = ?", id).First(&entity).Error; err != nil {
		return err, nil
	}
	return nil, entity
}

func (r *GORMRepository) GetByExternalId(ctx context.Context, externalId string) (error, go_commons.Base) {
	entity := r.creator()
	if err := r.db.WithContext(ctx).Table(string(entity.GetTable())).Where("external_id = ?", externalId).First(entity).Error; err != nil {
		return err, nil
	}
	return nil, entity
}

func (r *GORMRepository) populateRows(rows *sql.Rows) (error, []go_commons.Base) {
	var models []go_commons.Base
	for rows.Next() {
		entity := r.creator()
		entity, err := entity.FromSqlRow(rows)
		if err != nil {
			return err, nil
		}
		models = append(models, entity)
	}
	return nil, models
}

func (r *GORMRepository) MultiGetByExternalId(ctx context.Context, externalIds []string) (error, []go_commons.Base) {
	entity := r.creator()
	rows, err := r.db.WithContext(ctx).Table(string(entity.GetTable())).Where("external_id IN (?)", externalIds).Rows()
	if err != nil {
		return err, nil
	}
	return r.populateRows(rows)
}

func (r *GORMRepository) Create(ctx context.Context, base go_commons.Base) (error, go_commons.Base) {
	if err := r.db.Table(string(base.GetTable())).Model(base).WithContext(ctx).Create(base).Error; err != nil {
		return err, nil
	}
	return nil, base
}

func (r *GORMRepository) Update(ctx context.Context, externalId string, updatedBase go_commons.Base) (error, go_commons.Base) {
	err, entity := r.GetByExternalId(ctx, externalId)
	if err != nil {
		return err, nil
	}
	entity.Merge(updatedBase)
	if err := r.db.WithContext(ctx).Table(string(entity.GetTable())).Model(entity).Updates(entity).Error; err != nil {
		return err, nil
	}
	return nil, entity
}

func (r *GORMRepository) Search(ctx context.Context, params map[string]string) (error, []go_commons.Base) {
	return errors.New("not implemented"), nil
}
