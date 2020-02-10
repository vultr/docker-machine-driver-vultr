package vultr

import (
	"context"
	"strconv"
	"time"

	"github.com/docker/machine/libmachine/log"
	"github.com/docker/machine/libmachine/state"
	"github.com/vultr/govultr"
)

func (d *Driver) createBareMetalServer() error {

	enableIPv6 := "no"
	notifyActivate := "no"

	if d.IPV6 {
		enableIPv6 = "yes"
	}

	if d.NotifyActivate {
		notifyActivate = "yes"
	}

	serverOptions := &govultr.BareMetalServerOptions{
		SnapshotID:     d.SnapshotId,
		EnableIPV6:     enableIPv6,
		Label:          d.Label,
		SSHKeyIDs:      d.SSHKeyId,
		AppID:          d.AppId,
		UserData:       d.Userdata,
		NotifyActivate: notifyActivate,
		ReservedIPV4:   d.ReservedIPV4,
		Hostname:       d.Hostname,
		Tag:            d.Tag,
	}

	server, err := d.getClient().BareMetalServer.Create(
		context.Background(),
		strconv.Itoa(d.DCID),
		strconv.Itoa(d.VpsPlanId),
		strconv.Itoa(d.OSID),
		serverOptions,
	)

	if err != nil {
		return err
	}

	if server != nil {
		d.InstanceId = server.BareMetalServerID
	}

	if d.bareMetalServerIsReady() {
		log.Infof("installing and booting processes on the bare metal server are complete, it is ready to use")
	}

	return nil
}

func (d *Driver) bareMetalServerIsReady() bool {

	ticker := time.NewTicker(requestPeriod)
	defer ticker.Stop()

	log.Info("waiting for ip address to become available...")

	for _ = range ticker.C {
		serverInfo, err := d.getClient().BareMetalServer.GetServer(context.Background(), d.InstanceId)
		if err != nil {
			continue
		}

		d.IPAddress = serverInfo.MainIP

		if d.mainIpIsSet() {
			break
		}
	}

	return true
}

func (d *Driver) getBareMetalServerState() (state.State, error) {
	server, err := d.getClient().BareMetalServer.GetServer(context.Background(), d.InstanceId)
	if err != nil {
		return state.Error, err
	}

	switch server.Status {
	case serverStatusPending:
		return state.Starting, nil
	case serverStatusActive:
		return state.Running, nil
	default:
		return state.Stopped, nil
	}
}
