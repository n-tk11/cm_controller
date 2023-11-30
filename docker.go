package main

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/client"
)

var lastDaemonPort int = 7877

func runContainer(containerName string, imageName string, portMapping string, inputEnv []string, mounts []mount.Mount, caps []string) error {

	hostDaemonPort := lastDaemonPort + 1
	for isPortInUse(strconv.Itoa(hostDaemonPort)) {
		hostDaemonPort += 1
		//fmt.Println(host_daemon_port)
	}

	// Create a slice to hold the Docker command and its arguments
	cmdArgs := []string{
		"docker", "run",
		"--name", containerName,
		"-p", portMapping,
	}
	daemonPortMapping := strconv.Itoa(hostDaemonPort) + ":7878"
	cmdArgs = append(cmdArgs, "-p", daemonPortMapping)
	// Add capabilities to the command arguments
	capabilities := append([]string{"cap_sys_ptrace", "cap_checkpoint_restore"}, caps...)

	for _, cap := range capabilities {
		cmdArgs = append(cmdArgs, "--cap-add", cap)
	}

	//--security-opt systempaths=unconfined --security-opt apparmor=unconfined
	securityOpts := []string{"systempaths=unconfined", "apparmor=unconfined"}
	for _, secOpt := range securityOpts {
		cmdArgs = append(cmdArgs, "--security-opt", secOpt)
	}
	//Add mount arguments
	for _, mount := range mounts {
		mountArg := "type=" + string(mount.Type) + ",source=" + mount.Source + ",target=" + mount.Target
		cmdArgs = append(cmdArgs, "--mount", mountArg)
	}

	//Add env arguments
	for _, env := range inputEnv {
		cmdArgs = append(cmdArgs, "-e", env)
	}
	cmdArgs = append(cmdArgs, "-d", "--init")

	// Add the image name to the command arguments and command to start with
	cmdArgs = append(cmdArgs, imageName, "ff_daemon")

	// Create the command using the constructed arguments
	cmd := exec.Command(cmdArgs[0], cmdArgs[1:]...)

	var stdoutBuf bytes.Buffer
	// Set the output and error pipes to capture the command's output
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = os.Stderr

	// Run the Docker CLI command
	err := cmd.Run()
	if err != nil {
		fmt.Printf("Error running Docker command: %v\n", err)
		return err
	}
	containerId := strings.TrimSuffix(stdoutBuf.String(), "\n")
	fmt.Printf("ContaierID: %s\n", containerId)
	fmt.Println("Container started")
	service := serviceSubscribe(containerName, containerId, imageName, strconv.Itoa(hostDaemonPort))
	service.getUpdateServiceStatus()
	return nil
}

func isPortInUse(port string) bool {
	listener, err := net.Listen("tcp", ":"+port)
	if err != nil {
		// Port is in use
		return true
	}
	defer listener.Close()

	// Port is available
	return false
}

// TODO: Test status update feature

func getAllContainerInfo() {
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		panic(err)
	}
	defer cli.Close()

	containers, err := cli.ContainerList(ctx, types.ContainerListOptions{})
	if err != nil {
		panic(err)
	}

	for _, container := range containers {
		fmt.Printf("Container ID: %s\n", container.ID)
		fmt.Printf("Container Name: %s\n", container.Names[0])
		fmt.Printf("Container Status: %s\n", container.Status)
	}
}

func getContainerInfo(containerId string) (types.ContainerJSON, error) {

	// Create a Docker client
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		fmt.Println("Create Docker Client Error")
		return types.ContainerJSON{}, err
	}

	ctx := context.Background()

	//fmt.Printf("Will call Inspect for %s\n", containerId)
	// Inspect the container to get detailed information
	containerInfo, err := cli.ContainerInspect(ctx, containerId)
	if err != nil {
		fmt.Printf("Call ContainerInspect for %s error\n", containerId)
		return types.ContainerJSON{}, err
	}

	// Print container information
	fmt.Printf("Container ID: %s\n", containerInfo.ID)
	fmt.Printf("Container Name: %s\n", containerInfo.Name)
	fmt.Printf("Container Status: %s\n", containerInfo.State.Status)

	return containerInfo, nil
}
