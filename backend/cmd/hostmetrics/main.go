package main

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type availability string

const (
	availabilityOperational availability = "operational"
	availabilityDegraded    availability = "degraded"
	availabilityUnavailable availability = "unavailable"
)

type statusResponse struct {
	CPU     availability `json:"cpu"`
	Memory  availability `json:"memory"`
	Disk    availability `json:"disk"`
	Network availability `json:"network"`
	Uptime  availability `json:"uptime"`
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/status", statusHandler)
	server := &http.Server{Addr: ":8090", Handler: mux}
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		panic(err)
	}
}

func statusHandler(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodGet {
		writer.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	writer.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(writer).Encode(statusResponse{
		CPU: cpuAvailability(), Memory: memoryAvailability(), Disk: diskAvailability(), Network: networkAvailability(), Uptime: uptimeAvailability(),
	})
}

func readProc(name string) ([]byte, error) {
	return os.ReadFile(filepath.Join("/host/proc", name))
}

func cpuAvailability() availability {
	data, err := readProc("loadavg")
	if err != nil {
		return availabilityUnavailable
	}
	fields := strings.Fields(string(data))
	if len(fields) == 0 {
		return availabilityUnavailable
	}
	load, err := strconv.ParseFloat(fields[0], 64)
	if err != nil {
		return availabilityUnavailable
	}
	if load >= 4 {
		return availabilityDegraded
	}
	return availabilityOperational
}

func memoryAvailability() availability {
	data, err := readProc("meminfo")
	if err != nil {
		return availabilityUnavailable
	}
	var total, available float64
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		value, err := strconv.ParseFloat(fields[1], 64)
		if err != nil {
			continue
		}
		switch fields[0] {
		case "MemTotal:":
			total = value
		case "MemAvailable:":
			available = value
		}
	}
	if total <= 0 {
		return availabilityUnavailable
	}
	if available/total < .15 {
		return availabilityDegraded
	}
	return availabilityOperational
}

func networkAvailability() availability {
	data, err := readProc("net/dev")
	if err != nil || !strings.Contains(string(data), ":") {
		return availabilityUnavailable
	}
	return availabilityOperational
}

func uptimeAvailability() availability {
	data, err := readProc("uptime")
	if err != nil || len(strings.Fields(string(data))) == 0 {
		return availabilityUnavailable
	}
	return availabilityOperational
}
