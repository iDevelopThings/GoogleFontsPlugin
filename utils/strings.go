package utils

import (
	"strings"
)

func GetPathSafeName(value string) string {
	return strings.ToLower(strings.ReplaceAll(value, " ", ""))
}
