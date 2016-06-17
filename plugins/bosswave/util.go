package bosswave

import (
	"strings"
)

func getMetadataKey(uri string) string {
	parts := strings.Split(uri, "/")
	if len(parts) == 0 {
		return ""
	}
	return parts[len(parts)-1]
}
