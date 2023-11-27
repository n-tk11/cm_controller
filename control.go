package main

import (
	"fmt"
	"sync"
)

type Service struct {
	ContainerName string
	ContainerId   string
	Image         string
	DaemonPort    string
	Status        string //running,checkpointed,standby,exited
}

var services = make(map[string]Service)
var mu sync.Mutex

func serviceSubscribe(containerName string, containerId string, image string, daemonPort string) Service {
	newService := Service{containerName, containerId, image, daemonPort, "new"}
	services[containerName] = newService
	return newService
}

func serviceUnsubscribe(containerName string, image string) int {
	if _, ok := services[containerName]; ok {
		delete(services, containerName)
	} else {
		fmt.Printf("No container name %s.\n", containerName)
		return 1
	}

	return 0
}

// TODO: This function will first check if all subscribed containers is still running(ffdaemon is alive) if not it will update the status to "stopped" / report all subscribed containers (including its status) in json
func getAllServices() {

}

// TODO: A method to get a service status from its name
// Things to carefully think about: stopped, no container
// This method will also autoupdate if container dead/stopped/exited
func (s Service) getContainerStatus(containerName string) string {
	containerInfo, err := getContainerInfo(services[containerName].ContainerId)
	if err != nil {
		fmt.Printf("Err: %s\n", err)
		return "error"
	}
	return containerInfo.State.Status
}

func updateServiceStatus(key string, status string) int {
	mu.Lock()
	defer mu.Unlock()
	if entry, ok := services[key]; ok {
		entry.Status = status
		services[key] = entry
		return 0
	} else {
		fmt.Println("Service not found/subscribed!")
		return 1
	}

}
