package core

import (
	"context"

	"gorm.io/gorm"
)

type (
	MixinUser struct {
		gorm.Model
		UUID           string `gorm:"size:36;unique_index;index" json:"uuid,omitempty"`
		ConversationID string `gorm:"size:36;unique_index" json:"conversation_id,omitempty"`
	}

	MixinUserStore interface {
		// Save
		Save(ctx context.Context, user *MixinUser) error
		// Delete
		Delete(ctx context.Context, user *MixinUser) error
		// Find
		Find(ctx context.Context, id uint) (*MixinUser, error)
		// List returns a list of users and next offset by offset and limit
		List(ctx context.Context, offset int, limit int) ([]*MixinUser, int, error)
	}
)
