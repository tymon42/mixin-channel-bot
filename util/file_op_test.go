package util

import (
	"fmt"
	"testing"
)

func Test_getDirFirstPhoto(t *testing.T) {
	type args struct {
		dirPath string
	}
	tests := []struct {
		name          string
		args          args
		wantPhotoPath string
		wantPhotoName string
		wantErr       bool
	}{
		{
			name: "test1",
			args: args{
				dirPath: "../files",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotPhotoPath, gotPhotoName, err := GetDirFirstPhoto(tt.args.dirPath)
			if (err != nil) != tt.wantErr {
				t.Errorf("getDirFirstPhoto() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			fmt.Printf("gotPhotoPath: %v\n", gotPhotoPath)
			fmt.Printf("gotPhotoName: %v\n", gotPhotoName)

		})
	}
}
