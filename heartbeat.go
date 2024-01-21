package main

import (
	"bytes"
	"net/http"

	"go.uber.org/zap"
)

func sendHeartbeat() {

	managerURL := "http://" + managerAddr + "/cm_manager/v1.0/heartbeat"

	payload := []byte(`{"worker_id":"` + workerId + `"}`)

	resp, err := http.Post(managerURL, "application/json", bytes.NewBuffer(payload))
	if err != nil {
		logger.Debug("Error sending heartbeat", zap.Error(err))
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		logger.Debug("Heartbeat sent successfully")
	} else {
		logger.Debug("Unexpected status code", zap.Int("statusCode", resp.StatusCode))
	}
}
