package main

import (
	"strconv"
	"log"
	"context"
	"fmt"
	"github.com/docker/docker/client"
	natting "github.com/docker/go-connections/nat"
	"github.com/docker/docker/api/types/container"
	network "github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types"
	"github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/docker/docker/api/types/mount"
	"net"
)

var last_daemon_port int = 7877 

func run_container(containername string,imagename string, port string, inputEnv []string, mounts []mount.Mount) error {
	client, err := client.NewEnvClient()
	if err != nil {
		log.Fatalf("Unable to create docker client")
	}

	// Define a PORT opening
	newport, err := natting.NewPort("tcp", port)
	if err != nil {
		fmt.Println("Unable to create docker port")
		return err
	}

	new_daemon_port, err := natting.NewPort("tcp", "7878")
	if err != nil {
		fmt.Println("Unable to create docker port")
		return err
	}
	host_daemon_port := last_daemon_port+1
	for isPortInUse(strconv.Itoa(host_daemon_port)) {
		host_daemon_port += 1
		//fmt.Println(host_daemon_port)
	}
	// Configured hostConfig: 
	// https://godoc.org/github.com/docker/docker/api/types/container#HostConfig
	hostConfig := &container.HostConfig{
		PortBindings: natting.PortMap{
			newport: []natting.PortBinding{
				{
					HostIP:   "0.0.0.0",
					HostPort: port,
				},
			},
			new_daemon_port: []natting.PortBinding{
				{
					HostIP: "0.0.0.0",
					HostPort: strconv.Itoa(host_daemon_port),
				},
			},
		},
		RestartPolicy: container.RestartPolicy{
			Name: "always",
		},
		LogConfig: container.LogConfig{
			Type:   "json-file",
			Config: map[string]string{},
		},
		CapAdd: []string{
			"cap_sys_ptrace", 
			"cap_checkpoint_restore"},
		SecurityOpt: []string{
			"apparmor=unconfined",
			"systempaths=unconfined",
		},
		Mounts: mounts,
		/*
		[]mount.Mount{
            {
                Type:   mount.TypeBind,
                Source: "/mnt/checkpointfs",
                Target: "/checkpointfs",
            },
						{
							  Type: mount.TypeBind,
								Source: "/tmp",
								Target: "/tmp",
						},
    },*/
	}

	// Define Network config (why isn't PORT in here...?:
	// https://godoc.org/github.com/docker/docker/api/types/network#NetworkingConfig
	networkConfig := &network.NetworkingConfig{
		EndpointsConfig: map[string]*network.EndpointSettings{},
	}
	gatewayConfig := &network.EndpointSettings{
		Gateway: "gatewayname",
	}
	networkConfig.EndpointsConfig["bridge"] = gatewayConfig

	// Define ports to be exposed (has to be same as hostconfig.portbindings.newport)
	exposedPorts := map[natting.Port]struct{}{
		newport: struct{}{},
		new_daemon_port: struct{}{},
	}

	// Configuration 
	// https://godoc.org/github.com/docker/docker/api/types/container#Config
	config := &container.Config{
		Cmd: []string{"ff_daemon"},
		Image:        imagename,
		Env: 		  inputEnv,
		ExposedPorts: exposedPorts,
		Hostname:     fmt.Sprintf("%s", imagename),
		AttachStdin: true,
		//AttachStdout: true,
	}

	platform := &v1.Platform{
		Architecture: "amd64",
		OS: "linux",
	}

	// Creating the actual container. This is "nil,nil,nil" in every example.
	cont, err := client.ContainerCreate(
		context.Background(),
		config,
		hostConfig,
		networkConfig,
		platform,
		containername,
	)

	if err != nil {
		log.Println(err)
		return err
	}

	// Run the actual container 
	client.ContainerStart(context.Background(), cont.ID, types.ContainerStartOptions{})
	log.Printf("Container %s is created", cont.ID)

	service := container_subscribe(containername, imagename,  strconv.Itoa(host_daemon_port))
	service.Status = "running"
	last_daemon_port = host_daemon_port

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
