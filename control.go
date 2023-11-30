package main

import (
	"fmt"
	"os"
	"sync"
	"syscall"
)

type Service struct {
	ContainerName string `json:"container_name"`
	ContainerId   string `json:"container_id"`
	Image         string `json:"image"`
	DaemonPort    string `json:"daemon_port"`
	Status        string `json:"status"`
	//running,checkpointed,standby,exited
}

var services = make(map[string]Service)
var mu sync.Mutex

func serviceSubscribe(containerName string, containerId string, image string, daemonPort string) Service {
	newService := Service{containerName, containerId, image, daemonPort, "new"}
	services[containerName] = newService

	createServiceDir(containerName)
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

// TODO: Test
func (s Service) getUpdateServiceStatus() string {
	//fmt.Println("Enter getUpdateServiceStatus")
	if contStat, err := getContainerStatus(s.ContainerName); err == nil {
		//fmt.Println(contStat)
		if contStat == "running" {
			stat := readStatusPipe(s.ContainerName)
			if stat == '0' {
				//fmt.Println("case 0")
				updateServiceStatus(s.ContainerName, "standby")
			} else if stat == '1' {
				//fmt.Println("case 1")
				updateServiceStatus(s.ContainerName, "running")
			} else if stat == '2' {
				updateServiceStatus(s.ContainerName, "checkpointed")
			} else {
				//fmt.Println("case 2")
				fmt.Println("Error reading status from pipe,status will remain the same")
			}
		} else {
			//fmt.Println("case 3")
			updateServiceStatus(s.ContainerName, contStat)
		}
	}
	return s.Status
}

// Get "container" status (as docker status)
func getContainerStatus(containerName string) (string, error) {
	containerInfo, err := getContainerInfo(services[containerName].ContainerId)
	if err != nil {
		fmt.Printf("Err: %s\n", err)
		return "", err
	}
	return containerInfo.State.Status, nil
}

func updateServiceStatus(containerName string, status string) int {
	mu.Lock()
	defer mu.Unlock()
	if entry, ok := services[containerName]; ok {
		entry.Status = status
		services[containerName] = entry
		return 0
	} else {
		fmt.Println("Service not found/subscribed!")
		return 1
	}

}

func readStatusPipe(containerName string) byte {
	pipeName := "services/" + containerName + "/pipes/status"

	// Open the named pipe for reading
	pipe, err := os.OpenFile(pipeName, os.O_RDONLY, os.ModeNamedPipe)
	if err != nil {
		fmt.Println("Error opening named pipe:", err)
		return 0
	}
	defer pipe.Close()

	// Read one byte at a time from the named pipe

	var b [1]byte
	n, err := pipe.Read(b[:])
	if err != nil {
		fmt.Println("Error reading from named pipe:", err)
		return 0
	}

	if n > 0 {
		return b[0]
	}
	return 0
}

func createRootServiceDir() {
	if _, err := os.Stat("services/"); os.IsNotExist(err) {
		if err := os.MkdirAll("services/", os.ModePerm); err != nil {
			fmt.Printf("Error mkdir root service : %v\n", err)
		}
	}
}

// Create per-service dir with pipe dir and files
func createServiceDir(containerName string) {
	filePath := "services/" + containerName
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		if err := os.Mkdir(filePath, os.ModePerm); err != nil {
			fmt.Printf("Error mkdir service: %v\n", err)
		}
	}
	dirPath := "services/" + containerName + "/pipes"
	if _, err := os.Stat(dirPath); os.IsNotExist(err) {
		if err := os.Mkdir(dirPath, os.ModePerm); err != nil {
			fmt.Printf("failed to create pipes dir: %v\n", err)
		}
	}
	pipePath := "services/" + containerName + "/pipes" + "/status"
	if _, err := os.Stat(pipePath); os.IsNotExist(err) {
		originalUmask := syscall.Umask(0)
		file, err := os.Create(pipePath)
		if err != nil {
			fmt.Printf("failed to create named pipe: %v\n", err)
		}
		defer file.Close()
		syscall.Umask(originalUmask)
	}
}
