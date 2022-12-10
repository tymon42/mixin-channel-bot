package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"image"
	_ "image/jpeg"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	pkg_db "github.com/leaper-one/pkg/db"

	user_core "github.com/tymon42/mixin-channel-bot/user/core"
	user_store "github.com/tymon42/mixin-channel-bot/user/store"

	"github.com/fox-one/mixin-sdk-go"
	"github.com/glebarez/sqlite"
	"github.com/gofrs/uuid"
	"gorm.io/gorm"
)

var (
	// Specify the keystore file in the -config parameter
	config = flag.String("config", "keystore.json", "keystore file path")
)

// saveMixinUser save mixin user to db
func saveMixinUser(ctx context.Context, dbc user_core.MixinUserStore, mixinUser *user_core.MixinUser) error {
	err := dbc.Save(ctx, mixinUser)
	if err != nil {
		return err
	}

	return nil
}

// deleteMixinUser delete mixin user from db
func deleteMixinUser(ctx context.Context, dbc user_core.MixinUserStore, mixinUser *user_core.MixinUser) error {
	err := dbc.Delete(ctx, mixinUser)
	if err != nil {
		return err
	}
	return nil
}

// moveFile move file to another directory
func moveFile(src, dst string) error {
	err := os.Rename(src, dst)
	if err != nil {
		return err
	}
	return nil
}

// uploadPhoto send photo to mixin
func uploadPhoto(ctx context.Context, client *mixin.Client, photoPath string) (string, int, int, error) {
	// Read the photo file
	f, err := ioutil.ReadFile(photoPath)
	if err != nil {
		return "", 0, 0, err
	}

	// open the photo file
	photo, err := os.Open(photoPath)
	if err != nil {
		return "", 0, 0, err
	}
	defer photo.Close()

	c, _, err := image.DecodeConfig(photo)
	if err != nil {
		return "", 0, 0, err
	}

	attachment, err := client.CreateAttachment(ctx)
	if err != nil {
		return "", 0, 0, err
	}
	err = mixin.UploadAttachment(ctx, attachment, f)
	if err != nil {
		return "", 0, 0, err
	}

	return attachment.AttachmentID, c.Width, c.Height, nil
}

