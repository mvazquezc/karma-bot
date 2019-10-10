package main

import (
	"github.com/mvazquezc/karma-bot/pkg/cmd/karmabot"
	"os"
)

func main() {

	apiToken := os.Getenv("API_TOKEN")
	karmabot.NewKarmaBot(apiToken)
}
