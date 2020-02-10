package vultr

import (
	"context"
	"fmt"
	"io/ioutil"
	"time"

	"github.com/docker/machine/libmachine/drivers"
	"github.com/docker/machine/libmachine/log"
	"github.com/docker/machine/libmachine/mcnflag"
	"github.com/docker/machine/libmachine/ssh"
	"github.com/docker/machine/libmachine/state"
	"github.com/vultr/govultr"
)

const (
	defaultDCID       = 1   // New Jersey
	defaultVPSPlanID  = 201 // 1024 MB RAM,25 GB SSD,1.00 TB BW
	defaultOSID       = 270 // Ubuntu 18.04 x64
	defaultDockerPort = 2376

	osSnapshot       = 164
	osCustom         = 159
	osApplication    = 186
	osBackup         = 180
	osFamilyWindows  = "windows"
	osFamilyIso      = "iso"
	osFamilySnapshot = "snapshot"

	scriptTypeBoot = "boot"
	scriptTypePxe  = "pxe"

	serverStateOk       = "ok"
	powerStatusRunning  = "running"
	serverStatusPending = "pending"
	serverStatusActive  = "active"

	requestPeriod = 10 * time.Second
)

const (
	none = iota
	serverTypeSSD
	serverTypeBareMetal
	serverTypeDedicatedCloud
)

type Driver struct {
	*drivers.BaseDriver
	client *govultr.Client

	APIKey string

	InstanceId   string
	ServerType   int
	DCID         int
	VpsPlanId    int
	OSID         int
	InternalIp   string
	ReservedIPV4 string
	DockerPort   int

	IpxeChainURL string
	ISOID        int
	ScriptId     string
	SnapshotId   string
	AppId        string

	IPV6                    bool
	PrivateNetwork          bool
	AutoBackups             bool
	NotifyActivate          bool
	DDOSProtection          bool
	DDOSProtectionAvailable bool

	Userdata          string
	Label             string
	Hostname          string
	Tag               string
	FirewallGroupId   string
	SSHKeyIdAvailable bool
	SSHKeyId          []string
	NetworkId         []string
}

