package tests

import (
	"fmt"
	"os/exec"
)

type MachineConfig struct {
	VultrAPIKey string
	MachineName string
}

// CreateMachine ... test new machine creation
func (c MachineConfig) CreateMachine() ([]byte, error) {
	args := []string{
		"create",
		"-d",
		"vultr",
		"--vultr-api-key",
		c.VultrAPIKey,
		"--engine-install-url",
		"https://releases.rancher.com/install-docker/19.03.9.sh",
	}

	args = append(args, c.MachineName)

	cmd := exec.Command("docker-machine", args...)

	fmt.Println(cmd.Args)

	return cmd.CombinedOutput()
}

// DeleteMachine ... test delete machine creation
func (c MachineConfig) DeleteMachine() ([]byte, error) {
	args := []string{
		"rm",
		c.MachineName,
	}
	cmd := exec.Command("docker-machine", args...)
	return cmd.CombinedOutput()
}

// StartMachine ... test start machine
func (c MachineConfig) StartMachine() ([]byte, error) {
	args := []string{
		"start",
		c.MachineName,
	}
	cmd := exec.Command("docker-machine", args...)
	return cmd.CombinedOutput()
}

// StopMachine ... test stop machine
func (c MachineConfig) StopMachine() ([]byte, error) {
	args := []string{
		"stop",
		c.MachineName,
	}
	cmd := exec.Command("docker-machine", args...)
	return cmd.CombinedOutput()
}
