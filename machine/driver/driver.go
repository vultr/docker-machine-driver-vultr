package driver

import (
	"context"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"net"
	"strings"
	"time"

	"github.com/docker/machine/libmachine/drivers"
	"github.com/docker/machine/libmachine/mcnflag"
	"github.com/docker/machine/libmachine/ssh"
	"github.com/docker/machine/libmachine/state"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cast"
	"github.com/vultr/docker-machine-driver-vultr/utils"
	govultr "github.com/vultr/govultr/v2"
	"golang.org/x/oauth2"
)

const (
	defaultOSID        = 387 // Ubuntu 20.04
	defaultRegion      = "ewr"
	defaultPlan        = "vc2-1c-2gb"
	defaultDockerPort  = 2376
	defaultBackups     = "disabled"
	defaultLabelPrefix = "vultr-rancher-node-"
)

// VultrDriver ... driver struct
type VultrDriver struct {
	*drivers.BaseDriver
	RequestPayloads struct {
		InstanceCreateReq govultr.InstanceCreateReq
	}
	ResponsePayloads struct {
		Instance *govultr.Instance
	}
	APIKey         string
	DockerPort     int
	UFWPortsToOpen []string
	DisableUFW     bool
}

// getDefaultUFWPortsToOpen ...
func (d *VultrDriver) getDefaultUFWPortsToOpen() []string {
	return []string{"22", "80", "443", "2376", "2379", "2380", "6443", "9099", "9796", "10250", "10254", "30000:32767/tcp", "8472/udp", "30000:32767/udp"}
}

// GetCreateFlags ... returns the mcnflag.Flag slice representing the flags
// that can be set, their descriptions and defaults.
func (d *VultrDriver) GetCreateFlags() []mcnflag.Flag {
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
		mcnflag.StringFlag{
			EnvVar: "VULTR_LABEL",
			Name:   "vultr-label",
			Usage:  "Resource label (default: The supplied machine name)",
		},
		mcnflag.StringSliceFlag{
			EnvVar: "VULTR_TAGS",
			Name:   "vultr-tags",
			Usage:  "Tags you'd like to attach to this resource",
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
			EnvVar: "VULTR_HOSTNAME",
			Name:   "vultr-hostname",
			Usage:  "Hostname you'd like to assign to this resource (default: The supplied machine name)",
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
		},
		mcnflag.StringSliceFlag{
			EnvVar: "VULTR_SSH_KEY_IDS",
			Name:   "vultr-ssh-key-ids",
			Usage:  "SSH Key IDs you'd like installed on this resource. If no SSH Key ID is provided, one will be generated for you",
		},
		mcnflag.StringFlag{
			EnvVar: "VULTR_VPS_BACKUPS",
			Name:   "vultr-vps-backups",
			Usage:  "Enable automatic backups of this VPS (default: disabled)",
			Value:  defaultBackups,
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
		mcnflag.IntFlag{
			EnvVar: "VULTR_DOCKER_PORT",
			Name:   "vultr-docker-port",
			Usage:  "Port the docker machine will host on (default: 2376)",
			Value:  defaultDockerPort,
		},
		mcnflag.BoolFlag{
			EnvVar: "VULTR_DISABLE_OS_FIREWALL",
			Name:   "vultr-disable-os-firewall",
			Usage:  "Disable the UFW firewall that comes standard on every Vultr OS (default: false)",
		},
		mcnflag.StringSliceFlag{
			EnvVar: "VULTR_PORTS_TO_OPEN_ON_OS_FIREWALL",
			Name:   "vultr-ports-to-open-on-os-firewall",
			Usage:  "Comma delimited list of ports to open on the UFW firewall that comes standard on every Vultr OS (default: " + strings.Join(d.getDefaultUFWPortsToOpen()[:], ",") + " )",
			Value:  d.getDefaultUFWPortsToOpen(),
		},
	}
}