// GetCreateFlags returns the mcnflag.Flag slice representing the flags
// that can be set, their descriptions and defaults.
func (d *Driver) GetCreateFlags() []mcnflag.Flag {
	return []mcnflag.Flag{
		mcnflag.StringFlag{
			EnvVar: "VULTR_API_KEY",
			Name:   "vultr-api-key",
			Usage:  "Vultr APIKey",
		},
		mcnflag.IntFlag{
			EnvVar: "VULTR_SERVER_TYPE",
			Name:   "vultr-server-type",
			Usage:  "Vultr Server Type ( 1-SSD, 2-BareMetal, 3-DedicatedCloud)",
			Value:  serverTypeSSD,
		},
		mcnflag.IntFlag{
			EnvVar: "VULTR_DC_ID",
			Name:   "vultr-dc-id",
			Usage:  "VPS DCID (Region)",
			Value:  defaultDCID,
		},
		mcnflag.IntFlag{
			EnvVar: "VULTR_VPS_PLAN_ID",
			Name:   "vultr-vps-plan-id",
			Usage:  "VPS Plan ID",
			Value:  defaultVPSPlanID,
		},
		mcnflag.IntFlag{
			EnvVar: "VULTR_OS_ID",
			Name:   "vultr-os-id",
			Usage:  "VPS Operating System ID",
			Value:  defaultOSID,
		},
		mcnflag.StringFlag{
			EnvVar: "VULTR_IPXE_CHAIN_URL",
			Name:   "vultr-ipxe-chain-url",
			Usage:  "If you've selected the 'custom' operating system, this can be set to chainload the specified URL on bootup, via iPXE",
		},
		mcnflag.IntFlag{
			EnvVar: "VULTR_ISO_ID",
			Name:   "vultr-iso-id",
			Usage:  "If you've selected the 'custom' operating system, this is the ID of a specific ISO to mount during the deployment",
		},
		mcnflag.StringFlag{
			EnvVar: "VULTR_SCRIPT_ID",
			Name:   "vultr-script-id",
			Usage:  "If you've not selected a 'custom' operating system, this can be the SCRIPTID of a startup script to execute on boot",
		},
		mcnflag.StringFlag{
			EnvVar: "VULTR_SNAPSHOT_ID",
			Name:   "vultr-snapshot-id",
			Usage:  "If you've selected the 'snapshot' operating system, this should be the SNAPSHOTID (see v1/snapshot/list) to restore for the initial installation",
		},
		mcnflag.StringFlag{
			EnvVar: "VULTR_APP_ID",
			Name:   "vultr-app-id",
			Usage:  "If launching an application (OSID 186), this is the APPID to launch",
		},
		mcnflag.StringFlag{
			EnvVar: "VULTR_RESERVED_IP_V4",
			Name:   "vultr-reserved-ip-v4",
			Usage:  "IP address of the floating IP to use as the main IP of this server",
		},
		mcnflag.BoolFlag{
			EnvVar: "VULTR_IP_V6",
			Name:   "vultr-ip-v6",
			Usage:  "If true, an IPv6 subnet will be assigned to the machine (where available)",
		},
		mcnflag.BoolFlag{
			EnvVar: "VULTR_PRIVATE_NETWORK",
			Name:   "vultr-private-network",
			Usage:  "If true, private networking support will be added to the new server",
		},
		mcnflag.BoolFlag{
			EnvVar: "VULTR_AUTO_BACKUPS",
			Name:   "vultr-auto-backups",
			Usage:  "If true, automatic backups will be enabled for this server (these have an extra charge associated with them)",
		},
		mcnflag.StringSliceFlag{
			EnvVar: "VULTR_NETWORK_ID",
			Name:   "vultr-network-id",
			Usage:  "List of private networks to attach to this server. Use either this field or enable_private_network, not both",
		},
		mcnflag.BoolFlag{
			EnvVar: "VULTR_NOTIFY_ACTIVATE",
			Name:   "vultr-notify-activate",
			Usage:  "If true, an activation email will be sent when the server is ready",
		},
		mcnflag.BoolFlag{
			EnvVar: "VULTR_DDOS_PROTECTION",
			Name:   "vultr-ddos-protection",
			Usage:  "If true, DDOS protection will be enabled on the subscription (there is an additional charge for this)",
		},
		mcnflag.StringFlag{
			EnvVar: "VULTR_USERDATA",
			Name:   "vultr-userdata",
			Usage:  "Base64 encoded user-data",
		},
		mcnflag.StringFlag{
			EnvVar: "VULTR_LABEL",
			Name:   "vultr-label",
			Usage:  "This is a text label that will be shown in the control panel",
		},
		mcnflag.StringFlag{
			EnvVar: "VULTR_HOSTNAME",
			Name:   "vultr-hostname",
			Usage:  "The hostname to assign to this server",
		},
		mcnflag.StringFlag{
			EnvVar: "VULTR_TAG",
			Name:   "vultr-tag",
			Usage:  "The tag to assign to this server",
		},
		mcnflag.StringFlag{
			EnvVar: "VULTR_FIREWALL_GROUP_ID",
			Name:   "vultr-firewall-group-id",
			Usage:  "The firewall group to assign to this server",
		},
		mcnflag.StringSliceFlag{
			EnvVar: "VULTR_SSHKEY_ID",
			Name:   "vultr-sshkey-id",
			Usage:  "List of SSH keys to apply to this server on install (only valid for Linux/FreeBSD)",
		},
		mcnflag.IntFlag{
			EnvVar: "VULTR_DOCKER_PORT",
			Name:   "vultr-docker-port",
			Usage:  "Docker port",
			Value:  defaultDockerPort,
		},
		mcnflag.IntFlag{
			EnvVar: "VULTR_SSH_PORT",
			Name:   "vultr-ssh-port",
			Usage:  "SSH Port",
			Value:  drivers.DefaultSSHPort,
		},
		mcnflag.StringFlag{
			EnvVar: "VULTR_SSH_USER",
			Name:   "vultr-ssh-user",
			Usage:  "SSH User",
			Value:  drivers.DefaultSSHUser,
		},
	}
}

// DriverName returns the name of the driver
func (d *Driver) DriverName() string {
	return "vultr"
}

// NewDriver creates and returns a new instance of the Vultr driver
func NewDriver(hostName, storePath string) *Driver {
	return &Driver{
		DCID:              defaultDCID,
		VpsPlanId:         defaultVPSPlanID,
		OSID:              defaultOSID,
		SSHKeyIdAvailable: false,
		BaseDriver: &drivers.BaseDriver{
			MachineName: hostName,
			StorePath:   storePath,
		},
	}
}

// getClient prepares the Vultr Client
func (d *Driver) getClient() *govultr.Client {
	if d.client == nil {
		d.client = govultr.NewClient(nil, d.APIKey)
		d.client.SetRetryLimit(300)
	}
	return d.client
}

