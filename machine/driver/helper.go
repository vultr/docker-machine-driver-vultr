package driver

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"strings"

	"github.com/docker/machine/libmachine/ssh"
	log "github.com/sirupsen/logrus"
	"github.com/vultr/govultr/v2"
	"golang.org/x/oauth2"
)

// addSSHKeyToCloudInitUserData ... generates a new sshkey and adds it to cloud-init userdata cloud-config
func (d *Driver) addSSHKeyToCloudInitUserData() error {
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

// getNewPublicSSHKey ... generates a fresh public ssh key based off the path to the private ssh key
func (d *Driver) getNewPublicSSHKey() (publicKey []byte, err error) {
	// Generate Public SSH Key
	err = ssh.GenerateSSHKey(d.GetSSHKeyPath())
	if err != nil {
		log.Errorf("Error generating public ssh key: %v", err)
		return publicKey, err
	}

	// Grab the SSH key we just created
	publicKey, err = os.ReadFile(fmt.Sprintf("%s.pub", d.GetSSHKeyPath()))
	if err != nil {
		log.Errorf("Error reading public ssh key: %v", err)
		return publicKey, err
	}

	log.Infof("SSH pub key ready (%s)", publicKey)

	return publicKey, nil
}

// validatePlan ... checks plan is available in region
func (d *Driver) validatePlan() error {

	// List plan type
	plantype := strings.Split(d.RequestPayloads.InstanceCreateReq.Plan, "-")
	plans, _, err := d.getVultrClient().Plan.List(context.Background(), plantype[0], &govultr.ListOptions{Region: d.RequestPayloads.InstanceCreateReq.Region, PerPage: 500})
	if err != nil {
		log.Errorf("Error getting getting Plan List: [%v]", err)
		return err
	}

	// Couple scenarios where this error will return
	notAvailableErr := fmt.Errorf("Plan %s not available in region %s .", d.RequestPayloads.InstanceCreateReq.Plan, d.RequestPayloads.InstanceCreateReq.Region)

	// Loop through plans
	for _, plan := range plans {
		// Plan is listed
		if plan.ID == d.RequestPayloads.InstanceCreateReq.Plan {
			// No locations listed
			if len(plan.Locations) == 0 {
				return notAvailableErr
			}

			// Loop through the locations and try to find a match
			for _, location := range plan.Locations {
				// Plan found
				if location == d.RequestPayloads.InstanceCreateReq.Region {
					return nil
				}
			}
		}
	}

	return notAvailableErr
}

// addUFWCommandsToCloudInitUserDataCloudConfig ...
func (d *Driver) addUFWCommandsToCloudInitUserDataCloudConfig() {

	// First add the run command
	d.appendToCloudInitUserDataCloudConfig([]byte("\r\nruncmd:"))

	// Let's keep track of this
	var dockerPortWasOpened bool
	dockerPortAsString := string(d.DockerPort)

	// Now add all the UFW rules
	for _, port := range d.UFWPortsToOpen {
		// A little insurance to make sure we opened the docker port
		if port == dockerPortAsString {
			dockerPortWasOpened = true
		}

		// Add to the cloud init user data cloud config
		d.appendToCloudInitUserDataCloudConfig([]byte("\r\n  - ufw allow " + port))
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
func (d *Driver) appendToCloudInitUserDataCloudConfig(additionalCloudConfig []byte) {
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

// getGod.client ... returns a govultr client
func (d *Driver) getVultrClient() *govultr.Client {
	// Setup govultr client
	config := &oauth2.Config{}
	ctx := context.Background()
	ts := config.TokenSource(ctx, &oauth2.Token{AccessToken: d.APIKey})

	return govultr.NewClient(oauth2.NewClient(ctx, ts))
}

func getBackupStatus(status bool) string {
	if status {
		return "enabled"
	}
	return "disabled"
}
