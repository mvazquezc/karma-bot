package utils

import (
	"log"
	"regexp"
	"strings"

	"github.com/mvazquezc/karma-bot/pkg/database"
	"github.com/slack-go/slack"
)


// FixEmptyKarma When user types @user and hits tab a space is inserted
// that ends up in a space between the user handler and the karma modifier
// this function will fix that by removing that space when detected
func FixEmptyKarma(text []string) []string {
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
            // We only want to fix the extra space added when pressing tab for autocomplete a user handler
            r := regexp.MustCompile("(<@)(.*)(>)")
            matched := r.MatchString(newWord)
            if matched {
                // The word preceding the karma modifiers and a space is a username, we want to fix it
                finalText[finalIndex-1] = newWord
            }
        } else {
            finalIndex++
            finalText = append(finalText, newWord)
        }
    }
    return finalText
}

// HandleKarma Updates the karma for a given word and sends a message if required
func HandleKarma(rtm *slack.RTM, ev *slack.MessageEvent, db database.Database, word string, channelName string, karmaCounter int) {

    alias := db.GetAlias(word, channelName)

    if len(alias) > 0 {
        log.Printf("Word %s has an alias configured, using alias %s", word, alias)
        word = alias
    }

    if karmaCounter != 0 {
        userKarma, notifyKarma := db.UpdateKarma(channelName, word, karmaCounter)
        if notifyKarma {
            karmaMessage := "`" + word + "` has `" + userKarma + "` karma points!"
            rtm.SendMessage(rtm.NewOutgoingMessage(karmaMessage, ev.Channel))
        }
    }
}

// GetUsername Queries the Slack API in order to get the configured name for a given user
func GetUsername(api *slack.Client, word string) string {
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
        displayName = strings.ToLower(user.Profile.RealName)
    }
    log.Printf("Display name for user %s is %s", userName, displayName)
    return strings.Replace(displayName, " ", ".", -1)
}

// PrintCommandsUsage Prints a help messages for implemented commands
func PrintCommandsUsage(rtm *slack.RTM, ev *slack.MessageEvent) {
    karmaHelp := "*Karma Commands*:\n- Add/Remove karma to the word's current karma: `kb set karma <word> <+karma|-karma>`\n- Get current karma for a given word: `kb get karma <word>`\n"
    adminHelp := "*Admin Commands*:\n- Set admin on current channel: `kb set admin @user`\n- Get admins on current channel: `kb get admin`\n- Remove admin on current channel: `kb del admin @user`\n"
    settingsHelp := "*Settings Commands*:\n- Set setting on current channel: `kb set setting <setting_name> <setting_value>`\n- Get setting value on current channel: `kb get setting <setting_name>`\n"
    aliasHelp := "*Alias Commands*:\n- Set alias for a given word on current channel: `kb set alias <word> <alias>`\n- Get aliases for a word on current channel: `kb get alias <word>`\n- Remove alias for a word: `kb del alias <word> <alias>`\n"
    commandsHelp := karmaHelp + adminHelp + settingsHelp + aliasHelp
    rtm.SendMessage(rtm.NewOutgoingMessage(commandsHelp, ev.Channel))
}
