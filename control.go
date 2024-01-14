package main

import (
	"errors"
	"fmt"
	"io"
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
	err_p := writeServicePort(containerName, daemonPort)
	if err_p != nil {
		logger.Error("Error writing service port", zap.String("containerName", containerName), zap.Error(err_p))
		return Service{}, err_p
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
	logger.Debug("Getting service status", zap.String("containerName", s.ContainerName))
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
	logger.Debug("Getting container status", zap.String("containerName", containerName))
	containerInfo, err := getContainerInfo(containerName)
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
	logger.Debug("Reading status file", zap.String("containerName", containerName))
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

func writeServicePort(containerName string, port string) error {
	filePath := "services/" + containerName + "/port"
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		originalUmask := syscall.Umask(0)
		file, err := os.Create(filePath)
		if err != nil {
			logger.Error("Error creating port file", zap.String("containerName", containerName), zap.Error(err))
			return err
		}
		defer file.Close()
		syscall.Umask(originalUmask)
	}
	file, err := os.OpenFile(filePath, os.O_WRONLY, os.ModeNamedPipe)
	if err != nil {
		logger.Error("Error opening port file", zap.String("containerName", containerName), zap.Error(err))
		return err
	}
	defer file.Close()

	_, err = file.WriteString(port)
	if err != nil {
		logger.Error("Error writing port file", zap.String("containerName", containerName), zap.Error(err))
		return err
	}
	return nil
}

func isSubscribed(name string) bool {
	if _, ok := services[name]; ok {
		return true
	}
	return false
}

// It will scan the /services directory and check all services stuatus via getUpdateServiceStatus()
func checkServices() {
	logger.Debug("Checking services")
	dirPath := "services/"
	dirEntries, err := os.ReadDir(dirPath)
	if err != nil {
		logger.Error("Error reading services dir", zap.Error(err))
		return
	}

	for _, dirEntry := range dirEntries {
		if dirEntry.IsDir() {
			conInfo, err := getContainerInfo(dirEntry.Name())
			if err != nil {
				logger.Error("Error getting container info", zap.String("containerName", dirEntry.Name()), zap.Error(err))
				continue
			}
			portFilePath := fmt.Sprintf("services/%s/port", dirEntry.Name())
			port, err := readServicePort(portFilePath)
			if err != nil {
				logger.Error("Error reading port file", zap.String("containerName", dirEntry.Name()), zap.Error(err))
				continue
			}
			service := Service{dirEntry.Name(), conInfo.ID, conInfo.Config.Image, port, "new"}
			services[dirEntry.Name()] = service
			service.getUpdateServiceStatus()
			logger.Debug("Added a former service", zap.String("containerName", dirEntry.Name()))
		}
	}

}

func readServicePort(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	portBytes, err := io.ReadAll(file)
	if err != nil {
		return "", err
	}

	return string(portBytes), nil
}
