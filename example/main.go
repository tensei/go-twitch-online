package main

import (
	"time"

	"github.com/sirupsen/logrus"
	twitch "github.com/tensei/go-twitch-online"
)

func main() {
	c, _ := twitch.New(&twitch.Params{
		ClientID: "xxxxxxxxxxxxxxx",
		OAuth:    "xxxxxxxxxxxxxxx",
	})

	c.AddStreamer("71092938")
	c.OnOnlineCallback(func(stream twitch.Stream) {
		logrus.Infof("%s is online", stream.UserName)
	})
	c.OnOfflineCallback(func(channelID string) {
		logrus.Infof("%s is offline", channelID)
	})
	c.SetInterval(time.Second * 10)
	c.Start()
}
