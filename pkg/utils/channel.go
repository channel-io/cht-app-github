package utils

import (
	"fmt"
	"regexp"
	"strings"
)

// TODO 아래 utils 검토 & 개선

const (
	teamChatPathRegex = "^(%s\\/#\\/channels\\/[0-9]+\\/team_chats\\/groups\\/[0-9]+\\/)(.*)$"
)

func ParseMessageIdFromTeamChatDeskUrl(deskUrl, uri string) string {
	escaped := strings.Replace(deskUrl, "/", "\\/", -1)
	deskUrlRegex := fmt.Sprintf(teamChatPathRegex, escaped)
	matches := regexp.MustCompile(deskUrlRegex).FindStringSubmatch(uri)
	return matches[2]
}

func IsChannelTeamChatUriFormat(deskUrl, uri string) bool {
	deskUrlRegex := fmt.Sprintf(teamChatPathRegex, deskUrl)
	return regexp.MustCompile(deskUrlRegex).Match([]byte(uri))
}

func ChannelTalkTeamChatFormat(deskUrl, channelId, groupId, rootMessageId string) string {
	return fmt.Sprintf("%s/#/channels/%s/team_chats/groups/%s/%s", deskUrl, channelId, groupId, rootMessageId)
}
