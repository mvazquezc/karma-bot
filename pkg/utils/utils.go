package utils

import (
	"log"
	"regexp"
	"strconv"
	"strings"

	"github.com/mvazquezc/karma-bot/pkg/database"
	"github.com/slack-go/slack"
)

// FixEmptyKarma When user types @user and hits tab a space is inserted
// that ends up in a space between the user handler and the karma modifier
// this function will fix that by removing that space when detected
func FixEmptyKarma(text []string) []string {
	karmaModifiers := map[string]bool{
		"++":  true,
		"--":  true,
		"+++": true,
		"---": true,
	}
	var finalText []string
	var finalIndex int = 0
	for index, word := range text {
		var newWord string = word

		if karmaModifiers[word] && index > 0 && !strings.HasSuffix(finalText[finalIndex-1], word) {
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

	useKarmaEmojisSetting := db.GetSetting(channelName, "use_karma_emojis")

	if len(useKarmaEmojisSetting) <= 0 {
		useKarmaEmojisSetting = "0"
	}
	useKarmaEmojis, _ := strconv.Atoi(useKarmaEmojisSetting)

	user := strings.ToLower("<@" + ev.User + ">")
	userAlias := db.GetAlias(user, channelName)

	if word == userAlias {
		// Check that user is not giving karma to one of their aliases
		log.Printf("User %s granted karma to theirself, skipping", user)
		return
	}

	if len(alias) > 0 {
		log.Printf("Word %s has an alias configured, using alias %s", word, alias)
		word = alias
	}

	if karmaCounter != 0 {
		wordKarma, notifyKarma, intWordKarma := db.UpdateKarma(channelName, word, karmaCounter)
		// Only send emojis if those are enabled in the channel
		karmaEmoji := ""
		globalKarmaMsg := ""
		if useKarmaEmojis == 1 {
			karmaEmoji = ":thumbsup:"
			if karmaCounter < 0 {
				karmaEmoji = ":thumbsdown:"
			}
		}
		if notifyKarma {
			// Get Global Karma
			globalKarma := db.GetGlobalKarma(word)
			log.Printf("Word karma %d, global karma %d", intWordKarma, globalKarma)
			// We only want to add the global karma if the word has karma outside this channel
			if (globalKarma > intWordKarma || globalKarma < intWordKarma) && globalKarma != 0 {
				globalKarmaStr := strconv.Itoa(globalKarma)
				globalKarmaMsg = "(`" + globalKarmaStr + "` points across channels) "
			}
			karmaMessage := "`" + word + "` has `" + wordKarma + "` karma points! " + globalKarmaMsg + karmaEmoji
			resp := rtm.NewOutgoingMessage(karmaMessage, ev.Channel)
			// Check if message is from a thread, and if so set the response to be in-thread
			if ev.Msg.ThreadTimestamp != "" {
				resp.ThreadTimestamp = ev.Msg.ThreadTimestamp
			} else { // Reply in a new thread otherwise
				resp.ThreadTimestamp = ev.Msg.Timestamp
			}
			rtm.SendMessage(resp)
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
	karmaHelp := "*Karma Commands*:\n- Add/Remove karma to the word's current karma: `kb set karma <word> <+karma|-karma>`\n- Get current karma for a given word: `kb get karma <word>`\n- Get current karma ranking for the channel: `kb rank karma [all]`\n"
	adminHelp := "*Admin Commands*:\n- Set admin on current channel: `kb set admin @user`\n- Get admins on current channel: `kb get admin`\n- Remove admin on current channel: `kb del admin @user`\n"
	settingsHelp := "*Settings Commands*:\n- Set setting on current channel: `kb set setting <setting_name> <setting_value>`\n- Get setting value on current channel: `kb get setting <setting_name>`\n"
	aliasHelp := "*Alias Commands*:\n- Set alias for a given word on current channel: `kb set alias <word> <alias>`\n- Get aliases for a word on current channel: `kb get alias <word>`\n- Remove alias for a word: `kb del alias <word> <alias>`\n"
	rankHelp := "*Rank Commands*:\n- Get top 10 words on current channel: `kb rank karma`\n- Get full rank of words on current channel: `kb rank karma all`\n- Get top 10 words rank of words across channels: `kb rank globalkarma`\n- Get full rank of words across channels: `kb rank globalkarma all`"
	commandsHelp := karmaHelp + adminHelp + settingsHelp + aliasHelp + rankHelp
	rtm.SendMessage(rtm.NewOutgoingMessage(commandsHelp, ev.Channel))
}
