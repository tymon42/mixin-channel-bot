package util

import (
	"context"
	"fmt"
	"image"
	"io/ioutil"
	"os"
	"strings"

	"github.com/fox-one/mixin-sdk-go"
)

/*
getDirFirstPhotoPath gets first photo path in dir

	递归读取目录下的第一个 .jpg 文件
	传入相对路径则返回相对路径，传入绝对路径则返回绝对路
*/
func GetDirFirstPhoto(dirPath string) (photoPath string, photoName string, err error) {
	files, _ := ioutil.ReadDir(dirPath) // 读取目录下的所有文件（包括文件夹）
	fmt.Printf("files: %+v\n", files)
	for _, f := range files {
		if f.IsDir() { // 递归读取，直到第一个 .jpg
			return GetDirFirstPhoto(dirPath + "/" + f.Name())
		} else {
			if strings.HasSuffix(f.Name(), ".jpg") {
				photoName = f.Name()
				break
			}
		}
	}
	return dirPath + "/" + photoName, photoName, nil
}

// UploadPhoto upload photo to mixin S3
func UploadPhoto(ctx context.Context, client *mixin.Client, photoPath string) (*mixin.Attachment, int, int, error) {
	// Read the photo file
	f, err := ioutil.ReadFile(photoPath)
	if err != nil {
		return nil, 0, 0, err
	}

	// open the photo file
	photo, err := os.Open(photoPath)
	if err != nil {
		return nil, 0, 0, err
	}
	defer photo.Close()

	c, _, err := image.DecodeConfig(photo)
	if err != nil {
		return nil, 0, 0, err
	}

	attachment, err := client.CreateAttachment(ctx)
	if err != nil {
		return nil, 0, 0, err
	}
	err = mixin.UploadAttachment(ctx, attachment, f)
	if err != nil {
		return nil, 0, 0, err
	}

	return attachment, c.Width, c.Height, nil
}