// SetConfigFromFlags ... configures the driver with the object that was returned by RegisterCreateFlags
func (d *VultrDriver) SetConfigFromFlags(opts drivers.DriverOptions) error {
	d.APIKey = opts.String("vultr-api-key")
	if len(d.APIKey) == 0 {
		return fmt.Errorf("vultr-api-key cannot be empty")
	}

	defaultHostnameAndLabel := defaultLabelPrefix + cast.ToString(time.Now().Unix())

	// ** Set Label ** //
	// We have nothing to work with, use a default label
	if len(opts.String("vultr-label")) == 0 && len(d.BaseDriver.MachineName) == 0 {
		d.RequestPayloads.InstanceCreateReq.Label = defaultHostnameAndLabel
	}

	// We have a label set, we'll use that
	if len(opts.String("vultr-label")) > 0 {
		d.RequestPayloads.InstanceCreateReq.Label = opts.String("vultr-label")
	}

	// there's no label but we have a machine name so we'll use that
	if len(opts.String("vultr-label")) == 0 && len(d.BaseDriver.MachineName) > 0 {
		d.RequestPayloads.InstanceCreateReq.Label = d.BaseDriver.MachineName
	}

	// ** Set Hostname ** //
	// We have nothing to work with, use a default label as the hostname
	if len(opts.String("vultr-hostname")) == 0 && len(d.BaseDriver.MachineName) == 0 {
		d.RequestPayloads.InstanceCreateReq.Hostname = defaultHostnameAndLabel
	}

	// We have a hostname set, we'll use that
	if len(opts.String("vultr-hostname")) > 0 {
		d.RequestPayloads.InstanceCreateReq.Hostname = opts.String("vultr-hostname")
	}

	// there's no hostname but we have a machine name so we'll use that
	if len(opts.String("vultr-hostname")) == 0 && len(d.BaseDriver.MachineName) > 0 {
		d.RequestPayloads.InstanceCreateReq.Hostname = d.BaseDriver.MachineName
	}

	d.RequestPayloads.InstanceCreateReq.Region = opts.String("vultr-region")
	d.RequestPayloads.InstanceCreateReq.Plan = opts.String("vultr-vps-plan")
	d.RequestPayloads.InstanceCreateReq.Tags = opts.StringSlice("vultr-tags")
	d.RequestPayloads.InstanceCreateReq.OsID = opts.Int("vultr-os-id")
	d.RequestPayloads.InstanceCreateReq.ISOID = opts.String("vultr-iso-id")
	d.RequestPayloads.InstanceCreateReq.AppID = opts.Int("vultr-app-id")
	d.RequestPayloads.InstanceCreateReq.ImageID = opts.String("vultr-image-id")
	d.RequestPayloads.InstanceCreateReq.FirewallGroupID = opts.String("vultr-firewall-group-id")
	d.RequestPayloads.InstanceCreateReq.IPXEChainURL = opts.String("vultr-ipxe-chain-url")
	d.RequestPayloads.InstanceCreateReq.ScriptID = opts.String("vultr-startup-script-id")
	d.RequestPayloads.InstanceCreateReq.EnableIPv6 = utils.BoolPtr(opts.Bool("vultr-enabled-ipv6"))
	d.RequestPayloads.InstanceCreateReq.EnableVPC = utils.BoolPtr(opts.Bool("vultr-enable-vpc"))
	d.RequestPayloads.InstanceCreateReq.AttachVPC = opts.StringSlice("vultr-vpc-ids")
	d.RequestPayloads.InstanceCreateReq.SSHKeys = opts.StringSlice("vultr-ssh-key-ids")
	d.RequestPayloads.InstanceCreateReq.Backups = opts.String("vultr-vps-backups")
	d.RequestPayloads.InstanceCreateReq.DDOSProtection = utils.BoolPtr(opts.Bool("vultr-ddos-protection"))
	d.RequestPayloads.InstanceCreateReq.UserData = opts.String("vultr-cloud-init-user-data")
	d.RequestPayloads.InstanceCreateReq.ReservedIPv4 = opts.String("vultr-floating-ipv4-id")
	d.RequestPayloads.InstanceCreateReq.ActivationEmail = utils.BoolPtr(opts.Bool("vultr-send-activation-email"))
	d.DockerPort = opts.Int("vultr-docker-port")
	d.DisableUFW = opts.Bool("vultr-disable-os-firewall")
	d.UFWPortsToOpen = opts.StringSlice("vultr-ports-to-open-on-os-firewall")

	return nil
}

