package driver

import (
	"context"
	"fmt"
	"github.com/vultr/govultr/v2"
	"os"
	"strings"

	"github.com/docker/machine/libmachine/log"
	"github.com/docker/machine/libmachine/ssh"
	"golang.org/x/oauth2"
)

func (d *Driver) createSSHKey() error {
	if len(d.RequestPayloads.InstanceCreateReq.SSHKeys) > 0 {
		for _, id := range d.RequestPayloads.InstanceCreateReq.SSHKeys {
			_, err := d.getVultrClient().SSHKey.Get(context.Background(), id)
			if err != nil {
				return err
			}
		}

	}

	err := ssh.GenerateSSHKey(d.GetSSHKeyPath())
	if err != nil {
		log.Errorf("Error generating public ssh key: %v", err)
		return err
	}

	// Grab the SSH key we just created
	publicKey, err := os.ReadFile(fmt.Sprintf("%s.pub", d.GetSSHKeyPath()))
	if err != nil {
		log.Errorf("Error reading public ssh key: %v", err)
		return err
	}

	createRequest := &govultr.SSHKeyReq{
		SSHKey: string(publicKey),
		Name:   d.MachineName,
	}

	key, err := d.getVultrClient().SSHKey.Create(context.Background(), createRequest)
	if err != nil {
		return err
	}

	d.RequestPayloads.InstanceCreateReq.SSHKeys = append(d.RequestPayloads.InstanceCreateReq.SSHKeys, key.ID)

	return nil
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

// getGod.client ... returns a govultr client
func (d *Driver) getVultrClient() *govultr.Client {
	// Setup govultr client
	var config = &oauth2.Config{}
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
