//go:build windows

package main

import "os"

func diskAvailability() availability {
	if _, err := os.Stat("/host-root"); err != nil {
		return availabilityUnavailable
	}
	return availabilityOperational
}
