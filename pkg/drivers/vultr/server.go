package vultr

import (
	"context"
	"time"

	"github.com/docker/machine/libmachine/log"
	"github.com/docker/machine/libmachine/state"
	"github.com/vultr/govultr"
)

func (d *Driver) createServer() error {
	serverOptions := &govultr.ServerOptions{
		IPXEChain:            d.IpxeChainURL,
		IsoID:                d.ISOID,
		SnapshotID:           d.SnapshotId,
		ScriptID:             d.ScriptId,
		EnableIPV6:           d.IPV6,
		EnablePrivateNetwork: d.PrivateNetwork,
		NetworkID:            d.NetworkId,
		Label:                d.Label,
		SSHKeyIDs:            d.SSHKeyId,
		AutoBackups:          d.AutoBackups,
		AppID:                d.AppId,
		UserData:             d.Userdata,
		NotifyActivate:       d.NotifyActivate,
		DDOSProtection:       d.DDOSProtection,
		ReservedIPV4:         d.ReservedIPV4,
		Hostname:             d.Hostname,
		Tag:                  d.Tag,
		FirewallGroupID:      d.FirewallGroupId,
	}

	server, err := d.getClient().Server.Create(
		context.Background(),
		d.DCID,
		d.VpsPlanId,
		d.OSID,
		serverOptions,
	)

	if err != nil {
		return err
	}

	if server != nil {
		d.InstanceId = server.InstanceID
	}

	if d.serverIsReady() {
		log.Infof("installing and booting processes on the server are complete, server is ready to use")
	}

	return nil
}

func (d *Driver) serverIsReady() bool {

	ticker := time.NewTicker(requestPeriod)
	defer ticker.Stop()

	log.Info("waiting for ip address to become available...")

	for _ = range ticker.C {
		serverInfo, err := d.getClient().Server.GetServer(context.Background(), d.InstanceId)
		if err != nil {
			continue
		}

		d.IPAddress = serverInfo.MainIP
		d.InternalIp = serverInfo.InternalIP

		if d.mainIpIsSet() {
			break
		}
	}

	log.Infof("Created Vultr VPS with ID: %s, Public IP: %s",
		d.InstanceId,
		d.IPAddress,
	)

	log.Info("waiting for server state to become 'ok'...")

	for _ = range ticker.C {
		serverInfo, err := d.getClient().Server.GetServer(context.Background(), d.InstanceId)
		if err != nil {
			continue
		}

		serverStatus, err := d.GetState()
		if err != nil {
			continue
		}

		if serverStatus == state.Running && serverInfo.ServerState == serverStateOk {
			break
		}
	}

	return true
}

func (d *Driver) getServerState() (state.State, error) {
	server, err := d.getClient().Server.GetServer(context.Background(), d.InstanceId)
	if err != nil {
		return state.Error, err
	}

	switch server.Status {
	case serverStatusPending:
		return state.Starting, nil
	case serverStatusActive:
		switch server.PowerStatus {
		case powerStatusRunning:
			switch server.ServerState {
			case serverStateOk:
				return state.Running, nil
			default:
				return state.Starting, nil
			}
		default:
			return state.Stopped, nil
		}
	}
	return state.None, nil
}
