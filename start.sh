#!/bin/bash

if [ ! -f "./mixin-channel-bot" ];then
  go build -o mixin-channel-bot main.go
fi

# while cat auto is 1, while do
while [ `cat auto` -eq 1 ]
do
    ./mixin-channel-bot
    sleep 1
done
