package driver

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/docker/machine/libmachine/drivers"
	"github.com/docker/machine/libmachine/mcnflag"
	"github.com/docker/machine/libmachine/state"
	log "github.com/sirupsen/logrus"
	"github.com/vultr/docker-machine-driver-vultr/utils"
	"github.com/vultr/govultr/v2"
)

const (
	defaultOSID   = 387 // Ubuntu 20.04
	defaultRegion = "ewr"
	defaultPlan   = "vc2-1c-2gb"
)

// Driver ... driver struct
type Driver struct {
	*drivers.BaseDriver
	RequestPayloads struct {
		InstanceCreateReq govultr.InstanceCreateReq
	}
	ResponsePayloads struct {
		Instance *govultr.Instance
	}
	APIKey     string
	InstanceID string
}

// GetCreateFlags ... returns the mcnflag.Flag slice representing the flags
// that can be set, their descriptions and defaults.
func (d *Driver) GetCreateFlags() []mcnflag.Flag {
	return []mcnflag.Flag{
		mcnflag.StringFlag{
			EnvVar: "VULTR_API_KEY",
			Name:   "vultr-api-key",
			Usage:  "Vultr API Key",
		},
		mcnflag.StringFlag{
			EnvVar: "VULTR_REGION",
			Name:   "vultr-region",
			Usage:  "Region of where resource will be deployed (default: [ewr] New Jersey)",
			Value:  defaultRegion,
		},
		mcnflag.StringFlag{
			EnvVar: "VULTR_VPS_PLAN",
			Name:   "vultr-vps-plan",
			Usage:  "VPS Plan (default: [vc2-1c-2gb] 1 vCPU, 2GB RAM, 55GB SSD)",
			Value:  defaultPlan,
		},
		mcnflag.StringSliceFlag{
			EnvVar: "VULTR_TAGS",
			Name:   "vultr-tags",
			Usage:  "Tags you'd like to attach to this resource",
			Value:  []string{},
		},
		mcnflag.IntFlag{
			EnvVar: "VULTR_OSID",
			Name:   "vultr-os-id",
			Usage:  "Operating system ID (default: [387] Ubuntu 20.04)",
			Value:  defaultOSID,
		},
		mcnflag.StringFlag{
			EnvVar: "VULTR_ISOID",
			Name:   "vultr-iso-id",
			Usage:  "ISO ID you'd like to boot this resource into",
		},
		mcnflag.IntFlag{
			EnvVar: "VULTR_APPID",
			Name:   "vultr-app-id",
			Usage:  "App ID of the Vultr Marketplace App you'd like to deploy to this resource",
		},
		mcnflag.StringFlag{
			EnvVar: "VULTR_IMAGEID",
			Name:   "vultr-image-id",
			Usage:  "Specific Image ID of the Vultr Marketplace App you'd like to deploy to this resource",
		},
		mcnflag.StringFlag{
			EnvVar: "VULTR_FIREWALL_GROUP_ID",
			Name:   "vultr-firewall-group-id",
			Usage:  "Firewall Group ID you'd like to attach this resource to",
		},
		mcnflag.StringFlag{
			EnvVar: "VULTR_IPXE_CHAIN_URL",
			Name:   "vultr-ipxe-chain-url",
			Usage:  "IPXE Chain URL you'd like to boot this resource to",
		},
		mcnflag.StringFlag{
			EnvVar: "VULTR_STARTUP_SCRIPT_ID",
			Name:   "vultr-startup-script-id",
			Usage:  "Startup Script ID you'd like to run on this resource after boot",
		},
		mcnflag.StringFlag{
			EnvVar: "VULTR_SNAPSHOT_ID",
			Name:   "vultr-snapshot-id",
			Usage:  "Snapshot ID you'd like to deploy to this resource",
		},
		mcnflag.BoolFlag{
			EnvVar: "VULTR_ENABLE_IPV6",
			Name:   "vultr-enabled-ipv6",
			Usage:  "Enable IPV6 on this resource (default: false)",
		},
		mcnflag.BoolFlag{
			EnvVar: "VULTR_ENABLE_VPC",
			Name:   "vultr-enable-vpc",
			Usage:  "Enable VPC on this resource (default: false)",
		},
		mcnflag.StringSliceFlag{
			EnvVar: "VULTR_VPC_IDS",
			Name:   "vultr-vpc-ids",
			Usage:  "VPC IDs you want to attach to this resource",
			Value:  []string{},
		},
		mcnflag.StringSliceFlag{
			EnvVar: "VULTR_SSH_KEY_IDS",
			Name:   "vultr-ssh-key-ids",
			Usage:  "SSH Key IDs you'd like installed on this resource. If no SSH Key ID is provided, one will be generated for you",
			Value:  []string{},
		},
		mcnflag.BoolFlag{
			EnvVar: "VULTR_VPS_BACKUPS",
			Name:   "vultr-vps-backups",
			Usage:  "Enable automatic backups of this VPS (default: disabled)",
		},
		mcnflag.BoolFlag{
			EnvVar: "VULTR_DDOS_PROTECTION",
			Name:   "vultr-ddos-protection",
			Usage:  "Enable DDOS Protection on this resource (default: false)",
		},
		mcnflag.StringFlag{
			EnvVar: "VULTR_CLOUD_INIT_USER_DATA",
			Name:   "vultr-cloud-init-user-data",
			Usage:  "Pass base64 encoded cloud-init user data to this resource to execute after successful provision",
		},
		mcnflag.StringFlag{
			EnvVar: "VULTR_FLOATING_IPV4_ID",
			Name:   "vultr-floating-ipv4-id",
			Usage:  "ID of the floating/reserved IPV4 address to use as the main IP of this resource",
		},
		mcnflag.BoolFlag{
			EnvVar: "VULTR_SEND_ACTIVATION_EMAIL",
			Name:   "vultr-send-activation-email",
			Usage:  "Send activation email when your server begins deployment (default: false)",
		},
	}
}

