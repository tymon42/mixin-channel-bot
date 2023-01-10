package util

import (
	"encoding/json"
	"log"
	"os"

	"github.com/fox-one/mixin-sdk-go"
	"github.com/glebarez/sqlite"
	pkg_db "github.com/leaper-one/pkg/db"
	user_core "github.com/tymon42/mixin-channel-bot/user/core"
	user_store "github.com/tymon42/mixin-channel-bot/user/store"
	"gorm.io/gorm"
)

func InitMixinAndDB(configFilePath string) (*mixin.Client, user_core.MixinUserStore, error) {

	// Open the keystore file
	f, err := os.Open(configFilePath)
	if err != nil {
		log.Panicln(err)
	}

	// Read the keystore file as json into mixin.Keystore, which is a go struct
	var store mixin.Keystore
	if err := json.NewDecoder(f).Decode(&store); err != nil {
		log.Panicln(err)
	}

	// Create a Mixin Client from the keystore, which is the instance to invoke Mixin APIs
	client, err := mixin.NewFromKeystore(&store)
	if err != nil {
		log.Panicln(err)
		return nil, nil, err
	}

	// Connect to db
	db, err := gorm.Open(sqlite.Open("mixin-bot.db"), &gorm.Config{})
	if err != nil {
		log.Printf("open db failed, err: %v", err)
		return nil, nil, err
	}
	err = db.AutoMigrate(&user_core.MixinUser{})
	if err != nil {
		log.Printf("auto migrate failed, err: %v", err)
		return nil, nil, err
	}
	dbc := user_store.NewMixinUserStore(&pkg_db.DB{
		Write: db,
		Read:  db,
	})

	return client, dbc, nil
}
