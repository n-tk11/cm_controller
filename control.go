package main

type service struct {
	container_name string
	image string
	daemon_port string
	status string
}

var services = make(map[string]service)

func container_subscribe(container_name string,image string, daemon_port string){
	new_service := service{container_name, image, daemon_port, "new"}
	services[container_name] = new_service
}