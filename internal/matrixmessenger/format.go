package matrixmessenger

import (
	"fmt"
	"regexp"
	"strings"
)

// StripReply removes the quoted reply from a message
func StripReply(msg string) string {
	strippedMsg := strings.Builder{}
	for _, line := range strings.Split(msg, "\n") {
		if strings.HasPrefix(line, ">") {
			continue
		}
		strippedMsg.WriteString(line)
	}

	return strippedMsg.String()
}

// StripReplyFormatted removes the quoted reply from a message
func StripReplyFormatted(msg string) string {
	re := regexp.MustCompile(`(?s)<mx-reply>.*?<\/mx-reply>`)
	return re.ReplaceAllString(msg, "")
}

var regexUsernameFromUserID = regexp.MustCompile("@([^:]+)")

// GetMatrixLinkForUser creates a clickable link pointing to the given user id
func GetMatrixLinkForUser(userID string) string {
	link := fmt.Sprintf(`<a href="https://matrix.to/#/%s">%s</a>`, userID, regexUsernameFromUserID.Find([]byte(userID)))

	return link
}

// GetHomeserverFromUserID returns the homeserver from a user id
func GetHomeserverFromUserID(userID string) string {
	if !strings.Contains(userID, ":") {
		return "matrix.org"
	}

	return strings.Split(userID, ":")[1]
}

// GetUSerNameFromUserIdentififer extracts the username from the user identifier string.
// E.g. @testuser:matrix.org will result in testuser
func GetUsernameFromUserIdentifier(userID string) string {
	if !strings.Contains(userID, ":") {
		return strings.TrimPrefix(userID, "@")
	}

	return strings.TrimPrefix(strings.Split(userID, ":")[0], "@")
}