// SetConfigFromFlags ... configures the driver with the object that was returned by RegisterCreateFlags
func (d *Driver) SetConfigFromFlags(opts drivers.DriverOptions) error {
	d.APIKey = opts.String("vultr-api-key")
	if len(d.APIKey) == 0 {
		return fmt.Errorf("vultr-api-key cannot be empty")
	}

	machineName := d.GetMachineName()
	// check if MachineName is set
	if d.BaseDriver.MachineName == "" {
		return fmt.Errorf("machine name is not set")
	}

	// ** Set Hostname and Label ** //
	d.RequestPayloads.InstanceCreateReq.Hostname = machineName
	d.RequestPayloads.InstanceCreateReq.Label = machineName

	// ** Handle VPC ** //
	enableVPC := opts.Bool("vultr-enable-vpc")
	vpcSlice := opts.StringSlice("vultr-vpc-ids")

	if enableVPC && len(vpcSlice) > 0 {
		return fmt.Errorf("if enable-vpc is set you cannot attach additional VPC's")
	}

	// ** Set Backups **//
	d.RequestPayloads.InstanceCreateReq.Backups = getBackupStatus(opts.Bool("vultr-vps-backups"))

	d.RequestPayloads.InstanceCreateReq.Region = opts.String("vultr-region")
	d.RequestPayloads.InstanceCreateReq.Plan = opts.String("vultr-vps-plan")
	d.RequestPayloads.InstanceCreateReq.Tags = opts.StringSlice("vultr-tags")
	d.RequestPayloads.InstanceCreateReq.OsID = opts.Int("vultr-os-id")
	d.RequestPayloads.InstanceCreateReq.ISOID = opts.String("vultr-iso-id")
	d.RequestPayloads.InstanceCreateReq.AppID = opts.Int("vultr-app-id")
	d.RequestPayloads.InstanceCreateReq.ImageID = opts.String("vultr-image-id")
	d.RequestPayloads.InstanceCreateReq.SnapshotID = opts.String("vultr-snapshot-id")
	d.RequestPayloads.InstanceCreateReq.FirewallGroupID = opts.String("vultr-firewall-group-id")
	d.RequestPayloads.InstanceCreateReq.IPXEChainURL = opts.String("vultr-ipxe-chain-url")
	d.RequestPayloads.InstanceCreateReq.ScriptID = opts.String("vultr-startup-script-id")
	d.RequestPayloads.InstanceCreateReq.EnableIPv6 = utils.BoolPtr(opts.Bool("vultr-enabled-ipv6"))
	d.RequestPayloads.InstanceCreateReq.EnableVPC = utils.BoolPtr(opts.Bool("vultr-enable-vpc"))
	d.RequestPayloads.InstanceCreateReq.AttachVPC = opts.StringSlice("vultr-vpc-ids")
	d.RequestPayloads.InstanceCreateReq.SSHKeys = opts.StringSlice("vultr-ssh-key-ids")
	d.RequestPayloads.InstanceCreateReq.DDOSProtection = utils.BoolPtr(opts.Bool("vultr-ddos-protection"))
	d.RequestPayloads.InstanceCreateReq.UserData = opts.String("vultr-cloud-init-user-data")
	d.RequestPayloads.InstanceCreateReq.ReservedIPv4 = opts.String("vultr-floating-ipv4-id")
	d.RequestPayloads.InstanceCreateReq.ActivationEmail = utils.BoolPtr(opts.Bool("vultr-send-activation-email"))

	return nil
}

// NewDriver returns a new driver
func NewDriver(hostname, storePath string) *Driver {
	return &Driver{
		BaseDriver: &drivers.BaseDriver{
			MachineName: hostname,
			StorePath:   storePath,
		},
	}
}

