package commands

import (
	"log"
	"strconv"
	"strings"
    "sort"

	"github.com/mvazquezc/karma-bot/pkg/database"
)

// Commands type
type Commands struct {
    db *database.Database
}

// New Settings constructor
func New(database *database.Database) Commands {
    commands := Commands{db: database}
    return commands
}

// ProcessCommand processes a command
func (cmd *Commands) ProcessCommand(channel string, who string, operation string, operationGroup string, operationArgs string) string {
    //trim spaces from the args
    operationArgs = strings.TrimSpace(operationArgs)
    log.Printf("Processing operation: %s, operationGroup: %s, operationArgs: %s, in channel %s sent by user %s", operation, operationGroup, operationArgs, channel, who)
    var commandOutput string
    switch operationGroup {
    case "karma":
        if operation == "set" {
            commandOutput = cmd.setKarma(channel, operationArgs, who)
        } else if operation == "rank" {
            commandOutput = cmd.getKarmaRank(channel, operationArgs)
        } else {
            commandOutput = cmd.getKarma(channel, operationArgs)
        }
    case "admin":
        if operation == "set" {
            commandOutput = cmd.setAdmin(channel, operationArgs, who)
        } else if operation == "get" {
            _, commandOutput = cmd.getAdmins(channel)
        } else {
            commandOutput = cmd.delAdmin(channel, operationArgs, who)
        }
    case "setting":
        if operation == "set" {
            commandOutput = cmd.setSetting(channel, operationArgs, who)
        } else {
            commandOutput = cmd.getSetting(channel, operationArgs)
        }
    case "alias":
        if operation == "set" {
            commandOutput = cmd.setAlias(channel, operationArgs, who)
        } else if operation == "get" {
            
            commandOutput = cmd.getAlias(channel, operationArgs)
        } else {
            commandOutput = cmd.delAlias(channel, operationArgs, who)
        }
    default:
        log.Printf("Unknown operationGroup %s", operationGroup)
        break
    }
    return commandOutput
}

// usage: kb set setting setting_name setting_value
func (cmd *Commands) setSetting(channel string, parameters string, who string) string {
    var commandResult string
    admins, _ := cmd.getAdmins(channel)
    requesterIsAdmin := contains(admins, who)
    if requesterIsAdmin {
        // We expect parameters to have something like "setting_name setting_value" so we need to check that
        params := strings.Fields(parameters)
        if len(params) != 2 {
            log.Printf("Received more than 2 parameters. Params: %s", parameters)
            commandResult = "Incorrect parameters. Usage kb set setting setting_name integer_setting_value :warning:"
        } else {
            settingName := params[0]
            settingValue := params[1]
            // We need to ensure the setting is within the valid settings list
            validSettings := []string{"notify_karma"}
            validSetting := contains(validSettings, settingName)
            if validSetting {
                // Convert string to int to ensure we received a setting value
                _, err := strconv.Atoi(settingValue)
                if err != nil {
                    log.Printf("Received incorrect setting value %s", settingValue)
                    commandResult = "Incorrect parameters. Usage kb set setting setting_name integer_setting_value :warning:"
                } else {
                    log.Printf("Received setting %s and setting value %s", settingName, settingValue)
                    cmd.db.SetSetting(channel, settingName, settingValue)
                    log.Printf("Setting %s configured to %s", settingName, settingValue)
                    commandResult = "User <@" + strings.ToUpper(who) + "> configured setting `" + settingName + "` to `" + settingValue + "` on this channel :white_check_mark:"
                }
            } else {
                log.Printf("Received incorrect setting %s", settingName)
                commandResult = "Incorrect setting name, setting `" + settingName + "` is not a valid setting :warning:"
            }
        }
    } else {
        log.Printf("Requester user %s, is not admin on channel %s. Operation canceled", who, channel)
        commandResult = "User <@" + strings.ToUpper(who) + "> has no permissions to set settings on this channel :no_entry_sign:"
    }
    return commandResult
}

