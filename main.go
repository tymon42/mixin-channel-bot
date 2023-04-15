package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	_ "image/jpeg"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/robfig/cron/v3"
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
	Dbc    user_core.MixinUserStore
	Client *mixin.Client
}

// pubPhotoMsgToAll 向所有用户发送图片
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

// pubTextMsgToAll 向所有用户发送文字
func (s Services) pubTextMsgToAll(ctx context.Context, text string) error {
	// 向所有用户发送文字
	var offset int = 0
	for { // 为最多一百个用户发送文字
		users, count, err := s.Dbc.List(ctx, offset, 100)
		if err != nil {
			return err
		}
		fmt.Printf("count: %v\n", count)

		// 生成发送消息的请求
		var msgs []*mixin.MessageRequest
		for _, user := range users {
			uuid, _ := uuid.NewV4()
			// Create a request
			reply := &mixin.MessageRequest{
				ConversationID: user.ConversationID,
				RecipientID:    user.UUID,
				MessageID:      uuid.String(),
				Category:       mixin.MessageCategoryPlainText,
				Data:           base64.StdEncoding.EncodeToString([]byte(text)),
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

// pubPhotoMsgToBotUser 向 bot 用户发送图片
func (s Services) pubPhotoMsgToBotUser(ctx context.Context, attachmentID string, width, height int) error {
	// 向 Bot 用户发送图片
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
	uuid, _ := uuid.NewV4()
	msg := &mixin.MessageRequest{
		ConversationID: mixin.UniqueConversationID("c66ed586-33fc-44b5-b3f0-071083ffd049", "64abce35-ad54-4828-9e87-b2f46148b0ad"),
		RecipientID:    "64abce35-ad54-4828-9e87-b2f46148b0ad", //
		MessageID:      uuid.String(),
		Category:       mixin.MessageCategoryPlainImage,
		Data:           encoded,
	}
	err := s.Client.SendMessage(ctx, msg)
	if err != nil {
		return err
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

	c := cron.New() // 创建一个定时任务

	// 添加定时任务
	c.AddFunc("2,3,4 18 * * *", func() {
		// 重复三次
		for i := 0; i < 1; i++ {
			root := "./files/夜半丽影"
			// 获取第一张图片
			photoPath, _, err := util.GetDirFirstPhoto(root)
			if err != nil {
				fmt.Printf("err: %v\n", err)
			}

			// 上传图片，获取图片比例信息
			attachmentInfo, width, height, err := util.UploadPhoto(ctx, client, photoPath)
			if err != nil {
				fmt.Printf("err: %v\n", err)
			}

			err = services.pubPhotoMsgToBotUser(ctx, attachmentInfo.AttachmentID, width, height)
			if err != nil {
				fmt.Printf("err: %v\n", err)
			}

			// move file to sent folder
			err = os.Remove(photoPath)
			if err != nil {
				fmt.Printf("err: %v\n", err)
			}

			time.Sleep(1 * time.Second)
		}
	})

	// 启动定时任务
	c.Start()

	// 启动服务, 保证服务一直运行
	// should use a simple channel send/receive instead of select with a single case
	select {
	case <-ctx.Done():
		fmt.Println("Done")

		// 关闭定时任务
		c.Stop()

		// return 0
		os.Exit(0)
	}

}
