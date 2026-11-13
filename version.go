package main

import (
	"fmt"
	"strconv"
	"strings"
)

// appVersion is the version reported by the app and compared with the latest
// GitHub release. Override it at build time with:
//
//	go build -ldflags "-X main.appVersion=1.2.3"
var appVersion = "2.0.0"

const releasesRepository = "munsif-solkar/stockbook-desktop"
const releasesAPI = "https://api.github.com/repos/" + releasesRepository + "/releases/latest"
const releasesPage = "https://github.com/" + releasesRepository + "/releases"

func (a *App) Version() string {
	return appVersion
}

func normalizeVersion(value string) ([]int, error) {
	trimmed := strings.TrimSpace(value)
	trimmed = strings.TrimPrefix(strings.TrimPrefix(trimmed, "v"), "V")
	if trimmed == "" {
		return nil, fmt.Errorf("empty version")
	}
	if index := strings.IndexAny(trimmed, "-+"); index >= 0 {
		trimmed = trimmed[:index]
	}
	parts := strings.Split(trimmed, ".")
	if len(parts) == 0 || len(parts) > 4 {
		return nil, fmt.Errorf("invalid version %q", value)
	}
	numbers := make([]int, len(parts))
	for i, part := range parts {
		if part == "" {
			return nil, fmt.Errorf("invalid version %q", value)
		}
		number, err := strconv.Atoi(part)
		if err != nil || number < 0 {
			return nil, fmt.Errorf("invalid version %q", value)
		}
		numbers[i] = number
	}
	return numbers, nil
}

// compareVersions returns -1 when a is older than b, 0 when they are equal,
// and 1 when a is newer than b. Missing components count as zero, so 1.2 and
// 1.2.0 are equal.
func compareVersions(a, b string) (int, error) {
	left, err := normalizeVersion(a)
	if err != nil {
		return 0, err
	}
	right, err := normalizeVersion(b)
	if err != nil {
		return 0, err
	}
	length := len(left)
	if len(right) > length {
		length = len(right)
	}
	for i := 0; i < length; i++ {
		var x, y int
		if i < len(left) {
			x = left[i]
		}
		if i < len(right) {
			y = right[i]
		}
		if x < y {
			return -1, nil
		}
		if x > y {
			return 1, nil
		}
	}
	return 0, nil
}

func isNewerVersion(latest, current string) (bool, error) {
	comparison, err := compareVersions(latest, current)
	if err != nil {
		return false, err
	}
	return comparison > 0, nil
}
