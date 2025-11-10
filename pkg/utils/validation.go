package utils

import "regexp"

func IsId(id string) bool {
	re := regexp.MustCompile(`^[0-9a-fA-F]{2}(?:-[0-9a-fA-F]{2}){5}$`)
	return re.MatchString(id)
}