// SetConfigFromFlags configures the driver with the object that was returned
// by RegisterCreateFlags
func (d *Driver) SetConfigFromFlags(flags drivers.DriverOptions) error {
	d.APIKey = flags.String("vultr-api-key")
	d.ServerType = flags.Int("vultr-server-type")
	d.OSID = flags.Int("vultr-os-id")
	d.DCID = flags.Int("vultr-dc-id")
	d.VpsPlanId = flags.Int("vultr-vps-plan-id")
	d.IpxeChainURL = flags.String("vultr-ipxe-chain-url")
	d.ISOID = flags.Int("vultr-iso-id")
	d.ScriptId = flags.String("vultr-script-id")
	d.SnapshotId = flags.String("vultr-snapshot-id")
	d.AppId = flags.String("vultr-app-id")
	d.ReservedIPV4 = flags.String("vultr-reserved-ip-v4")
	d.IPV6 = flags.Bool("vultr-ip-v6")
	d.PrivateNetwork = flags.Bool("vultr-private-network")
	d.AutoBackups = flags.Bool("vultr-auto-backups")
	d.NetworkId = flags.StringSlice("vultr-network-id")
	d.NotifyActivate = flags.Bool("vultr-notify-activate")
	d.DDOSProtection = flags.Bool("vultr-ddos-protection")
	d.Userdata = flags.String("vultr-userdata")
	d.Hostname = flags.String("vultr-hostname")
	d.Tag = flags.String("vultr-tag")
	d.FirewallGroupId = flags.String("vultr-firewall-group-id")
	d.SSHKeyId = flags.StringSlice("vultr-sshkey-id")
	d.SSHPort = flags.Int("vultr-ssh-port")
	d.SSHUser = flags.String("vultr-ssh-user")
	d.DockerPort = flags.Int("vultr-docker-port")

	d.SetSwarmConfigFromFlags(flags)

	if d.APIKey == "" {
		return fmt.Errorf("vultr driver requires the --vultr-api-key option")
	}

	if len(d.Label) == 0 {
		d.Label = d.GetMachineName()
	}

	return nil
}

// GetSSHHostname returns hostname for use with ssh
func (d *Driver) GetSSHHostname() (string, error) {
	return d.GetIP()
}

// GetIP returns an IP or hostname that this host is available at
// e.g. 1.2.3.4 or docker-host-d60b70a14d3a.cloudapp.net
func (d *Driver) GetIP() (string, error) {
	if !d.mainIpIsSet() {
		return "", fmt.Errorf("IP address is not set")
	}

	return d.IPAddress, nil
}

// GetURL returns a Docker compatible host URL for connecting to this host
// e.g. tcp://1.2.3.4:2376
func (d *Driver) GetURL() (string, error) {

	if err := drivers.MustBeRunning(d); err != nil {
		return "", err
	}

	ip, err := d.GetIP()
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("tcp://%s:%d", ip, d.DockerPort), nil
}

// Create a host using the driver's config
func (d *Driver) Create() error {

	var err error

	if len(d.SSHKeyId) == 0 {
		log.Debug("Generating SSH key...")
		key, err := d.createSSHKey()
		if err != nil {
			return err
		}
		d.SSHKeyId = []string{key.SSHKeyID}
	}

	switch d.ServerType {
	case serverTypeBareMetal:
		err = d.createBareMetalServer()
	default:
		err = d.createServer()
	}

	if err != nil {
		return err
	}

	return nil
}

// GetState returns the state that the host is in (running, stopped, etc)
func (d *Driver) GetState() (state.State, error) {

	var (
		srvState state.State
		err      error
	)

	switch d.ServerType {
	case serverTypeBareMetal:
		srvState, err = d.getBareMetalServerState()
	default:
		srvState, err = d.getServerState()
	}

	if err != nil {
		return srvState, err
	}

	return srvState, nil
}

// Stop a host gracefully
func (d *Driver) Stop() error {
	srvState, err := d.GetState()
	if err != nil {
		return err
	}

	if srvState == state.Stopped {
		log.Debug("server is already stopped..")
		return nil
	}

	log.Debugf("stopping %s", d.MachineName)

	switch d.ServerType {
	case serverTypeBareMetal:
		return d.getClient().BareMetalServer.Halt(context.Background(), d.InstanceId)
	default:
		return d.getClient().Server.Halt(context.Background(), d.InstanceId)
	}
}

// Kill stops a host forcefully
func (d *Driver) Kill() error {
	srvState, err := d.GetState()
	if err != nil {
		return err
	}

	if srvState == state.Stopped {
		log.Debug("server is already stopped..")
		return nil
	}

	log.Debugf("stopping %s", d.MachineName)
	switch d.ServerType {
	case serverTypeBareMetal:
		return d.getClient().BareMetalServer.Halt(context.Background(), d.InstanceId)
	default:
		return d.getClient().Server.Halt(context.Background(), d.InstanceId)
	}
}

