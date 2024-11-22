package callback

import (
	"github.com/google/go-github/v60/github"
)

func isSentFromBot(sender *github.User) bool {
	if sender != nil && sender.GetType() == "Bot" {
		return true
	}
	return false
}
