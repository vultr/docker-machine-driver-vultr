package vultr

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/vultr/govultr"
)

func (d *Driver) validateApiCredentials() error {
	_, err := d.getClient().Account.GetInfo(context.Background())
	if err != nil {
		return err
	}

	return nil
}

func (d *Driver) validatePlan() error {

	var (
		plans []int
		err   error
	)

	switch d.ServerType {
	case serverTypeBareMetal:
		plans, err = d.getClient().Region.BareMetalAvailability(context.Background(), d.DCID)
	case serverTypeDedicatedCloud:
		plans, err = d.getClient().Region.Vdc2Availability(context.Background(), d.DCID)
	default:
		plans, err = d.getClient().Region.Availability(context.Background(), d.DCID, "all")
	}

	if err != nil {
		return err
	}

	for i := range plans {
		if d.VpsPlanId == plans[i] {
			return nil
		}
	}

	return fmt.Errorf("planId %d not available in the chosen region. Available plans for RegionId %d: %v", d.VpsPlanId, d.DCID, plans)
}

func (d *Driver) validateOSID() error {
	opSystems, err := d.getClient().OS.List(context.Background())
	if err != nil {
		return err
	}

	for i := range opSystems {
		if d.OSID == opSystems[i].OsID {
			if isSSHKeyAvailable(opSystems[i].Family) {
				d.SSHKeyIdAvailable = true
			}
			return nil
		}
	}

	return fmt.Errorf("operating system with Id - %d is not available. Available Operating Systems: %v", d.OSID, opSystems)
}

func (d *Driver) validateDCID() error {

	regions, err := d.getClient().Region.List(context.Background())
	if err != nil {
		return err
	}

	curRegion := strconv.Itoa(d.DCID)

	for i := range regions {
		if curRegion == regions[i].RegionID {
			d.DDOSProtectionAvailable = regions[i].Ddos
			return nil
		}
	}

	return fmt.Errorf("region with Id - %s is not available. Available regions: %v", curRegion, regions)
}

func (d *Driver) validateSnapshot() error {
	if d.OSID != osSnapshot {
		return fmt.Errorf("snapshot is only available with Snapshot OS type (OSID:%v)", osSnapshot)
	}

	snapshots, err := d.getClient().Snapshot.List(context.Background())
	if err != nil {
		return err
	}

	for i := range snapshots {
		if d.SnapshotId == snapshots[i].SnapshotID {
			return nil
		}
	}

	return fmt.Errorf("snapshot with Id - %s is not exist. Existing snapshots: %v", d.SnapshotId, snapshots)
}

func (d *Driver) validateISO() error {
	if d.OSID != osCustom {
		return fmt.Errorf("server creation using ISO is only available with Custom OS type (OSID:%v)", osCustom)
	}

	iso, err := d.getClient().ISO.List(context.Background())
	if err != nil {
		return err
	}

	for i := range iso {
		if d.ISOID == iso[i].ISOID {
			return nil
		}
	}

	return fmt.Errorf("ISO with Id - %d is not exist. Existing ISO: %v", d.ISOID, iso)
}

func (d *Driver) validateApp() error {

	if d.OSID != osApplication {
		return fmt.Errorf("application is only available with Application OS type (OSID:%v)", osApplication)
	}

	apps, err := d.getClient().Application.List(context.Background())
	if err != nil {
		return err
	}

	for i := range apps {
		if d.AppId == apps[i].AppID {
			return nil
		}
	}

	return fmt.Errorf("application with Id - %s is not exist. Existing applications: %v", d.AppId, apps)
}

func (d *Driver) validateScript() error {

	var (
		exists bool
		script *govultr.StartupScript
	)

	scripts, err := d.getClient().StartupScript.List(context.Background())
	if err != nil {
		return err
	}

	for i := range scripts {
		if d.ScriptId == scripts[i].ScriptID {
			exists = true
			script = &scripts[i]
		}
	}

	if !exists {
		return fmt.Errorf("startup script with Id - %s is not exist. Existing scripts: %v", d.ScriptId, scripts)
	}

	if script != nil {
		if script.Type == scriptTypePxe && (d.OSID != osCustom || d.ServerType == serverTypeBareMetal) {
			return fmt.Errorf("PXE script only available if there is no operating system installed to the server's disk (OSID:%v), not available on bare metal server", osCustom)
		}

		if script.Type == scriptTypeBoot && (d.OSID == osCustom || d.OSID == osSnapshot || d.OSID == osBackup) {
			return fmt.Errorf("boot script type is only available on server with installed OS")
		}
	}

	return nil
}

func (d *Driver) validateNetworkId() error {

	availableNetworks, err := d.getClient().Network.List(context.Background())
	if err != nil {
		return err
	}

	var networks []string
	for i := range availableNetworks {
		networks = append(networks, availableNetworks[i].NetworkID)
	}

	available, notAvailable := inspect(d.NetworkId, networks)

	if len(notAvailable) == 0 {
		return nil
	}

	return fmt.Errorf("some of networkIds that you specified is not available - %s; available networks - %s", strings.Join(notAvailable, ","), strings.Join(available, ","))
}

func (d *Driver) validateFirewallGroupId() error {

	groupList, err := d.getClient().FirewallGroup.List(context.Background())
	if err != nil {
		return err
	}

	for i := range groupList {
		if d.FirewallGroupId == groupList[i].FirewallGroupID {
			return nil
		}
	}

	return fmt.Errorf("firewall group ID - %s that you specified is not exist, existing groups - %v", d.FirewallGroupId, groupList)
}

func (d *Driver) validateSSHKeyId() error {

	if !d.SSHKeyIdAvailable {
		return fmt.Errorf("SSHKEYID is not valid for this OS (only valid for Linux/FreeBSD)")
	}

	availableKeys, err := d.getClient().SSHKey.List(context.Background())
	if err != nil {
		return err
	}

	var sshKeys []string

	for i := range availableKeys {
		sshKeys = append(sshKeys, availableKeys[i].SSHKeyID)
	}

	available, notAvailable := inspect(d.SSHKeyId, sshKeys)

	if len(notAvailable) == 0 {
		return nil
	}

	return fmt.Errorf("some of SSH Key Id that you specified is not available - %s; available SSH Key Id - %s", strings.Join(notAvailable, ","), strings.Join(available, ","))
}

func (d *Driver) mainIpIsSet() bool {
	if d.IPAddress != "" && d.IPAddress != "0" && d.IPAddress != "0.0.0.0" {
		return true
	}
	return false
}

func inspect(specified, existing []string) (available []string, notAvailable []string) {

	notAvailable = specified

	if len(existing) == 0 {
		return
	}

	for i := range specified {
		for j := range existing {
			if specified[i] == existing[j] {
				available = append(available, specified[i])
				notAvailable = remove(notAvailable, specified[i])
				break
			}
		}
	}

	return
}

func isSSHKeyAvailable(osFamily string) bool {
	if osFamily == osFamilyIso || osFamily == osFamilySnapshot || osFamily == osFamilyWindows || osFamily == "" {
		return false
	}
	return true
}

func isIPv4(host string) bool {

	parts := strings.Split(host, ".")

	if len(parts) < 4 {
		return false
	}

	for i := range parts {
		if i, err := strconv.Atoi(parts[i]); err == nil {
			if i < 0 || i > 255 {
				return false
			}
		} else {
			return false
		}
	}

	return true
}

func remove(s []string, item string) []string {

	if len(s) == 0 {
		return s
	}

	var newItems []string

	for _, el := range s {
		if el != item {
			newItems = append(newItems, el)
		}
	}

	return newItems
}
