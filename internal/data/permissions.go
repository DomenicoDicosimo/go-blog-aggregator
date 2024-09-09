package data

import "slices"

func Include(permissions []string, code string) bool {
	return slices.Contains(permissions, code)
}
