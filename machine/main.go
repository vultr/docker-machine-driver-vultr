package main

import (
	"github.com/docker/machine/libmachine/drivers/plugin"
	"github.com/vultr/docker-machine-driver-vultr/machine/driver"
)

func main() {
	plugin.RegisterDriver(driver.NewDriver("", ""))
}
