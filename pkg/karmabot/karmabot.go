package karmabot

import (
    "github.com/mvazquezc/karma-bot/pkg/database"
    "github.com/mvazquezc/karma-bot/pkg/commands"
    "github.com/nlopes/slack"
    "regexp"
    "strings"
    "log"
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
            if groupInfo != nil {
                channelName = groupInfo.NameNormalized
            } else {
                channelName = channelInformation.NameNormalized
            }
            
            text := ev.Text
            text = strings.TrimSpace(text)
            text = strings.ToLower(text)
            // Commands are implemented using a keyword rather than using slash commands to avoid
            // having to publish the bot in order to receive webhooks
            r := regexp.MustCompile("^(kb) (set|get|del) (karma|admin|setting|alias)(.*)$")
            matched := r.MatchString(text)
            if matched {
                captureGroups := r.FindStringSubmatch(text)
                operation := captureGroups[2]
                operationGroup := captureGroups[3]
                operationArgs := captureGroups[4]
                who := strings.ToLower(ev.User)
                // add user that fires the command to the args
                commandOutput := commands.ProcessCommand(channelName, who, operation, operationGroup, operationArgs)
                rtm.SendMessage(rtm.NewOutgoingMessage(commandOutput, ev.Channel))
            }
            splitText := strings.Fields(text)
            splitText = fixEmptyKarma(splitText)
            for _, word := range splitText {
                r := regexp.MustCompile("(.[A-Za-z0-9<>@.]+?)([+-]+)$")
                matched := r.MatchString(word)
                captureGroups := r.FindStringSubmatch(word)
                if ev.User != info.User.ID && matched {
                    karmaWord := captureGroups[1]
                    karmaModifier := captureGroups[2]
                    if strings.HasPrefix(karmaWord, "<@") && strings.HasSuffix(karmaWord, ">") {
                        karmaWord = getUsername(api, karmaWord)
                    }
                    // Todo: create a logger and use it :D
                    log.Printf("Karma word: %s, Karma modifier: %s, Channel: %s", karmaWord, karmaModifier, channelName)

                    alias := db.GetAlias(karmaWord, channelName)
                    if len(alias) > 0 {
                        log.Printf("Word %s has an alias configured, using alias %s", karmaWord, alias)
                        karmaWord = alias
                    }
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
                    if karmaCounter != 0 {
                        userKarma, notifyKarma := db.UpdateKarma(channelName, karmaWord, karmaCounter)
                        if notifyKarma {
                            karmaMessage := "`" + karmaWord + "` has `" + userKarma + "` karma points!"
                            rtm.SendMessage(rtm.NewOutgoingMessage(karmaMessage, ev.Channel))
                        }
                    }
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

// fixEmptyKarma When user types @user and hits tab a space is inserted
// that ends up in a space between the user handler and the karma modifier
// this function will fix that by removing that space when detected
func fixEmptyKarma(text []string) []string {
    karmaModifiers := map[string]bool {
        "++": true,
        "--": true,
        "+++": true,
        "---": true,
    }
    var finalText []string
    var finalIndex int = 0
    for index, word := range text {
        var newWord string = word;
        if karmaModifiers[word] && index>0 && !strings.HasSuffix(finalText[finalIndex-1], word) {
            newWord = text[index-1] + newWord
            finalText[finalIndex-1] = newWord
        } else {
            finalIndex++
            finalText = append(finalText, newWord)
        }
    }
    return finalText
}

func getUsername(api *slack.Client, word string) string {
    log.Printf("Getting username for user %s", word)
    displayName := "not_set"
    r := regexp.MustCompile("(<@)(.*)(>)")
    captureGroups := r.FindStringSubmatch(word)
    userName := captureGroups[2]
    userName = strings.ToUpper(userName)
    user, err := api.GetUserInfo(userName)
    if err != nil {
        panic(err)
    }
    if len(user.Profile.DisplayNameNormalized) > 0 {
        displayName = strings.ToLower(user.Profile.DisplayNameNormalized)
    } else {
        displayName = strings.ToLower(user.Profile.FirstName)
    }
    log.Printf("Display name for user %s is %s", userName, displayName)
    return strings.Replace(displayName, " ", ".", -1)
}

// Implement commands for setting karma, alias, etc..
// https://github.com/nlopes/slack/blob/master/examples/slash/slash.go
// https://api.slack.com/apps/AP5MM8YC8/slash-commands?