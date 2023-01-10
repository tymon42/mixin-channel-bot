package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	_ "image/jpeg"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/robfig/cron/v3"
	"github.com/tymon42/mixin-channel-bot/user/core"
	"github.com/tymon42/mixin-channel-bot/util"

	"github.com/fox-one/mixin-sdk-go"
	"github.com/gofrs/uuid"
	user_core "github.com/tymon42/mixin-channel-bot/user/core"
)

var (
	// Specify the keystore file in the -config parameter
	config = flag.String("config", "keystore.json", "keystore file path")
)

type Services struct {
	Dbc    core.MixinUserStore
	Client *mixin.Client
}

func (s Services) pubPhotoMsgToAll(ctx context.Context, attachmentID string, width, height int) error {
	// 向所有用户发送图片
	var offset int = 0
	for { // 为最多一百个用户发送图片
		users, count, err := s.Dbc.List(ctx, offset, 100)
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

		// 生成发送消息的请求
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
		s.Client.SendMessages(ctx, msgs)

		// TODO: 有点不太确定要不要加这个一百
		offset += 100

		// count 小于 100 说明已经处理完最后一批用户了
		if count < 100 {
			break
		}
	}
	return nil
}

func main() {
	// Use flag package to parse the parameters
	flag.Parse()

	// 初始化 mixin client 和数据库
	client, dbc, err := util.InitMixinAndDB(*config)
	if err != nil {
		fmt.Printf("err: %v\n", err)
	}

	services := &Services{
		Client: client,
		Dbc:    dbc,
	}

	ctx, stop := signal.NotifyContext(
		context.Background(),
		syscall.SIGINT,
		syscall.SIGTERM,
	)
	defer stop()

	h := func(ctx context.Context, msg *mixin.MessageView, userID string) error {
		// if there is no valid user id in the message, drop it
		if userID, _ := uuid.FromString(msg.UserID); userID == uuid.Nil {
			return nil
		}

		// The incoming message's message ID, which is an UUID.
		msgId, _ := uuid.FromString(msg.MessageID)

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
				MessageID:      uuid.NewV5(msgId, "reply").String(),
				Category:       mixin.MessageCategoryPlainText,
				Data:           base64.StdEncoding.EncodeToString([]byte("你好，欢迎关注本频道，\n\n/help /h 查看帮助\n\n/subscribe /s 订阅\n\n/unsubscribe /u 取消订阅")),
			}
			return client.SendMessage(ctx, reply)
		case "/subscribe", "/s":
			dbc.Save(ctx, &user_core.MixinUser{
				UUID:           msg.UserID,
				ConversationID: msg.ConversationID,
			})
			// 回复订阅成功
			reply := &mixin.MessageRequest{
				ConversationID: msg.ConversationID,
				RecipientID:    msg.UserID,
				MessageID:      uuid.NewV5(msgId, "reply").String(),
				Category:       mixin.MessageCategoryPlainText,
				Data:           base64.StdEncoding.EncodeToString([]byte("订阅成功")),
			}
			return client.SendMessage(ctx, reply)
		case "/unsubscribe", "/u":
			dbc.Delete(ctx, &user_core.MixinUser{
				UUID: msg.UserID,
			})
			// 回复取消订阅成功
			reply := &mixin.MessageRequest{
				ConversationID: msg.ConversationID,
				RecipientID:    msg.UserID,
				MessageID:      uuid.NewV5(msgId, "reply").String(),
				Category:       mixin.MessageCategoryPlainText,
				Data:           base64.StdEncoding.EncodeToString([]byte("取消订阅成功")),
			}
			return client.SendMessage(ctx, reply)
		default:
			return nil
		}
	}

	c := cron.New()
	c.AddFunc("30 9 * * *", func() {
		// 重复三次
		for i := 0; i < 3; i++ {
			root := "./files"
			// 获取第一张图片
			photoPath, photoName, err := util.GetDirFirstPhoto(root)
			if err != nil {
				fmt.Printf("err: %v\n", err)
			}

			// 上传图片，获取图片比例信息
			attachmentInfo, width, height, err := util.UploadPhoto(ctx, client, photoPath)
			if err != nil {
				fmt.Printf("err: %v\n", err)
			}

			err = services.pubPhotoMsgToAll(ctx, attachmentInfo.AttachmentID, width, height)
			if err != nil {
				fmt.Printf("err: %v\n", err)
			}

			// move file to sent folder
			os.Rename(photoPath, "./sent/"+photoName)
		}
	})

	c.Start()

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