// getSetting returns the value for a setting in a given channel
// usage: kb get setting setting_name
func (cmd *Commands) getSetting(channel string, parameters string) string {
    log.Printf("Getting value for setting %s in channel %s", parameters, channel)
    settings := strings.Fields(parameters)
    var commandResult string
    for _, a := range settings {
        settingValue := cmd.db.GetSetting(channel, a)
        if len(settingValue) <= 0 {
            log.Printf("Setting %s does not exist", a)
            commandResult += "Setting `" + a + "` is not configured\n"
        } else {
            log.Printf("Setting %s is configured to %s", a, settingValue)
            commandResult += "Setting `" + a + "` is configured to `" + settingValue + "`\n"
        }
    }
    return commandResult
}

// usage: kb set karma word karmaValue
func (cmd *Commands) setKarma(channel string, parameters string, who string) string {
    var commandResult string
    admins, _ := cmd.getAdmins(channel)
    requesterIsAdmin := contains(admins, who)
    if requesterIsAdmin {
        // We expect parameters to have something like "word karmaValue" so we need to check that
        params := strings.Fields(parameters)
        if len(params) != 2 {
            log.Printf("Received more than 2 parameters. Params: %s", parameters)
            commandResult = "Incorrect parameters. Usage kb set karma word integer :warning:"
        } else {
            word := params[0]
            karmaValue := params[1]
            // Convert string to int to ensure we received a valid karma value
            karmaValueInt, err := strconv.Atoi(karmaValue)
            if err != nil {
                log.Printf("Received incorrect karma value %s", karmaValue)
                commandResult = "Incorrect parameters. Usage kb set karma word integer :warning:"
            } else {
                log.Printf("Received word %s and karma value %s", word, karmaValue)
                finalKarma, _ := cmd.db.UpdateKarma(channel, word, karmaValueInt)
                log.Printf("Karma for word %s updated to %s", word, finalKarma)
                commandResult = "User <@" + strings.ToUpper(who) + "> set karma for word `" + word + "` to `" + finalKarma + "` on this channel :white_check_mark:"
            }
        }
    } else {
        log.Printf("Requester user %s, is not admin on channel %s. Operation canceled", who, channel)
        commandResult = "User <@" + strings.ToUpper(who) + "> has no permissions to set karma on this channel :no_entry_sign:"
    }
    return commandResult
}

// usage: kb get karma word/s
func (cmd *Commands) getKarma(channel string, args string) string {
    log.Printf("Getting karma for words %s in channel %s", args, channel)
    words := strings.Fields(args)
    var commandResult string
    for _, a := range words {
        // Get alias for the word
        alias := cmd.db.GetAlias(a, channel)
        if len(alias) > 0 {
            log.Printf("Word %s has an alias configured, using alias %s", a, alias)
            a = alias
        }
        karmaValue := cmd.db.GetCurrentKarma(channel, a)
        if karmaValue == -256256 {
            karmaValue = 0
        }
        result := strconv.Itoa(karmaValue)
        commandResult += "`" + a + "` has `" + result + "` karma points!\n"
    }
    return commandResult
}

// usage: kb rank karma [all], we return top10 words by default
func (cmd *Commands) getKarmaRank(channel string, args string) string {
    log.Printf("Getting karma rank in channel %s", channel)
    var commandResult string
    getAll := false
    if args == "all" {
        getAll = true
    }
    rank := cmd.db.GetKarmaRank(channel, getAll)
    // Rank is an ordered map, we need to order it (https://code-maven.com/slides/golang/sort-map-by-value)
    ranks := make([]string, 0, len(rank))
    for word := range rank {
        ranks = append(ranks, word)
    }
    sort.Slice(ranks, func(i, j int) bool {
        return rank[ranks[i]] > rank[ranks[j]]
    })
    commandResult = ":trophy: Karma Rank :trophy: \n"
    var karmaValue string
    for _, word := range ranks {
        karmaValue = strconv.Itoa(rank[word])
        commandResult += "  `" + word + " (" + karmaValue + ")`\n"
    }
    return commandResult
}

