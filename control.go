package main

import "fmt"

type Service struct {
	ContainerName string
	Image         string
	DaemonPort    string
	Status        string
}

var services = make(map[string]Service)

func containerSubscribe(containerName string, image string, daemonPort string) Service {
	newService := Service{containerName, image, daemonPort, "new"}
	services[containerName] = newService
	return newService
}

func containerUnsubscribe(containerName string, image string) int {
	if _, ok := services[containerName]; ok {
		delete(services, containerName)
	} else {
		fmt.Printf("No container name %s.\n", containerName)
		return 1
	}

	return 0
}
