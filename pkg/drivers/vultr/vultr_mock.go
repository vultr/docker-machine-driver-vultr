package vultr

import (
	"net/http"
	"net/http/httptest"
)

func NewMockApiServer() *httptest.Server {
	apiMux := http.NewServeMux()
	apiMux.HandleFunc("/v1/regions/availability_baremetal", availabilityBareMetal)
	apiMux.HandleFunc("/v1/regions/availability_vdc2", availabilityVdc2)
	apiMux.HandleFunc("/v1/regions/availability", availabilityAll)
	apiMux.HandleFunc("/v1/os/list", osList)
	apiMux.HandleFunc("/v1/regions/list", dcList)
	apiMux.HandleFunc("/v1/app/list", appList)
	apiMux.HandleFunc("/v1/account/info", accountInfo)
	apiMux.HandleFunc("/v1/snapshot/list", snapshotList)
	apiMux.HandleFunc("/v1/iso/list", isoList)
	apiMux.HandleFunc("/v1/startupscript/list", scriptList)
	apiMux.HandleFunc("/v1/network/list", networkList)
	apiMux.HandleFunc("/v1/firewall/group_list", firewallGroupList)
	apiMux.HandleFunc("/v1/sshkey/list", sshKeyList)

	return httptest.NewServer(apiMux)
}

func accountInfo(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "application/json")
	w.Write([]byte(`{"balance":"-500","pending_charges":"4.46","last_payment_date":"2020-01-15 11:55:43","last_payment_amount":"-500.00"}`))
}

func availabilityBareMetal(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "application/json")
	if r.FormValue("DCID") == "5" {
		w.Write([]byte(`[100]`))
	} else {
		w.Write([]byte(`[]`))
	}
}

func availabilityVdc2(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "application/json")
	if r.FormValue("DCID") == "5" {
		w.Write([]byte(`[201,202,203,204,205,206,29,93,94,95,96,97,98,100]`))
	} else {
		w.Write([]byte(`[201,202,203,204,205,206]`))
	}
}

func availabilityAll(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "application/json")
	if r.FormValue("DCID") == "5" {
		w.Write([]byte(`[201,202,203,204,205,206,400,401,402,403,404,29,93,94,95,96,97,98,100]`))
	} else {
		w.Write([]byte(`[201,202,203,204,205,206,400,401,402,403]`))
	}
}

func osList(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "application/json")
	w.Write([]byte(`{
    "127": {
        "OSID": "127",
        "name": "CentOS 6 x64",
        "arch": "x64",
        "family": "centos",
        "windows": false
    },
    "148": {
        "OSID": "148",
        "name": "Ubuntu 12.04 i386",
        "arch": "i386",
        "family": "ubuntu",
        "windows": false
    },
	"186":{
		"OSID":186,
		"name":"Application",
		"arch":"x64",
		"family":"application",
		"windows":false}
}`))
}

func dcList(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "application/json")
	w.Write([]byte(`{
    "1": {
        "DCID": "1",
        "name": "New Jersey",
        "country": "US",
        "continent": "North America",
        "state": "NJ",
        "ddos_protection": true,
        "block_storage": true,
        "regioncode": "EWR"
    },
    "2": {
        "DCID": "2",
        "name": "Chicago",
        "country": "US",
        "continent": "North America",
        "state": "IL",
        "ddos_protection": false,
        "block_storage": false,
        "regioncode": "ORD"
    }
}`))
}

func snapshotList(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "application/json")
	w.Write([]byte(`{
    "5359435d28b9a": {
        "SNAPSHOTID": "5359435d28b9a",
        "date_created": "2014-04-18 12:40:40",
        "description": "Test snapshot",
        "size": "42949672960",
        "status": "complete",
        "OSID": "127",
        "APPID": "0"
    },
    "5359435dc1df3": {
        "SNAPSHOTID": "5359435dc1df3",
        "date_created": "2014-04-22 16:11:46",
        "description": "",
        "size": "10000000",
        "status": "complete",
        "OSID": "127",
        "APPID": "0"
    }
}`))
}

func isoList(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "application/json")
	w.Write([]byte(`{
    "24": {
        "ISOID": 24,
        "date_created": "2014-04-01 14:10:09",
        "filename": "CentOS-6.5-x86_64-minimal.iso",
        "size": 9342976,
        "md5sum": "ec0669895a250f803e1709d0402fc411",
        "sha512sum": "1741f890bce04613f60b4f2b16fb8070c31640c53d4dbb4271b22610150928743eda1207f031b0b5bdd240ef1a6ed21fd5e6a2d192b9c87eff60b6d9698b8260",
        "status": "complete"
    }
}`))
}

func appList(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "application/json")
	w.Write([]byte(`{
    "1": {
        "APPID": "1",
        "name": "LEMP",
        "short_name": "lemp",
        "deploy_name": "LEMP on CentOS 6 x64",
        "surcharge": 0
    },
    "2": {
        "APPID": "2",
        "name": "WordPress",
        "short_name": "wordpress",
        "deploy_name": "WordPress on CentOS 6 x64",
        "surcharge": 0
    }
}`))
}

func scriptList(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "application/json")
	w.Write([]byte(`{
    "3": {
        "SCRIPTID": "3",
        "date_created": "2014-05-21 15:27:18",
        "date_modified": "2014-05-21 15:27:18",
        "name": "test ",
        "type": "boot",
        "script": "#!/bin/bash echo Hello World > /root/hello"
    },
    "5": {
        "SCRIPTID": "5",
        "date_created": "2014-08-22 15:27:18",
        "date_modified": "2014-09-22 15:27:18",
        "name": "test ",
        "type": "pxe",
        "script": "#!ipxe\necho Hello World\nshell"
    }
}`))
}

func networkList(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "application/json")
	w.Write([]byte(`{
    "net539626f0798d7": {
        "DCID": "1",
        "NETWORKID": "net539626f0798d7",
        "date_created": "2017-08-25 12:23:45",
        "description": "test1",
        "v4_subnet": "10.99.0.0",
        "v4_subnet_mask": 24
    },
    "net53962b0f2341f": {
        "DCID": "1",
        "NETWORKID": "net53962b0f2341f",
        "date_created": "2014-06-09 17:45:51",
        "description": "vultr",
        "v4_subnet": "0.0.0.0",
        "v4_subnet_mask": 0
    }
}`))
}

func firewallGroupList(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "application/json")
	w.Write([]byte(`{
    "1234abcd": {
        "FIREWALLGROUPID": "1234abcd",
        "description": "my http firewall",
        "date_created": "2017-02-14 17:48:40",
        "date_modified": "2017-02-14 17:48:40",
        "instance_count": 2,
        "rule_count": 2,
        "max_rule_count": 50
    }
}`))
}

func sshKeyList(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "application/json")
	w.Write([]byte(`{
    "541b4960f23bd": {
        "SSHKEYID": "541b4960f23bd",
        "date_created": null,
        "name": "test",
        "ssh_key": "ssh-rsa AA... test@example.com"
    },
	"742b4960f23bd": {
        "SSHKEYID": "742b4960f23bd",
        "date_created": null,
        "name": "test3",
        "ssh_key": "ssh-rsa AA... test3@example.com"
    }
}`))
}
