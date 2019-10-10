package karmabot

import (
	"fmt"
	"github.com/nlopes/slack"
	"regexp"
	"strings"
)

// NewKarmaBot New bot
func NewKarmaBot(apiToken string) {

	api := slack.New(apiToken)
	rtm := api.NewRTM()
	go rtm.ManageConnection()

	for {
		select {
		case msg := <-rtm.IncomingEvents:
			switch ev := msg.Data.(type) {
			case *slack.MessageEvent:
				if ev.SubType == "message_changed" {
					fmt.Println("Message edited... ignoring")
					break
				}

				info := rtm.GetInfo()
				text := ev.Text
				text = strings.TrimSpace(text)
				text = strings.ToLower(text)
				r := regexp.MustCompile("(.*?)([+-]+)$")
				matched := r.MatchString(text)
				captureGroups := r.FindStringSubmatch(text)
				//ev.User contains the internal id for the user
				userInformation, err := rtm.GetUserInfo(ev.User)
				if err != nil {
					fmt.Printf("Error 1 %s\n", err)
				}
				//ev.Channel contains the internal id for the channel
				channelInformation, err := rtm.GetChannelInfo(ev.Channel)
				if err != nil {
					fmt.Printf("Error 2 %s\n", err)
				}
				fmt.Printf("User: %s\n", userInformation.Name)
				fmt.Printf("Channel: %s\n", channelInformation.Name)
				fmt.Printf("Message: %s\n", text)

				if ev.User != info.User.ID && matched {
					fmt.Printf("Karma Word: %s\n", captureGroups[1])
					fmt.Printf("Karma modifier: %s\n", captureGroups[2])
					rtm.SendMessage(rtm.NewOutgoingMessage("Karma here", ev.Channel))

				}
			case *slack.RTMError:
				fmt.Printf("Error: %s\n", ev.Error())

			case *slack.InvalidAuthEvent:
				fmt.Printf("Invalid credentials")
				panic("Invalid credentials")

			default:
				fmt.Println("Default")
				continue
			}
		}
	}
}
