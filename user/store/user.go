package store

import (
	"context"
	"errors"

	"mixin-bot/user/core"

	"github.com/leaper-one/pkg/db"
	"gorm.io/gorm"
)

func NewMixinUserStore(db *db.DB) core.MixinUserStore {
	return &MixinUserStore{db: db}
}

type MixinUserStore struct {
	db *db.DB
}

func toUpdateParams(user *core.MixinUser) map[string]interface{} {
	return map[string]interface{}{
		"uuid":            user.UUID,
		"conversation_id": user.ConversationID,
	}
}

func update(db *db.DB, user *core.MixinUser) (int64, error) {
	updates := toUpdateParams(user)
	tx := db.Update().Model(user).Where("id = ? OR uuid = ?", user.ID, user.UUID).Updates(updates)
	return tx.RowsAffected, tx.Error
}

func (s *MixinUserStore) Save(_ context.Context, user *core.MixinUser) error {
	return s.db.Tx(func(tx *db.DB) error {
		var rows int64
		var err error
		rows, err = update(tx, user)
		if err != nil {
			return err
		}

		if rows == 0 {
			return tx.Update().Create(user).Error
		}

		return nil
	})
}

// Delete deletes user
func (s *MixinUserStore) Delete(_ context.Context, user *core.MixinUser) error {
	return s.db.Update().Where("uuid = ?", user.UUID).Delete(user).Error
}

// Find finds user by id
func (s *MixinUserStore) Find(_ context.Context, id uint) (*core.MixinUser, error) {
	user := core.MixinUser{}
	if err := s.db.View().Where("id = ?", id).Take(&user).Error; err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	} else if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &user, nil
}

// List returns a list of users and next offset by offset and limit
func (s *MixinUserStore) List(_ context.Context, offset int, limit int) ([]*core.MixinUser, int, error) {
	var users []*core.MixinUser
	var count int64
	if err := s.db.View().Offset(offset).Limit(limit).Find(&users).Count(&count).Error; err != nil {
		return nil, 0, err
	}
	return users, offset+int(count), nil
}
