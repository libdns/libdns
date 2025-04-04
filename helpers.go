package mijnhost

import (
	"strings"
)

func normalizeZone(zone string) string {
	return strings.TrimSuffix(strings.Replace(zone, "*.", "", 1), ".")
}
