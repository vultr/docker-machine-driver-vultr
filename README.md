# docker-machine-driver-vultr

Vultr Driver for docker-machine and Rancher Node Driver

## Install as a docker-machine driver

`docker-machine` is now maintained by Rancher as `rancher-machine`. It is recommended to install `rancher-machine` instead. See the official repository: <https://github.com/rancher/machine>.

`docker-machine` is required, [see the installation documentation](https://gitlab.com/gitlab-org/ci-cd/docker-machine/-/blob/main/docs/install-machine.md).

Then install the `docker-machine-driver-vultr` driver by copying the build to `/usr/local/bin/`.

## Install as a Rancher Node Driver

Use overlay2 as the storage engine

### Installing Rancher-Machine

To install `rancher-machine` from source

```bash
eval $(go env)
export PATH="$PATH:$GOPATH/bin"
```

```bash
# Download the Rancher Machine binary
curl -LO https://github.com/rancher/machine/releases/download/v0.15.0-rancher125/rancher-machine-amd64.tar.gz

# Extract the downloaded archive
tar -xzvf rancher-machine-amd64.tar.gz

# Move the binary to a system path
sudo mv rancher-machine /usr/local/bin/rancher-machine

# Ensure the binary is executable
sudo chmod +x /usr/local/bin/rancher-machine
```


And then compile the `docker-machine-driver-vultr` driver:

```bash
go get -u github.com/vultr/docker-machine-driver-vultr@latest
cd $GOPATH/src/github.com/vultr/docker-machine-driver-vultr
make install
```

## Run

You will need a Vultr APIv4 Personal API key. It is only available in the members area <https://my.vultr.com/settings/#settingsapi>. You need to create an account (<https://www.vultr.com/register/>) to get there.

You will also need to use the 19.03.9 install engine as shown in the example below:

```bash
rancher-machine create -d vultr --vultr-api-key=<vultr-api-key> --engine-install-url "https://releases.rancher.com/install-docker/19.03.9.sh" <machine-name>
```


### Options

| Argument                  | Env                       | Default | Description
|---------------------------|---------------------------| --- | ---
| `vultr-api-key`           | `VULTR_API_KEY`           | None | **required** Vultr API key (see [here](https://www.vultr.com/api/#overview))
| `vultr-server-type`       | `VULTR_SERVER_TYPE`       | `1` | Vultr Server Type ( 1-SSD, 2-BareMetal, 3-DedicatedCloud)
| `vultr-region`            | `VULTR_REGION`            | `1` (New Jersey) | VPS DCID (Region) (see [here](https://www.vultr.com/api/#regions))
| `vultr-vps-plan-id`       | `VULTR_VPS_PLAN_ID`       | `201` (1024 MB RAM,25 GB SSD,1.00 TB BW) | VPS Plan ID (see [here](https://www.vultr.com/api/#plans))
| `vultr-os-id`             | `VULTR_OS_ID`             | `1743` (Ubuntu 22.04 x64) | VPS Operating System ID (see [here](https://www.vultr.com/api/#os))
| `vultr-ipxe-chain-url`    | `VULTR_IPXE_CHAIN_URL`    | None | (optional) If you've selected the 'custom' operating system, this can be set to chainload the specified URL on bootup, via iPXE
| `vultr-iso-id`            | `VULTR_ISO_ID`            | None | (optional) If you've selected the 'custom' operating system, this is the ID of a specific ISO to mount during the deployment
| `vultr-script-id`         | `VULTR_SCRIPT_ID`         | None | (optional) If you've not selected a 'custom' operating system, this can be the SCRIPTID of a startup script to execute on boot
| `vultr-snapshot-id`       | `VULTR_SNAPSHOT_ID`       | None | (optional) If you've selected the 'snapshot' operating system, this should be the SNAPSHOTID (see v1/snapshot/list) to restore for the initial installation
| `vultr-app-id`            | `VULTR_APP_ID`            | None | (optional) If launching an application (OSID 186), this is the APPID to launch
| `vultr-reserved-ip-v4`    | `VULTR_RESERVED_IP_V4`    | None | (optional) IP address of the floating IP to use as the main IP of this server
| `vultr-ip-v6`             | `VULTR_IP_V6`             | None | (optional) If true, an IPv6 subnet will be assigned to the machine (where available)
| `vultr-auto-backups`      | `VULTR_AUTO_BACKUPS`      | None | (optional) If true, automatic backups will be enabled for this server (these have an extra charge associated with them)
| `vultr-private-network`   | `VULTR_PRIVATE_NETWORK`   | None | (optional) If true, private networking support will be added to the new server
| `vultr-network-id`        | `VULTR_NETWORK_ID`        | None | (optional) List of private networks to attach to this server. Use either this field or enable_private_network, not both
| `vultr-notify-activate`   | `VULTR_NOTIFY_ACTIVATE`   | None | (optional) If true, an activation email will be sent when the server is ready
| `vultr-ddos-protection`   | `VULTR_DDOS_PROTECTION`   | None | (optional) If true, DDOS protection will be enabled on the subscription (there is an additional charge for this)
| `vultr-userdata`          | `VULTR_USERDATA`          | None | (optional) Base64 encoded user-data
| `vultr-label`             | `VULTR_LABEL`             | None | (optional) This is a text label that will be shown in the control panel
| `vultr-hostname`          | `VULTR_HOSTNAME`          | None | (optional) The hostname to assign to this server
| `vultr-tag`               | `VULTR_TAG`               | None | (optional) The tag to assign to this server
| `vultr-firewall-group-id` | `VULTR_FIREWALL_GROUP_ID` | None | (optional) The firewall group to assign to this server
| `vultr-sshkey-id`         | `VULTR_SSHKEY_ID`         | None | (optional) List of SSH keys to apply to this server on install (only valid for Linux/FreeBSD)


## Debugging

Detailed run output will be emitted when using the `rancher-machine` `--debug` option.

```bash
rancher-machine --debug  create -d vultr --vultr-api-key=<vultr-api-key> machinename
```

## Examples

### Simple Example

```bash
rancher-machine create -d vultr --vultr-api-key=<vultr-api-key> vultr
eval $(rancher-machine env vultr)
```

```bash
$ rancher-machine ls
NAME      ACTIVE   DRIVER   STATE     URL                         SWARM   DOCKER     ERRORS
vultr     -        vultr    Running   tcp://207.246.87.114:2376           v19.03.5

$ rancher-machine rm vultr
About to remove vultr
WARNING: This action will delete both local reference and remote instance.
Are you sure? (y/n): y
Successfully removed vultr
```

## Required Ports

By default, Vultr images have UFW enabled with only port 22 open. The default cloud-init script in this driver completely disables UFW, however, for production environments we recommend only opening specific ports as needed.

The port requirements for Rancher can be found here:
https://ranchermanager.docs.rancher.com/getting-started/installation-and-upgrade/installation-requirements/port-requirements