// Remove a host
func (d *Driver) Remove() error {

	err := d.Stop()
	if err != nil {
		return err
	}

	log.Debugf("removing %s", d.MachineName)
	switch d.ServerType {
	case serverTypeBareMetal:
		return d.getClient().BareMetalServer.Delete(context.Background(), d.InstanceId)
	default:
		return d.getClient().Server.Delete(context.Background(), d.InstanceId)
	}

}

// Restart a host. This may just call Stop(); Start() if the provider does not
// have any special restart behaviour.
func (d *Driver) Restart() error {
	srvState, err := d.GetState()
	if err != nil {
		return err
	}

	if srvState == state.Stopped || srvState == state.Starting {
		log.Debug("server is stopped or starting..")
		return nil
	}

	log.Debugf("restarting %s", d.MachineName)
	switch d.ServerType {
	case serverTypeBareMetal:
		return d.getClient().BareMetalServer.Reboot(context.Background(), d.InstanceId)
	default:
		return d.getClient().Server.Reboot(context.Background(), d.InstanceId)
	}
}

// Start a host
func (d *Driver) Start() error {
	srvState, err := d.GetState()
	if err != nil {
		return err
	}

	if srvState == state.Running || srvState == state.Starting {
		log.Debug("server is already running or starting..")
		return nil
	}

	log.Debugf("starting %s", d.MachineName)
	switch d.ServerType {
	case serverTypeBareMetal:
		return d.getClient().BareMetalServer.Reboot(context.Background(), d.InstanceId)
	default:
		return d.getClient().Server.Start(context.Background(), d.InstanceId)
	}
}

// PreCreateCheck allows for pre-create operations to make sure a driver is ready for creation
func (d *Driver) PreCreateCheck() error {

	if err := d.validateApiCredentials(); err != nil {
		return err
	}

	if err := d.validateOSID(); err != nil {
		return err
	}

	if err := d.validateDCID(); err != nil {
		return err
	}

	if err := d.validatePlan(); err != nil {
		return err
	}

	if d.SnapshotId != "" {
		if err := d.validateSnapshot(); err != nil {
			return err
		}
	}

	if d.ISOID != 0 {
		if err := d.validateISO(); err != nil {
			return err
		}
	}

	if d.AppId != "" {
		if err := d.validateApp(); err != nil {
			return err
		}
	}

	if d.IpxeChainURL != "" && d.OSID != osCustom {
		return fmt.Errorf("iPXEChainURL requires the 'Custom OS' (OSID:%v)", osCustom)
	}

	if d.ScriptId != "" {
		if err := d.validateScript(); err != nil {
			return err
		}
	}

	if d.ReservedIPV4 != "" && !isIPv4(d.ReservedIPV4) {
		return fmt.Errorf("specified --vultr-reserved-ip-v4 parameter is not a valid IPv4 address")
	}

	if d.DDOSProtection && !d.DDOSProtectionAvailable {
		return fmt.Errorf("ddos protection is not available for this region")
	}

	if d.PrivateNetwork && len(d.NetworkId) > 0 {
		return fmt.Errorf("use either --vultr-private-network or --vultr-network-id, not both")
	}

	if len(d.NetworkId) > 0 {
		if err := d.validateNetworkId(); err != nil {
			return err
		}
	}

	if d.FirewallGroupId != "" {
		if err := d.validateFirewallGroupId(); err != nil {
			return err
		}
	}

	if len(d.SSHKeyId) > 0 {
		if err := d.validateSSHKeyId(); err != nil {
			return err
		}
	}

	return nil
}

func (d *Driver) createSSHKey() (*govultr.SSHKey, error) {
	if err := ssh.GenerateSSHKey(d.GetSSHKeyPath()); err != nil {
		return nil, err
	}

	publicKey, err := ioutil.ReadFile(d.publicSSHKeyPath())
	if err != nil {
		return nil, err
	}

	key, err := d.getClient().SSHKey.Create(context.Background(), d.MachineName, string(publicKey))
	if err != nil {
		return key, err
	}

	return key, nil
}

// publicSSHKeyPath is always SSH Key Path appended with ".pub"
func (d *Driver) publicSSHKeyPath() string {
	return d.GetSSHKeyPath() + ".pub"
}

func (d *Driver) GetSSHKeyPath() string {
	if d.SSHKeyPath == "" {
		d.SSHKeyPath = d.ResolveStorePath("id_rsa")
	}

	return d.SSHKeyPath
}