// NewDriver ... instanciate new driver
func NewDriver(hostname, storePath string) *VultrDriver {
	return &VultrDriver{
		BaseDriver: &drivers.BaseDriver{
			MachineName: hostname,
			StorePath:   storePath,
		},
	}
}

// Create ... Creates the VPS
func (d *VultrDriver) Create() (err error) {
	vultrClient := d.getGoVultrClient()

	// Validate the plan is available
	if err := d.validatePlan(); err != nil {
		log.Errorf("Error validating the plan: [%v]", err)
		return err
	}

	// Create new ssh key if none was supplied
	if len(d.RequestPayloads.InstanceCreateReq.SSHKeys) == 0 {
		d.addSSHKeyToCloudInitUserData()
	}

	// Add all the UFW commands to the cloud init user config
	d.addUFWCommandsToCloudInitUserDataCloudConfig()

	// Create instance
	d.ResponsePayloads.Instance, err = vultrClient.Instance.Create(context.Background(), &d.RequestPayloads.InstanceCreateReq)
	if err != nil {
		log.Errorf("Error creating the VPS: [%v]", err)
		return err
	}

	log.Infof("VPS %s successfully created", d.BaseDriver.MachineName)

	// Wait for the VPS obtain an IP address
	for i := 0; i < 60; i++ {
		_ip, err := d.GetIP()
		if err != nil {
			log.Infof("Waiting for VPS %s to get ip assigned", d.BaseDriver.MachineName)
			<-time.After(5 * time.Second)
			continue
		}
		log.Infof("VPS %s is now configured with ip address %s", d.BaseDriver.MachineName, _ip)
		break
	}

	// We need to also set the IP in the base driver
	d.IPAddress, _ = d.GetIP()

	return nil
}

// Start ... starts an instance
func (d *VultrDriver) Start() error {
	vultrClient := d.getGoVultrClient()

	err := vultrClient.Instance.Start(context.Background(), d.ResponsePayloads.Instance.ID)
	if err != nil {
		log.Errorf("Error starting VPS %s: [%v]", d.BaseDriver.MachineName, err)
		return err
	}

	return nil
}

// Restart ... power cycles an instance
func (d *VultrDriver) Restart() error {
	vultrClient := d.getGoVultrClient()

	err := vultrClient.Instance.Reboot(context.Background(), d.ResponsePayloads.Instance.ID)
	if err != nil {
		log.Errorf("Error power cycling VPS %s: [%v]", d.BaseDriver.MachineName, err)
		return err
	}

	return nil
}

// Kill ... stops a host forcefully
func (d *VultrDriver) Kill() error {
	vultrClient := d.getGoVultrClient()

	err := vultrClient.Instance.Halt(context.Background(), d.ResponsePayloads.Instance.ID)
	if err != nil {
		log.Errorf("Error stopping VPS %s: [%v]", d.BaseDriver.MachineName, err)
		return err
	}

	return nil
}

// Remove ... deltes a host
func (d *VultrDriver) Remove() error {
	vultrClient := d.getGoVultrClient()

	err := vultrClient.Instance.Delete(context.Background(), d.ResponsePayloads.Instance.ID)
	if err != nil {
		log.Errorf("Error deleting VPS %s: [%v]", d.BaseDriver.MachineName, err)
		return err
	}

	return nil
}