// Create ... Creates the VPS
func (d *Driver) Create() (err error) {
	// Validate the plan is available
	if err := d.validatePlan(); err != nil {
		log.Errorf("Error validating the plan: [%v]", err)
		return err
	}

	// Create new ssh key
	d.createSSHKey()

	// Create instance
	d.ResponsePayloads.Instance, err = d.getVultrClient().Instance.Create(context.Background(), &d.RequestPayloads.InstanceCreateReq)
	if err != nil {
		log.Errorf("Error creating the VPS: [%v]", err)
		return err
	}

	d.InstanceID = d.ResponsePayloads.Instance.ID

	log.Infof("VPS %s successfully created", d.BaseDriver.MachineName)

	// Wait for the VPS obtain an IP address
	for i := 0; i < 60; i++ {
		ip, err := d.GetIP()
		if err != nil {
			log.Infof("Waiting for VPS %s to get ip assigned", d.BaseDriver.MachineName)
			<-time.After(5 * time.Second)
			continue
		}
		log.Infof("VPS %s is now configured with ip address %s", d.BaseDriver.MachineName, ip)
		break
	}

	// We need to also set the IP in the base driver
	d.IPAddress, err = d.GetIP()
	if err != nil {
		return err
	}

	return nil
}

// Start ... starts an instance
func (d *Driver) Start() error {
	if err := d.getVultrClient().Instance.Start(context.Background(), d.InstanceID); err != nil {
		log.Errorf("Error starting VPS %s: [%v]", d.BaseDriver.MachineName, err)
		return err
	}

	return nil
}

// Restart ... power cycles an instance
func (d *Driver) Restart() error {
	if err := d.getVultrClient().Instance.Reboot(context.Background(), d.InstanceID); err != nil {
		log.Errorf("Error power cycling VPS %s: [%v]", d.BaseDriver.MachineName, err)
		return err
	}

	return nil
}

// Kill ... stops a host forcefully
func (d *Driver) Kill() error {
	if err := d.getVultrClient().Instance.Halt(context.Background(), d.InstanceID); err != nil {
		log.Errorf("Error stopping VPS %s: [%v]", d.BaseDriver.MachineName, err)
		return err
	}

	return nil
}

// Remove ... deletes a host
func (d *Driver) Remove() error {
	if err := d.getVultrClient().Instance.Delete(context.Background(), d.InstanceID); err != nil {
		log.Errorf("Error deleting VPS %s: [%v]", d.BaseDriver.MachineName, err)
		return err
	}

	return nil
}

// GetIP ... returns an IP or hostname that this host is available at
func (d *Driver) GetIP() (ip string, err error) {
	// IP is set, all is well
	if len(d.ResponsePayloads.Instance.MainIP) > 0 && d.ResponsePayloads.Instance.MainIP != "0.0.0.0" {
		return d.ResponsePayloads.Instance.MainIP, nil
	}

	// set this instance info again
	err = d.setVPSInstanceResponseAgain()
	if err != nil {
		return ip, err
	}

	// IP is still not set
	if len(d.ResponsePayloads.Instance.MainIP) == 0 || d.ResponsePayloads.Instance.MainIP == "0.0.0.0" {
		return ip, fmt.Errorf("VPS Main IP is not available yet")
	}

	// All is well
	return d.ResponsePayloads.Instance.MainIP, nil
}

// GetURL ... returns a Docker compatible host URL for connecting to this host
func (d *Driver) GetURL() (ip string, err error) {
	if err := drivers.MustBeRunning(d); err != nil {
		return "", fmt.Errorf("[GetURL]: could not execute drivers.MustBeRunning: %s", err)
	}
	ip, err = d.GetIP()
	if err != nil {
		return ip, err
	}
	return fmt.Sprintf("tcp://%s", net.JoinHostPort(ip, "2376")), nil
}

// GetState ... returns the state that the host is in (running, stopped, etc)
func (d *Driver) GetState() (status state.State, err error) {
	// set this instance info again
	inst, err := d.getVultrClient().Instance.Get(context.Background(), d.InstanceID)
	if err != nil {
		return status, err
	}

	switch strings.ToLower(inst.Status) {
	case "active":
		return state.Running, nil
	case "pending":
		return state.Starting, nil
	case "resizing":
		return state.Starting, nil
	case "suspended":
		return state.Error, nil
	}

	if strings.ToLower(inst.PowerStatus) == "stopped" {
		return state.Stopped, nil
	}

	return state.None, nil
}

// Stop ... should gracefully stop instance but we're just going to halt for now
func (d *Driver) Stop() error {
	return d.Kill()
}

// DriverName ... returns the name of the driver
func (d *Driver) DriverName() string {
	return "vultr"
}

// GetSSHHostname ... returns ip for use with ssh
func (d *Driver) GetSSHHostname() (string, error) {
	return d.GetIP()
}

// setVPSInstanceResponseAgain ... sets the VPS info again
func (d *Driver) setVPSInstanceResponseAgain() (err error) {
	d.ResponsePayloads.Instance, err = d.getVultrClient().Instance.Get(context.Background(), d.InstanceID)
	if err != nil {
		log.Errorf("Error getting the VPS instance info: [%v]", err)
		return err
	}

	return nil
}
