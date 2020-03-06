module github.com/vultr/docker-machine-driver-vultr

go 1.13

replace github.com/docker/docker => github.com/docker/engine v1.4.2-0.20190717161051-705d9623b7c1

require (
	github.com/Azure/go-ansiterm v0.0.0-20170929234023-d6e3b3328b78 // indirect
	github.com/docker/docker v1.13.1 // indirect
	github.com/docker/machine v0.16.2
	github.com/google/go-cmp v0.4.0 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/sirupsen/logrus v1.4.2 // indirect
	github.com/stretchr/testify v1.4.0
	github.com/vultr/govultr v0.3.0
	golang.org/x/crypto v0.0.0-20200206161412-a0c6ece9d31a // indirect
	gotest.tools v2.2.0+incompatible // indirect
)
