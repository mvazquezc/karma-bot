package main

import (
	"log"
	"os"

	"github.com/mvazquezc/karma-bot/pkg/karmabot"
)

func main() {
    version := "1.2"
    log.Printf("Karma-bot version %s", version)
    apiToken := os.Getenv("API_TOKEN")
    // Check if database exists
    dbFile := "/var/tmp/karma.db"
    // Send db connection details to NewKarmaBot
    karmabot.NewKarmaBot(apiToken, dbFile)
}
