package main

type Service struct {
	Container_name string
	Image string
	Daemon_port string
	Status string
}

var services = make(map[string]Service)

func container_subscribe(container_name string,image string, daemon_port string) Service{
	new_service := Service{container_name, image, daemon_port, "new"}
	services[container_name] = new_service
	return new_service
}