package main

import (
	goversion "github.com/hashicorp/go-version"
	"strings"
)

func isNewerVersion(latestVersion, currentVersion string) bool {
	latest, err := goversion.NewVersion(strings.TrimPrefix(latestVersion, "v"))
	if err != nil {
		return latestVersion != currentVersion
	}

	current, err := goversion.NewVersion(strings.TrimPrefix(currentVersion, "v"))
	if err != nil {
		return latestVersion != currentVersion
	}

	return latest.GreaterThan(current)
}