// GetIP ... returns an IP or hostname that this host is available at
func (d *VultrDriver) GetIP() (ip string, err error) {
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
func (d *VultrDriver) GetURL() (ip string, err error) {
	ip, err = d.GetIP()
	if err != nil {
		return ip, err
	}
	return fmt.Sprintf("tcp://%s", net.JoinHostPort(ip, cast.ToString(d.DockerPort))), nil
}

// GetState ... returns the state that the host is in (running, stopped, etc)
func (d *VultrDriver) GetState() (_state state.State, err error) {

	// set this instance info again
	err = d.setVPSInstanceResponseAgain()
	if err != nil {
		return _state, err
	}

	log.Infof("Status: %s", d.ResponsePayloads.Instance.Status)
	log.Infof("ServerStatus: %s", d.ResponsePayloads.Instance.ServerStatus)
	log.Infof("PowerStatus: %s", d.ResponsePayloads.Instance.PowerStatus)

	switch d.ResponsePayloads.Instance.Status {
	case "pending":
		return state.Starting, nil
	case "resizing":
		return state.Starting, nil
	case "suspended":
		return state.Error, nil
	}

	switch d.ResponsePayloads.Instance.ServerStatus {
	case "none":
		return state.None, nil
	case "locked":
		return state.Error, nil
	// This ServerStatus errs on the side of safety and exceeds maximum retries
	//case "installingbooting":
	//	return state.Starting, nil
	case "ok":
		return state.Running, nil
	}

	switch d.ResponsePayloads.Instance.PowerStatus {
	case "running":
		return state.Running, nil
	case "stopped":
		return state.Stopped, nil
	}

	return state.None, nil
}

// Stop ... should gracefully stop instance but we're just going to halt for now
func (d *VultrDriver) Stop() error {
	return d.Kill()
}

// DriverName ... returns the name of the driver
func (d *VultrDriver) DriverName() string {
	return "vultr"
}

// GetSSHHostname ... returns ip for use with ssh
func (d *VultrDriver) GetSSHHostname() (string, error) {
	return d.GetIP()
}

// setVPSInstanceResponseAgain ... sets the VPS info again
func (d *VultrDriver) setVPSInstanceResponseAgain() (err error) {
	vultrClient := d.getGoVultrClient()

	d.ResponsePayloads.Instance, err = vultrClient.Instance.Get(context.Background(), d.ResponsePayloads.Instance.ID)
	if err != nil {
		log.Errorf("Error getting the VPS instance info: [%v]", err)
		return err
	}

	return nil
}

// addSSHKeyToCloudInitUserData ... generates a new sshkey and adds it to cloud-init userdata cloud-config
func (d *VultrDriver) addSSHKeyToCloudInitUserData() error {
	// Gets a new public SSH Key
	pubKey, err := d.getNewPublicSSHKey()
	if err != nil {
		log.Errorf("Error getting new public ssh key: %v", err)
		return err
	}

	// Add new authorized key to user data so cloud-init can add it
	sshKey := []byte("\r\nusers:\r\n - name: root\r\n   ssh_authorized_keys:\r\n    - " + string(pubKey))
	d.appendToCloudInitUserDataCloudConfig(sshKey)

	return nil
}

// getNewPublicSSHKey ... generates a fresh public ssh key besed off the path to the private ssh key
func (d *VultrDriver) getNewPublicSSHKey() (publicKey []byte, err error) {
	// Generate Public SSH Key
	err = ssh.GenerateSSHKey(d.GetSSHKeyPath())
	if err != nil {
		log.Errorf("Error generating public ssh key: %v", err)
		return publicKey, err
	}

	// Grab the SSH key we just created
	publicKey, err = ioutil.ReadFile(fmt.Sprintf("%s.pub", d.GetSSHKeyPath()))
	if err != nil {
		log.Errorf("Error reading public ssh key: %v", err)
		return publicKey, err
	}

	log.Infof("SSH pub key ready (%s)", publicKey)

	return publicKey, nil
}

// validatePlan ... checks plan is available in region
func (d *VultrDriver) validatePlan() error {
	vultrClient := d.getGoVultrClient()

	// List plan type
	plantype := strings.Split(d.RequestPayloads.InstanceCreateReq.Plan, "-")
	plans, _, err := vultrClient.Plan.List(context.Background(), plantype[0], &govultr.ListOptions{Region: d.RequestPayloads.InstanceCreateReq.Region, PerPage: 500})
	if err != nil {
		log.Errorf("Error getting getting Plan List: [%v]", err)
		return err
	}

	// Couple scenarios where this error will return
	notAvailableErr := fmt.Errorf("Plan %s not available in region %s", d.RequestPayloads.InstanceCreateReq.Plan, d.RequestPayloads.InstanceCreateReq.Region)

	// Loop through plans
	for _, _plan := range plans {
		// Plan is listed
		if _plan.ID == d.RequestPayloads.InstanceCreateReq.Plan {
			// No locations listed
			if len(_plan.Locations) == 0 {
				return notAvailableErr
			}

			// Loop through the locations and try to find a match
			for _, _location := range _plan.Locations {
				// Plan found
				if _location == d.RequestPayloads.InstanceCreateReq.Region {
					return nil
				}
			}
		}
	}

	return notAvailableErr
}

// addUFWCommandsToCloudInitUserDataCloudConfig ...
func (d *VultrDriver) addUFWCommandsToCloudInitUserDataCloudConfig() {

	// First add the run command
	d.appendToCloudInitUserDataCloudConfig([]byte("\r\nruncmd:"))

	// Lets keep track of this
	var dockerPortWasOpened bool
	dockerPortAsString := cast.ToString(d.DockerPort)

	// Now add all the UFW rules
	for _, _port := range d.UFWPortsToOpen {
		// A little insurance to make sure we opened the docker port
		if _port == dockerPortAsString {
			dockerPortWasOpened = true
		}

		// Add to the cloud init user data cloud config
		d.appendToCloudInitUserDataCloudConfig([]byte("\r\n  - ufw allow " + _port))
	}

	// Docker port was NOT opened, lets do that
	if !dockerPortWasOpened {
		d.appendToCloudInitUserDataCloudConfig([]byte("\r\n  - ufw allow " + dockerPortAsString))
	}

	// Disable firewall
	if d.DisableUFW {
		d.appendToCloudInitUserDataCloudConfig([]byte("\r\n  - ufw disable"))
	}
}

// appendToCloudInitUserDataCloudConfig ... appends to the #cloud-config of the userdata
func (d *VultrDriver) appendToCloudInitUserDataCloudConfig(additionalCloudConfig []byte) {
	var userData []byte
	// There's nothing so lets give it the heading
	if len(d.RequestPayloads.InstanceCreateReq.UserData) == 0 {
		userData = append(userData, []byte("#cloud-config")...)
	} else {
		// There's something, we expect it to be Base64 so lets decode it
		userData, _ = base64.StdEncoding.DecodeString(d.RequestPayloads.InstanceCreateReq.UserData)
	}

	// Append the new data
	userData = append(userData, additionalCloudConfig...)

	// Put it all back
	d.RequestPayloads.InstanceCreateReq.UserData = base64.StdEncoding.EncodeToString(userData)

	// TODO: Handle issue where UserData might not be empty and there's a more complex yaml we need to merge
}

// getGoVultrClient ... returns a govultr client
func (d *VultrDriver) getGoVultrClient() *govultr.Client {
	// Setup govultr client
	config := &oauth2.Config{}
	ctx := context.Background()
	ts := config.TokenSource(ctx, &oauth2.Token{AccessToken: d.APIKey})
	return govultr.NewClient(oauth2.NewClient(ctx, ts))
}
