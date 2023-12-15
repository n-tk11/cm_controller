package main

import (
	"errors"
	"fmt"
	"os"
	"sync"
	"syscall"

	"go.uber.org/zap"
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

func serviceSubscribe(containerName string, containerId string, image string, daemonPort string) (Service, error) {
	if _, ok := services[containerName]; ok {
		logger.Error("Service already subscribed", zap.String("containerName", containerName))
		return services[containerName], errors.New("Service already subscribed")
	}
	err := createServiceDir(containerName)
	if err != nil {
		logger.Error("Error creating service dir", zap.String("containerName", containerName), zap.Error(err))
		return Service{}, err
	}
	newService := Service{containerName, containerId, image, daemonPort, "new"}
	services[containerName] = newService

	return newService, nil
}

func serviceUnsubscribe(containerName string) error {
	if _, ok := services[containerName]; ok {
		delete(services, containerName)
	} else {
		logger.Error("Service not found", zap.String("containerName", containerName))
		return fmt.Errorf("No container name %s", containerName)
	}
	return nil
}

// TODO: This function will first check if all subscribed containers is still running(ffdaemon is alive) if not it will update the status to "stopped" / report all subscribed containers (including its status) in json
func getAllServices() {

}

func (s Service) getUpdateServiceStatus() string {
	//fmt.Println("Enter getUpdateServiceStatus")
	logger.Info("Getting service status", zap.String("containerName", s.ContainerName))
	if contStat, err := getContainerStatus(s.ContainerName); err == nil {
		//fmt.Println(contStat)
		if contStat == "running" {
			stat, err := readStatusFile(s.ContainerName)
			if err != nil {
				logger.Error("Error reading status from status file", zap.String("containerName", s.ContainerName), zap.Error(err))
				return s.Status
			}
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
				logger.Error("Error reading status from status file", zap.String("containerName", s.ContainerName), zap.Error(err))
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
	logger.Info("Getting container status", zap.String("containerName", containerName))
	containerInfo, err := getContainerInfo(services[containerName].ContainerId)
	if err != nil {
		fmt.Printf("Err: %s\n", err)
		return "", err
	}
	return containerInfo.State.Status, nil
}

func updateServiceStatus(containerName string, status string) error {
	mu.Lock()
	defer mu.Unlock()
	if entry, ok := services[containerName]; ok {
		entry.Status = status
		services[containerName] = entry
		return nil
	} else {
		fmt.Println("Service not found/subscribed!")
		return errors.New("Service not found")
	}

}

func readStatusFile(containerName string) (byte, error) {
	logger.Info("Reading status file", zap.String("containerName", containerName))
	fileName := "services/" + containerName + "/comms/status"

	// Open the named pipe for reading
	pipe, err := os.OpenFile(fileName, os.O_RDONLY, os.ModeNamedPipe)
	if err != nil {
		fmt.Println("Error opening status file:", err)
		return 0, err
	}
	defer pipe.Close()

	// Read one byte at a time from the named pipe

	var b [1]byte
	n, err := pipe.Read(b[:])
	if err != nil {
		fmt.Println("Error reading from status file:", err)
		return 0, err
	}

	if n > 0 {
		return b[0], nil
	}
	return 0, fmt.Errorf("No byte read")
}

func createRootServiceDir() {
	if _, err := os.Stat("services/"); os.IsNotExist(err) {
		if err := os.MkdirAll("services/", os.ModePerm); err != nil {
			logger.Panic("Error mkdir root service", zap.Error(err))
			panic(err)
		}
	}
}

// Create per-service dir with pipe dir and files
func createServiceDir(containerName string) error {
	filePath := "services/" + containerName
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		if err := os.Mkdir(filePath, os.ModePerm); err != nil {
			logger.Error("Error mkdir service", zap.String("containerName", containerName), zap.Error(err))
			return err
		}
	}
	dirPath := "services/" + containerName + "/comms"
	if _, err := os.Stat(dirPath); os.IsNotExist(err) {
		if err := os.Mkdir(dirPath, os.ModePerm); err != nil {
			logger.Error("Error mkdir comms", zap.String("containerName", containerName), zap.Error(err))
			return err
		}
	}
	statusPath := "services/" + containerName + "/comms" + "/status"
	if _, err := os.Stat(statusPath); os.IsNotExist(err) {
		originalUmask := syscall.Umask(0)
		file, err := os.Create(statusPath)
		if err != nil {
			logger.Error("Error creating status file", zap.String("containerName", containerName), zap.Error(err))
			return err
		}
		defer file.Close()
		syscall.Umask(originalUmask)
	}
	return nil
}

func isSubscribed(name string) bool {
	if _, ok := services[name]; ok {
		return true
	}
	return false
}
