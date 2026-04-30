package chat

import (
	"sort"
	"strings"
)
// otherUserFromChatID extracts the other participant's ID from a private chat_id.

func otherUserFromChatID(chatID, myID string) string {
	trimmed := strings.TrimPrefix(chatID, "private:")
	parts := strings.SplitN(trimmed, ":", 2)
	if len(parts) != 2 {
		return ""
	}
	if parts[0] == myID {
		return parts[1]
	}
	return parts[0]
}
 
// PrivateChatID creates a deterministic chat ID for two users.
// Format: "private:<lower_id>:<higher_id>"
func PrivateChatID(a, b string) string {
	ids := []string{a, b}
	sort.Strings(ids)
	return "private:" + ids[0] + ":" + ids[1]
}
