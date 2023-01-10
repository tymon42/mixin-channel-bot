package util

import (
	"fmt"
	"testing"

	"github.com/fox-one/mixin-sdk-go"
	user_core "github.com/tymon42/mixin-channel-bot/user/core"
)

func TestInitMixinAndDB(t *testing.T) {
	type args struct {
		configFilePath string
	}
	tests := []struct {
		name    string
		args    args
		want    *mixin.Client
		want1   user_core.MixinUserStore
		wantErr bool
	}{
		{
			name: "test1",
			args: args{
				configFilePath: "../keystore.json",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1, err := InitMixinAndDB(tt.args.configFilePath)
			if (err != nil) != tt.wantErr {
				t.Errorf("InitMixinAndDB() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			fmt.Printf("got: %+v\n", got)
			fmt.Printf("got1: %+v\n", got1)
		})
	}
}