// usage: kb set alias word alias
func (cmd *Commands) setAlias(channel string, parameters string, who string) string {
    var commandResult string
    admins, _ := cmd.getAdmins(channel)
    requesterIsAdmin := contains(admins, who)
    if requesterIsAdmin {
        // We expect parameters to have something like "word alias" so we need to check that
        params := strings.Fields(parameters)
        if len(params) != 2 {
            log.Printf("Received more than 2 parameters. Params: %s", parameters)
            commandResult = "Incorrect parameters. Usage kb set alias word alias :warning:"
        } else {
            word := params[0]
            alias := params[1]
            log.Printf("Received word %s and alias %s", word, alias)
            if alias != word {
                aliasCreated := cmd.db.SetAlias(word, alias, channel)
                if aliasCreated == 0 {
                    log.Printf("Alias %s configured for word %s", alias, word)
                    commandResult = "User <@" + strings.ToUpper(who) + "> configured alias `" + alias + "` for word `" + word + "` on this channel :white_check_mark:"
                } else if aliasCreated == 1 {
                    log.Printf("Alias %s already exists for word %s", alias, word)
                    commandResult = "Alias `" + alias + "` for word `" + word + "` already exists on this channel :warning:"
                } else {
                    log.Printf("Word %s is already in use as an alias in this channel, operation not permitted", word)
                    commandResult = "Word `" + word + "` is already in use as an alias in this channel, operation not permitted :no_entry_sign:"
                }
            } else {
                log.Printf("Invalid alias %s for word %s", alias, word)
                commandResult = "Invalid alias `" + alias + "` for word `" + word + "` :warning:"
            }
        }
    } else {
        log.Printf("Requester user %s, is not admin on channel %s. Operation canceled", who, channel)
        commandResult = "User <@" + strings.ToUpper(who) + "> has no permissions to set alias on this channel :no_entry_sign:"
    }
    return commandResult
}

// usage: kb del alias word alias
func (cmd *Commands) delAlias(channel string, parameters string, who string) string {
    var commandResult string
    admins, _ := cmd.getAdmins(channel)
    requesterIsAdmin := contains(admins, who)
    if requesterIsAdmin {
        // We expect parameters to have something like "word alias" so we need to check that
        params := strings.Fields(parameters)
        if len(params) != 2 {
            log.Printf("Received more than 2 parameters. Params: %s", parameters)
            commandResult = "Incorrect parameters. Usage kb del alias word alias :warning:"
        } else {
            word := params[0]
            alias := params[1]
            log.Printf("Received word %s and alias %s", word, alias)
            if alias != word {
                cmd.db.DelAlias(channel, word, alias)
                log.Printf("Alias %s deleted for word %s", alias, word)
                commandResult = "User <@" + strings.ToUpper(who) + "> deleted alias `" + alias + "` for word `" + word + "` on this channel :white_check_mark:"
            } else {
                log.Printf("Invalid alias %s for word %s", alias, word)
                commandResult = "Invalid alias `" + alias + "` for word `" + word + "` :warning:"
            }
        }
    } else {
        log.Printf("Requester user %s, is not admin on channel %s. Operation canceled", who, channel)
        commandResult = "User <@" + strings.ToUpper(who) + "> has no permissions to delete alias on this channel :no_entry_sign:"
    }
    return commandResult
}


// usage: kb get alias word
func (cmd *Commands) getAlias(channel string, parameters string) string {
    log.Printf("Getting alias for word %s in channel %s", parameters, channel)
    words := strings.Fields(parameters)
    var commandResult string
    for _, a := range words {
        alias := cmd.db.GetAlias(a, channel)
        if len(alias) <= 0 {
            log.Printf("Setting %s does not exist", a)
            commandResult += "Word `" + a + "` has no alias configured\n"
        } else {
            log.Printf("Word %s has alias %s configured", a, alias)
            commandResult += "Word `" + a + "` has alias `" + alias + "` configured\n"
        }
    }
    return commandResult
}

