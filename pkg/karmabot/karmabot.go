package karmabot

import (
	"log"
	"regexp"
	"strings"

	"github.com/mvazquezc/karma-bot/pkg/commands"
	"github.com/mvazquezc/karma-bot/pkg/database"
	"github.com/mvazquezc/karma-bot/pkg/utils"
	"github.com/slack-go/slack"
)

// NewKarmaBot New bot
func NewKarmaBot(apiToken string, dbFile string) {

	api := slack.New(apiToken)
	rtm := api.NewRTM()
	db := database.New(dbFile)
	db.Connect()
	commands := commands.New(&db)

	go rtm.ManageConnection()

	for msg := range rtm.IncomingEvents {
		switch ev := msg.Data.(type) {
		case *slack.MessageEvent:
			if ev.SubType == "message_changed" {
				log.Print("Message edited... ignoring")
				continue
			}
			info := rtm.GetInfo()

			var channelInformation *slack.Channel
			var membersInformation []string

			// Get conversation information
			channelInformation, err := rtm.GetConversationInfo(ev.Channel, true)

			if err != nil {
				log.Print("Ignoring message since we cannot get channel information")
				continue
			}
			memberParameters := slack.GetUsersInConversationParameters{
				ChannelID: ev.Channel,
			}

			membersInformation, _, err = api.GetUsersInConversation(&memberParameters)
			if err != nil {
				log.Print("Ignoring message since we cannot get members information")
				continue
			}

			var channelName string
			var members []string

			channelName = channelInformation.NameNormalized
			members = membersInformation

			//log.Printf("Channel name: %s, members: %s", channelName, members)
			text := ev.Text
			text = strings.TrimSpace(text)
			text = strings.ToLower(text)

			// Commands are implemented using a keyword rather than using slash commands to avoid
			// having to publish the bot in order to receive webhooks
			r := regexp.MustCompile("^(kb) (set|get|del|rank) (karma|globalkarma|admin|setting|alias|help)(.*)$")
			matched := r.MatchString(text)
			if matched {
				captureGroups := r.FindStringSubmatch(text)
				operation := captureGroups[2]
				operationGroup := captureGroups[3]
				operationArgs := captureGroups[4]
				who := strings.ToLower(ev.User)
				if operation == "get" && operationGroup == "help" {
					log.Printf("Printing help on channel %s", channelName)
					utils.PrintCommandsUsage(rtm, ev)
				} else {
					// add user that fires the command to the args
					commandOutput := commands.ProcessCommand(channelName, who, operation, operationGroup, operationArgs)
					rtm.SendMessage(rtm.NewOutgoingMessage(commandOutput, ev.Channel))
				}
			}
			splitText := strings.Fields(text)
			splitText = utils.FixEmptyKarma(splitText)
			for _, word := range splitText {
				trimmedWord := strings.TrimSpace(word)
				// Get rid of ``` at the start of the word, usually added by code blocks on slack
				trimmedWord = strings.TrimLeft(trimmedWord, "```")
				// Get rid of +++ at the start of the word, usually added by code patch outputs
				trimmedWord = strings.TrimLeft(trimmedWord, "+++")
				// Get rid of --- at the start of the word, usually added by code patch outputs
				trimmedWord = strings.TrimLeft(trimmedWord, "---")
				// If the message is code, we will ignore it \x60 -> ` (In slack, code snippets are surrounded by ``)
				codeText := regexp.MustCompile("\x60")
				isCodeText := codeText.MatchString(text)
				r := regexp.MustCompile("(.[A-Za-z0-9äëïöüÄËÏÖÜñÑ<>@.-]+?)([+-]+)$")
				matched := r.MatchString(trimmedWord)
				captureGroups := r.FindStringSubmatch(trimmedWord)
				if ev.User != info.User.ID && matched && !isCodeText {
					karmaWord := captureGroups[1]
					karmaModifier := captureGroups[2]
					log.Printf("Karma word: %s, Karma modifier: %s, Channel: %s", karmaWord, karmaModifier, channelName)
					// Get karmaModifier
					karmaCounter := 0
					switch karmaModifier {
					case "++":
						karmaCounter++
					case "--":
						karmaCounter--
					case "+++":
						karmaCounter += 2
					case "---":
						karmaCounter -= 2
					default:
						//ignore karma #ERRTOOMANYKARMA
						log.Printf("Karma modifier %s not allowed", karmaModifier)
						continue
					}

					if strings.HasPrefix(karmaWord, "<@") && strings.HasSuffix(karmaWord, ">") {
						// Check that users are not giving karma to theirselfs
						user := strings.ToLower("<@" + ev.User + ">")
						if user == karmaWord {
							log.Printf("User %s granted karma to theirself, skipping", user)
							continue
						}
						// User can have an alias configured
						alias := db.GetAlias(karmaWord, channelName)
						if len(alias) > 0 {
							log.Printf("User %s has an alias configured, skipping username retrieval", karmaWord)
						} else {
							karmaWord = utils.GetUsername(api, karmaWord)
						}
					}
					if karmaWord == "!here>" {
						log.Printf("@here detected, getting all users from the channel for the karma command")
						karmaWord = ""
						for _, member := range members {
							member = strings.ToLower("<@" + member + ">")
							// User can have an alias configured
							alias := db.GetAlias(member, channelName)
							if len(alias) > 0 {
								log.Printf("User %s has an alias configured, skipping username retrieval", member)
								karmaWord = alias
							} else {
								karmaWord = utils.GetUsername(api, member)
							}
							utils.HandleKarma(rtm, ev, db, karmaWord, channelName, karmaCounter)
						}
						// Continue to next loop iteration since karma for @here is already managed
						continue
					}
					utils.HandleKarma(rtm, ev, db, karmaWord, channelName, karmaCounter)
				}
			}

		case *slack.RTMError:
			log.Printf("Error %s\n", ev.Error())

		case *slack.InvalidAuthEvent:
			panic("Invalid credentials")

		default:
			continue
		}
	}
}
