//go:build linux

package main

import "syscall"

func diskAvailability() availability {
	var filesystem syscall.Statfs_t
	if err := syscall.Statfs("/host-root", &filesystem); err != nil || filesystem.Blocks == 0 {
		return availabilityUnavailable
	}
	if float64(filesystem.Bavail)/float64(filesystem.Blocks) < .15 {
		return availabilityDegraded
	}
	return availabilityOperational
}
