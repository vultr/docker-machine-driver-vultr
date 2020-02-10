package main

import (
	"github.com/docker/machine/libmachine/drivers/plugin"
	"github.com/vultr/docker-machine-driver-vultr/pkg/drivers/vultr"
)

func main() {
	plugin.RegisterDriver(vultr.NewDriver("", ""))
}
