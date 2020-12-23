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
            var groupInfo *slack.Group

            channelInformation, err := rtm.GetChannelInfo(ev.Channel)

            if err != nil {
                log.Printf("Error getting channel information. Error was: %s", err)
                log.Print("Trying to get channel information from GroupInfo API")
                groupInfo, err = rtm.GetGroupInfo(ev.Channel)
                if err != nil {
                    log.Printf("Error getting channel information from GroupInfo API. Error was: %s", err)
                    log.Print("Ignoring message since we cannot get channel information")
                    continue
                }
            }

            var channelName string
            var members []string
            if groupInfo != nil {
                channelName = groupInfo.NameNormalized
                members = groupInfo.Members
            } else {
                channelName = channelInformation.NameNormalized
                members = channelInformation.Members
            }

            text := ev.Text
            text = strings.TrimSpace(text)
            text = strings.ToLower(text)
            // Commands are implemented using a keyword rather than using slash commands to avoid
            // having to publish the bot in order to receive webhooks
            r := regexp.MustCompile("^(kb) (set|get|del) (karma|admin|setting|alias|help)(.*)$")
            matched := r.MatchString(text)
            if matched {
                captureGroups := r.FindStringSubmatch(text)
                operation := captureGroups[2]
                operationGroup := captureGroups[3]
                operationArgs := captureGroups[4]
                who := strings.ToLower(ev.User)
                if (operation == "get" && operationGroup == "help") {
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
                r := regexp.MustCompile("(.[A-Za-z0-9äëïöüÄËÏÖÜ<>@.-]+?)([+-]+)$")
                matched := r.MatchString(trimmedWord)
                captureGroups := r.FindStringSubmatch(trimmedWord)
                if ev.User != info.User.ID && matched {
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
                        karmaWord = utils.GetUsername(api, karmaWord)
                    }
                    if karmaWord == "!here>" {
                        log.Printf("@here detected, getting all users from the channel for the karma command")
                        karmaWord = ""
                        for _, member := range members {
                            member = "<@" + member + ">"
                            karmaWord = utils.GetUsername(api, member)
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