func main() {
	// Use flag package to parse the parameters
	flag.Parse()

	// Open the keystore file
	f, err := os.Open(*config)
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
	}

	// Connect to db
	db, err := gorm.Open(sqlite.Open("mixin-bot.db"), &gorm.Config{})
	if err != nil {
		log.Printf("open db failed, err: %v", err)
	}
	err = db.AutoMigrate(&user_core.MixinUser{})
	if err != nil {
		log.Printf("auto migrate failed, err: %v", err)
	}
	dbc := user_store.NewMixinUserStore(&pkg_db.DB{
		Write: db,
		Read:  db,
	})

	// Prepare the message loop that handle every incoming messages,
	// and reply it with the same content.
	// We use a callback function to handle them.
	h := func(ctx context.Context, msg *mixin.MessageView, userID string) error {
		// if there is no valid user id in the message, drop it
		if userID, _ := uuid.FromString(msg.UserID); userID == uuid.Nil {
			return nil
		}

		// The incoming message's message ID, which is an UUID.
		id, _ := uuid.FromString(msg.MessageID)

		if userID == "53c81550-f7e1-4103-9501-b3147030f57a" {

			var offset int = 0
			for {
				users, count, err := dbc.List(ctx, offset, 100)
				if err != nil {
					return err
				}
				fmt.Printf("count: %v\n", count)

				var msgs []*mixin.MessageRequest
				for _, user := range users {
					uuid, _ := uuid.NewV4()
					// Create a request
					reply := &mixin.MessageRequest{
						ConversationID: user.ConversationID,
						RecipientID:    user.UUID,
						MessageID:      uuid.String(),
						Category:       msg.Category,
						Data:           msg.Data,
					}
					msgs = append(msgs, reply)
				}
				client.SendMessages(ctx, msgs)

				if count < 100 {
					break
				}
			}
		}

		// The incoming message's data is a Base64 encoded data, decode it.
		msgContentByte, err := base64.StdEncoding.DecodeString(msg.Data)
		if err != nil {
			return err
		}
		switch string(msgContentByte) {
		case "你好", "hello", "hi", "Hi", "/help", "/h":
			// Create a request
			reply := &mixin.MessageRequest{
				ConversationID: msg.ConversationID,
				RecipientID:    msg.UserID,
				MessageID:      uuid.NewV5(id, "reply").String(),
				Category:       mixin.MessageCategoryPlainText,
				Data:           base64.StdEncoding.EncodeToString([]byte("你好，欢迎关注本频道，\n\n/help /h 查看帮助\n\n/subscribe /s 订阅\n\n/unsubscribe /u 取消订阅")),
			}
			return client.SendMessage(ctx, reply)
		case "/subscribe", "/s":
			saveMixinUser(ctx, dbc, &user_core.MixinUser{
				UUID:           msg.UserID,
				ConversationID: msg.ConversationID,
			})
			// Create a request
			reply := &mixin.MessageRequest{
				ConversationID: msg.ConversationID,
				RecipientID:    msg.UserID,
				MessageID:      uuid.NewV5(id, "reply").String(),
				Category:       mixin.MessageCategoryPlainText,
				Data:           base64.StdEncoding.EncodeToString([]byte("订阅成功")),
			}
			return client.SendMessage(ctx, reply)
		case "/unsubscribe", "/u":
			deleteMixinUser(ctx, dbc, &user_core.MixinUser{
				UUID: msg.UserID,
			})
			reply := &mixin.MessageRequest{
				ConversationID: msg.ConversationID,
				RecipientID:    msg.UserID,
				MessageID:      uuid.NewV5(id, "reply").String(),
				Category:       mixin.MessageCategoryPlainText,
				Data:           base64.StdEncoding.EncodeToString([]byte("取消订阅成功")),
			}
			return client.SendMessage(ctx, reply)
		default:
			return nil
		}
	}

	ctx, stop := signal.NotifyContext(
		context.Background(),
		syscall.SIGINT,
		syscall.SIGTERM,
	)
	defer stop()

	go func() {
		// var files []string
		root := "./files"
		err = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
			// files = append(files, path)
			if !strings.Contains(path, ".jpg") {
				return nil
			}
			fmt.Println(path)
			attachmentID, width, height, err := uploadPhoto(ctx, client, path)
			if err != nil {
				return err
			}

			var offset int = 0
			for {
				users, count, err := dbc.List(ctx, offset, 100)
				if err != nil {
					return err
				}
				fmt.Printf("count: %v\n", count)

				data := &mixin.ImageMessage{
					AttachmentID: attachmentID,
					MimeType:     "image/jpeg",
					Width:        width,
					Height:       height,
					Size:         4096,
					Thumbnail:    "base64 encoded",
				}

				// turn data into json
				dataByte, _ := json.Marshal(data)
				// turn dataByte to base64
				encoded := base64.StdEncoding.EncodeToString(dataByte)

				var msgs []*mixin.MessageRequest
				for _, user := range users {
					uuid, _ := uuid.NewV4()
					// Create a request
					reply := &mixin.MessageRequest{
						ConversationID: user.ConversationID,
						RecipientID:    user.UUID,
						MessageID:      uuid.String(),
						Category:       mixin.MessageCategoryPlainImage,
						Data:           encoded,
					}
					msgs = append(msgs, reply)
				}
				client.SendMessages(ctx, msgs)

				if count < 100 {
					break
				}
			}

			// move file to sent folder
			err = os.Rename(path, "./sent/"+path)
			if err != nil {
				return err
			}

			time.Sleep(30 * time.Minute)
			// time.Sleep(1 * time.Second)

			return nil
		})
		if err != nil {
			log.Panicln(err)
		}
	}()

	// Start the message loop.
	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(time.Second):
			// Pass the callback function into the `BlazeListenFunc`
			if err := client.LoopBlaze(ctx, mixin.BlazeListenFunc(h)); err != nil {
				log.Printf("LoopBlaze: %v", err)
			}
		}
	}
}
