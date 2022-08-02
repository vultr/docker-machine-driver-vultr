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
	"github.com/prometheus/common/log"
	"github.com/spf13/cast"
	govultr "github.com/vultr/govultr/v2"
	"golang.org/x/oauth2"
)

const (
	defaultOSID       = 445
	defaultRegion     = "ewr"
	defaultPlan       = "vc2-1c-2gb"
	defaultDockerPort = 2376
	defaultLabel      = "vultr-rancher-node-"
	defaultBackups    = "disabled"
)

// VultrDriver ... driver struct
type VultrDriver struct {
	*drivers.BaseDriver
	RequestPayloads struct {
		*govultr.InstanceCreateReq
		*govultr.SSHKeyReq
	}
	APIKey     string
	DockerPort int
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
			Usage:  "Resource label (default: vultr-rancher-node-CURRENT_UNIX_TS)",
			Value:  defaultLabel + cast.ToString(time.Now().Unix()),
		},
		mcnflag.StringSliceFlag{
			EnvVar: "VULTR_TAGS",
			Name:   "vultr-tags",
			Usage:  "Tags you'd like to attach to this resource",
		},
		mcnflag.IntFlag{
			EnvVar: "VULTR_OSID",
			Name:   "vultr-os-id",
			Usage:  "Operating system ID (default: [445] Ubuntu 21.04)",
			Value:  defaultOSID,
		},
		mcnflag.IntFlag{
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
	}
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

func (d *VultrDriver) Create() error {
	vultrClient := d.getGoVultrClient()

	// Validate the plan is available
	if err := d.validatePlan(); err != nil {
		log.Errorf("Error validating the plan: [%v]", err)
		return err
	}

	// Create new ssh key
	d.addSSHKeyToCloudInitUserData()

	// Create instance
	res, err := vultrClient.Instance.Create(context.Background(), d.RequestPayloads.InstanceCreateReq)

	// Optional changes
	// _ = vultrClient.SetBaseURL("https://api.vultr.com")
	// vultrClient.SetUserAgent("vultr-rancher-node-driver")
	// vultrClient.SetRateLimit(500)
}

// DriverName returns the name of the driver
func (d *VultrDriver) DriverName() string {
	return "vultr"
}

// GetSSHHostname ... returns hostname for use with ssh
func (d *VultrDriver) GetSSHHostname() (string, error) {
	return d.GetIP()
}

// GetURL ... returns a Docker compatible host URL for connecting to this host
func (d *VultrDriver) GetURL() (string, error) {
	ip, err := d.GetIP()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("tcp://%s", net.JoinHostPort(ip, cast.ToString(d.DockerPort))), nil
}

// GetState ... returns the state that the host is in (running, stopped, etc)
func (d *VultrDriver) GetState() (state.State, error) {

	powerState := "ON"

	switch powerState {
	case "ON":
		return state.Running, nil
	case "OFF":
		return state.Stopped, nil
	}
	return state.None, nil
}

// addSSHKeyToCloudInitUserData ... generates a new sshkey and adds it to cloud-init userdata cloud-config
func (d *VultrDriver) addSSHKeyToCloudInitUserData() error {
	// Gets a new public SSH Key
	pubKey, err := d.getNewPublicSSHKey()
	if err != nil {
		log.Errorf("Error getting new public ssh key: %v", err)
		return err
	}

	// Userdata string
	userdata := []byte("#cloud-config\r\nusers:\r\n - name: root\r\n   ssh_authorized_keys:\r\n    - " + string(pubKey))

	// TODO: Handle issue where UserData might not be empty, right now we're straight up overriding it

	// Add base64 encoded userdata to instance create payload
	d.RequestPayloads.InstanceCreateReq.UserData = base64.StdEncoding.EncodeToString(userdata)

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

func (d *VultrDriver) getGoVultrClient() *govultr.Client {
	// Setup govultr client
	config := &oauth2.Config{}
	ctx := context.Background()
	ts := config.TokenSource(ctx, &oauth2.Token{AccessToken: d.APIKey})
	return govultr.NewClient(oauth2.NewClient(ctx, ts))
}
