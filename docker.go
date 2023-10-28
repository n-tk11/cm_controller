package main

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"strconv"

	"github.com/docker/docker/api/types/mount"
)

var last_daemon_port int = 7877

func run_container(containerName string, imageName string, portMapping string, inputEnv []string, mounts []mount.Mount) error {

	host_daemon_port := last_daemon_port + 1
	for isPortInUse(strconv.Itoa(host_daemon_port)) {
		host_daemon_port += 1
		//fmt.Println(host_daemon_port)
	}
	// Define variables to hold the command options based on your logic
	capabilities := []string{"cap_sys_ptrace", "cap_checkpoint_restore"}

	// Create a slice to hold the Docker command and its arguments
	cmdArgs := []string{
		"docker", "run",
		"--name", containerName,
		"-p", portMapping,
	}
	daemon_portMapping := strconv.Itoa(host_daemon_port) + ":7878"
	cmdArgs = append(cmdArgs, "-p", daemon_portMapping)
	// Add capabilities to the command arguments
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

	// Set the output and error pipes to capture the command's output
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Run the Docker CLI command
	err := cmd.Run()
	if err != nil {
		fmt.Printf("Error running Docker command: %v\n", err)
		return err
	}
	fmt.Println("Container started")
	service := container_subscribe(containerName, imageName, strconv.Itoa(host_daemon_port))
	service.Status = "Running"

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
