package main

import (
    "github.com/mvazquezc/karma-bot/pkg/karmabot"
    "os"
    "log"
)

func main() {
    version := "1.0"
    log.Printf("Karma-bot version %s", version)
    apiToken := os.Getenv("API_TOKEN")
    // Check if database exists
    dbFile := "/var/tmp/karma.db"
    // Send db connection details to NewKarmaBot
    karmabot.NewKarmaBot(apiToken, dbFile)
}