// usage: kb del admin @adminuser
func (cmd *Commands) delAdmin(channel string, user string, who string) string {
    var commandResult string
    admins, _ := cmd.getAdmins(channel)
    if strings.HasPrefix(user, "<@") && strings.HasSuffix(user, ">") {
        log.Printf("Detected user %s, removing special chars", user)
        user = strings.Replace(user, "<", "", -1)
        user = strings.Replace(user, ">", "", -1)
        user = strings.Replace(user, "@", "", -1)
        log.Printf("Final user: %s", user)
        if len(admins) == 0 {
            log.Println("Channel has no admins")
            commandResult = "Channel has no admins configured. Deletion canceled. :warning:"
        } else {
            // Check if user requesting an admin addition is admin for the channel
            requesterIsAdmin := contains(admins, who)
            if requesterIsAdmin {
                // Check that the user being deleted from admin is admin already
                adminAlreadyExists := contains(admins, user)
                if adminAlreadyExists {
                    log.Printf("User %s is already admin for channel %s, deleting it from admins users", user, channel)
                    cmd.db.DeleteAdmin(channel, user)
                    commandResult = "User <@" + strings.ToUpper(user) + "> deleted from admins for this channel :white_check_mark:"
                } else {
                    log.Printf("User %s is not configured as admin for channel %s. Deletion canceled.", user, channel)
                    commandResult = "User <@" + strings.ToUpper(user) + "> is not admin for this channel. Deletion canceled. :warning:"
                }
            } else {
                log.Printf("Requester user %s, is not admin on channel %s. Operation canceled", who, channel)
                commandResult = "User <@" + strings.ToUpper(who) + "> has no permissions to delete admins from this channel :no_entry_sign:"
            }
        } 
    } else {
        log.Printf("No user detected, received %s as user", user)
        commandResult = "No user detected. Usage kb del admin @user :warning:"
    }
    return commandResult
}

// usage: kb set admin @user
func (cmd *Commands) setAdmin(channel string, user string, who string) string {
    //if not admin exists for a channel, the first user can set user as himself
    //if an admin already exists, only the admin can set other admins
    var commandResult string
    admins, _ := cmd.getAdmins(channel)
    if strings.HasPrefix(user, "<@") && strings.HasSuffix(user, ">") {
        log.Printf("Detected user %s, removing special chars", user)
        user = strings.Replace(user, "<", "", -1)
        user = strings.Replace(user, ">", "", -1)
        user = strings.Replace(user, "@", "", -1)
        log.Printf("Final user: %s", user)
        if len(admins) == 0 {
            log.Println("No admins exists, we can create one")
            cmd.db.CreateAdmin(channel, user)
            log.Printf("Admin %s configured as first admin for channel %s", user, channel)
            commandResult = "User <@" + strings.ToUpper(user) + "> configured as admin :white_check_mark:"
        } else {
            // Check if user requesting an admin addition is admin for the channel
            requesterIsAdmin := contains(admins, who)
            if requesterIsAdmin {
                // Check that the user being configured as admin is not admin already
                adminAlreadyExists := contains(admins, user)
                if adminAlreadyExists {
                    log.Printf("User %s is already admin for channel %s", user, channel)
                    commandResult = "User <@" + strings.ToUpper(user) + "> is already an admin for this channel :warning:"
                } else {
                    cmd.db.CreateAdmin(channel, user)
                    log.Printf("User %s configured admin for channel %s by user %s", user, channel, who)
                    commandResult = "User <@" + strings.ToUpper(user) + "> configured as admin for this channel :white_check_mark:"
                }
            } else {
                log.Printf("Requester user %s, is not admin on channel %s. Operation canceled", who, channel)
                commandResult = "User <@" + strings.ToUpper(who) + "> has no permissions to configure admins for this channel :no_entry_sign:"
            }
        }
    } else {
        log.Printf("No user detected, received %s as user", user)
        commandResult = "No user detected. Usage kb set admin @user :warning:"
    }
    return commandResult
}

// usage: kb get admin
func (cmd *Commands) getAdmins(channel string) (admins []string, commandResult string) {
    log.Printf("Getting admins for channel %s", channel)
    admins = cmd.db.GetAdmins(channel)
    if len(admins) > 0 {
        commandResult = "Admins configured in this channel:\n"
        for _, a := range admins {
            commandResult += "* <@" + strings.ToUpper(a) + ">\n"
        }
    } else {
        commandResult = "No admins configured for this channel yet"
    }
    return admins, commandResult
}

// helper function that returns true if a string exists in a slice
func contains(s []string, e string) bool {
    for _, a := range s {
        if a == e {
            return true
        }
    }
    return false
}