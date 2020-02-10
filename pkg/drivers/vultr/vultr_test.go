package vultr

import (
	"reflect"
	"testing"

	"github.com/vultr/govultr"

	"github.com/docker/machine/libmachine/drivers"
	"github.com/stretchr/testify/assert"
)

type caseInspect struct {
	specified, existing     []string
	available, notAvailable []string
}

type caseRemove struct {
	slice       []string
	item        string
	resultSlice []string
}

type caseSSHAvailable struct {
	osFamily     string
	availability bool
}

type caseIpv4 struct {
	ip      string
	correct bool
}

type caseMainIp struct {
	driver Driver
	isSet  bool
}

type caseDriverValidation struct {
	driver Driver
	err    bool
}

func TestInspect(t *testing.T) {

	var emptySliceString []string

	cases := []caseInspect{
		{
			specified:    []string{"1", "2", "3"},
			existing:     []string{"1", "2", "3"},
			available:    []string{"1", "2", "3"},
			notAvailable: emptySliceString,
		},
		{
			specified:    []string{"1", "2", "3", "4"},
			existing:     []string{"1", "2", "3"},
			available:    []string{"1", "2", "3"},
			notAvailable: []string{"4"},
		},
		{
			specified:    []string{"1", "2", "3", "4"},
			existing:     emptySliceString,
			available:    emptySliceString,
			notAvailable: []string{"1", "2", "3", "4"},
		},
	}

	for i, item := range cases {
		a, n := inspect(item.specified, item.existing)

		if !reflect.DeepEqual(a, item.available) {
			t.Errorf("[%d] wrong results: got %+v, expected %+v",
				i, a, item.available)
		}

		if !reflect.DeepEqual(n, item.notAvailable) {
			t.Errorf("[%d] wrong results: got %+v, expected %+v",
				i, n, item.notAvailable)
		}
	}
}

func TestRemove(t *testing.T) {
	cases := []caseRemove{
		{
			slice:       []string{"1", "2", "3", "4"},
			item:        "1",
			resultSlice: []string{"2", "3", "4"},
		},
		{
			slice:       []string{"1", "2", "3", "4"},
			item:        "4",
			resultSlice: []string{"1", "2", "3"},
		},
		{
			slice:       []string{},
			item:        "3",
			resultSlice: []string{},
		},
	}

	for i, item := range cases {
		result := remove(item.slice, item.item)

		if !reflect.DeepEqual(result, item.resultSlice) {
			t.Errorf("[%d] wrong results: got %+v, expected %+v",
				i, result, item.resultSlice)
		}
	}
}

func TestIsSSHKeyAvailable(t *testing.T) {
	cases := []caseSSHAvailable{
		{
			osFamily:     "snapshot",
			availability: false,
		},
		{
			osFamily:     "iso",
			availability: false,
		},
		{
			osFamily:     "windows",
			availability: false,
		},
		{
			osFamily:     "",
			availability: false,
		},
		{
			osFamily:     "debian",
			availability: true,
		},
	}

	for i, item := range cases {
		result := isSSHKeyAvailable(item.osFamily)

		if !reflect.DeepEqual(result, item.availability) {
			t.Errorf("[%d] wrong results: got %+v, expected %+v",
				i, result, item.availability)
		}
	}
}

func TestIsIPv4(t *testing.T) {
	cases := []caseIpv4{
		{
			ip:      "1.2.3",
			correct: false,
		},
		{
			ip:      "err.2.3.4",
			correct: false,
		},
		{
			ip:      "1.2.3.444",
			correct: false,
		},
		{
			ip:      "1.2.3.4",
			correct: true,
		},
	}

	for i, item := range cases {
		result := isIPv4(item.ip)

		if !reflect.DeepEqual(result, item.correct) {
			t.Errorf("[%d] wrong results: ip %+v, got %+v, expected %+v",
				i, item.ip, result, item.correct)
		}
	}
}

func TestSetConfigFromFlags(t *testing.T) {
	driver := NewDriver("", "")

	checkFlags := &drivers.CheckDriverOptions{
		FlagsValues: map[string]interface{}{
			"vultr-api-key": "API-KEY",
		},
		CreateFlags: driver.GetCreateFlags(),
	}

	err := driver.SetConfigFromFlags(checkFlags)

	assert.NoError(t, err)
	assert.Empty(t, checkFlags.InvalidFlags)
}

func TestMainIpIsSet(t *testing.T) {
	cases := []caseMainIp{
		{
			Driver{
				BaseDriver: &drivers.BaseDriver{
					IPAddress: "",
				},
			},
			false,
		},
		{
			Driver{
				BaseDriver: &drivers.BaseDriver{
					IPAddress: "0",
				},
			},
			false,
		},
		{
			Driver{
				BaseDriver: &drivers.BaseDriver{
					IPAddress: "0.0.0.0",
				},
			},
			false,
		},
		{
			Driver{
				BaseDriver: &drivers.BaseDriver{
					IPAddress: "1.2.3.4",
				},
			},
			true,
		},
	}

	for i, item := range cases {
		result := item.driver.mainIpIsSet()

		if !reflect.DeepEqual(result, item.isSet) {
			t.Errorf("[%d] wrong results: got %+v, expected %+v",
				i, result, item.isSet)
		}
	}
}

func TestApiCredentials(t *testing.T) {

	cases := []caseDriverValidation{
		{
			driver: Driver{},
			err:    false,
		},
	}

	ts := NewMockApiServer()
	defer ts.Close()

	client := govultr.NewClient(nil, "")
	_ = client.SetBaseURL(ts.URL)

	for i, item := range cases {
		item.driver.client = client
		err := item.driver.validateApiCredentials()

		if err != nil && !item.err {
			t.Errorf("[%d] unexpected error: %#v", i, err)
		}
	}
}

func TestPlan(t *testing.T) {

	cases := []caseDriverValidation{
		{
			driver: Driver{
				VpsPlanId:  100,
				ServerType: serverTypeBareMetal,
				DCID:       5,
			},
			err: false,
		},
		{
			driver: Driver{
				VpsPlanId:  201,
				ServerType: serverTypeBareMetal,
				DCID:       5,
			},
			err: true,
		},
		{
			driver: Driver{
				VpsPlanId:  201,
				ServerType: serverTypeBareMetal,
				DCID:       3,
			},
			err: true,
		},
		{
			driver: Driver{
				VpsPlanId:  100,
				ServerType: serverTypeDedicatedCloud,
				DCID:       5,
			},
			err: false,
		},
		{
			driver: Driver{
				VpsPlanId:  400,
				ServerType: serverTypeDedicatedCloud,
				DCID:       5,
			},
			err: true,
		},
		{
			driver: Driver{
				VpsPlanId:  100,
				ServerType: serverTypeDedicatedCloud,
				DCID:       3,
			},
			err: true,
		},
		{
			driver: Driver{

				VpsPlanId:  100,
				ServerType: serverTypeSSD,
				DCID:       5,
			},
			err: false,
		},
		{
			driver: Driver{

				VpsPlanId:  777,
				ServerType: serverTypeSSD,
				DCID:       5,
			},
			err: true,
		},
		{
			driver: Driver{

				VpsPlanId:  100,
				ServerType: serverTypeSSD,
				DCID:       3,
			},
			err: true,
		},
	}

	ts := NewMockApiServer()
	defer ts.Close()

	client := govultr.NewClient(nil, "")
	_ = client.SetBaseURL(ts.URL)

	for i, item := range cases {
		item.driver.client = client
		err := item.driver.validatePlan()

		if err != nil && !item.err {
			t.Errorf("[%d] unexpected error: %#v", i, err)
		}
	}
}

func TestOSID(t *testing.T) {
	cases := []caseDriverValidation{
		{
			driver: Driver{
				OSID: 127,
			},
			err: false,
		},
		{
			driver: Driver{
				OSID: 401,
			},
			err: true,
		},
	}

	ts := NewMockApiServer()
	defer ts.Close()

	client := govultr.NewClient(nil, "")
	_ = client.SetBaseURL(ts.URL)

	for i, item := range cases {
		item.driver.client = client
		err := item.driver.validateOSID()

		if err != nil && !item.err {
			t.Errorf("[%d] unexpected error: %#v", i, err)
		}
	}
}

func TestDCID(t *testing.T) {
	cases := []caseDriverValidation{
		{
			driver: Driver{
				DCID: 2,
			},
			err: false,
		},
		{
			driver: Driver{
				DCID: 4,
			},
			err: true,
		},
	}

	ts := NewMockApiServer()
	defer ts.Close()

	client := govultr.NewClient(nil, "")
	_ = client.SetBaseURL(ts.URL)

	for i, item := range cases {
		item.driver.client = client
		err := item.driver.validateDCID()

		if err != nil && !item.err {
			t.Errorf("[%d] unexpected error: %#v", i, err)
		}
	}
}

func TestSnapshot(t *testing.T) {
	cases := []caseDriverValidation{
		{
			driver: Driver{
				OSID:       osSnapshot,
				SnapshotId: "5359435d28b9a",
			},
			err: false,
		},
		{
			driver: Driver{
				SnapshotId: "5359435d28b9a",
			},
			err: true,
		},
		{
			driver: Driver{
				OSID:       osSnapshot,
				SnapshotId: "5359435d28b9wrong",
			},
			err: true,
		},
	}

	ts := NewMockApiServer()
	defer ts.Close()

	client := govultr.NewClient(nil, "")
	_ = client.SetBaseURL(ts.URL)

	for i, item := range cases {
		item.driver.client = client
		err := item.driver.validateSnapshot()

		if err != nil && !item.err {
			t.Errorf("[%d] unexpected error: %#v", i, err)
		}
	}
}

func TestISOID(t *testing.T) {
	cases := []caseDriverValidation{
		{
			driver: Driver{
				OSID:  osCustom,
				ISOID: 24,
			},
			err: false,
		},
		{
			driver: Driver{
				ISOID: 53,
			},
			err: true,
		},
		{
			driver: Driver{
				OSID:  osSnapshot,
				ISOID: 53,
			},
			err: true,
		},
	}

	ts := NewMockApiServer()
	defer ts.Close()

	client := govultr.NewClient(nil, "")
	_ = client.SetBaseURL(ts.URL)

	for i, item := range cases {
		item.driver.client = client
		err := item.driver.validateISO()

		if err != nil && !item.err {
			t.Errorf("[%d] unexpected error: %#v", i, err)
		}
	}
}

func TestAppId(t *testing.T) {
	cases := []caseDriverValidation{
		{
			driver: Driver{
				OSID:  osApplication,
				AppId: "2",
			},
			err: false,
		},
		{
			driver: Driver{
				AppId: "3",
			},
			err: true,
		},
		{
			driver: Driver{
				OSID:  osApplication,
				AppId: "4",
			},
			err: true,
		},
	}

	ts := NewMockApiServer()
	defer ts.Close()

	client := govultr.NewClient(nil, "")
	_ = client.SetBaseURL(ts.URL)

	for i, item := range cases {
		item.driver.client = client
		err := item.driver.validateApp()

		if err != nil && !item.err {
			t.Errorf("[%d] unexpected error: %#v", i, err)
		}
	}
}

func TestScript(t *testing.T) {
	cases := []caseDriverValidation{
		{
			driver: Driver{
				ServerType: serverTypeSSD,
				ScriptId:   "3",
			},
			err: false,
		},
		{
			driver: Driver{
				OSID:       osCustom,
				ServerType: serverTypeSSD,
				ScriptId:   "5",
			},
			err: false,
		},
		{
			driver: Driver{
				ServerType: serverTypeBareMetal,
				ScriptId:   "5",
			},
			err: true,
		},
		{
			driver: Driver{
				OSID:       osApplication,
				ServerType: serverTypeSSD,
				ScriptId:   "5",
			},
			err: true,
		},
		{
			driver: Driver{
				OSID:       osCustom,
				ServerType: serverTypeSSD,
				ScriptId:   "3",
			},
			err: true,
		},
	}

	ts := NewMockApiServer()
	defer ts.Close()

	client := govultr.NewClient(nil, "")
	_ = client.SetBaseURL(ts.URL)

	for i, item := range cases {
		item.driver.client = client
		err := item.driver.validateScript()

		if err != nil && !item.err {
			t.Errorf("[%d] unexpected error: %#v", i, err)
		}
	}
}

func TestNetworkId(t *testing.T) {
	cases := []caseDriverValidation{
		{
			driver: Driver{
				NetworkId: []string{"net539626f0798d7", "net53962b0f2341f"},
			},
			err: false,
		},
		{
			driver: Driver{
				NetworkId: []string{"net539626f0798d7", "net53962b0f2341wrong"},
			},
			err: true,
		},
		{
			driver: Driver{
				NetworkId: []string{"net539626f0798d7"},
			},
			err: false,
		},
	}

	ts := NewMockApiServer()
	defer ts.Close()

	client := govultr.NewClient(nil, "")
	_ = client.SetBaseURL(ts.URL)

	for i, item := range cases {
		item.driver.client = client
		err := item.driver.validateNetworkId()

		if err != nil && !item.err {
			t.Errorf("[%d] unexpected error: %#v", i, err)
		}
	}
}

func TestFirewallGroupId(t *testing.T) {
	cases := []caseDriverValidation{
		{
			driver: Driver{
				FirewallGroupId: "1234abcd",
			},
			err: false,
		},
		{
			driver: Driver{
				FirewallGroupId: "1234wrong",
			},
			err: true,
		},
	}

	ts := NewMockApiServer()
	defer ts.Close()

	client := govultr.NewClient(nil, "")
	_ = client.SetBaseURL(ts.URL)

	for i, item := range cases {
		item.driver.client = client
		err := item.driver.validateFirewallGroupId()

		if err != nil && !item.err {
			t.Errorf("[%d] unexpected error: %#v", i, err)
		}
	}
}

func TestSSHKeyId(t *testing.T) {
	cases := []caseDriverValidation{
		{
			driver: Driver{
				SSHKeyIdAvailable: true,
				SSHKeyId:          []string{"541b4960f23bd", "742b4960f23bd"},
			},
			err: false,
		},
		{
			driver: Driver{
				SSHKeyIdAvailable: false,
				SSHKeyId:          []string{"541b4960f23bd", "742b4960f23bd"},
			},
			err: true,
		},
		{
			driver: Driver{
				SSHKeyIdAvailable: true,
				FirewallGroupId:   "1234wrong",
			},
			err: true,
		},
	}

	ts := NewMockApiServer()
	defer ts.Close()

	client := govultr.NewClient(nil, "")
	_ = client.SetBaseURL(ts.URL)

	for i, item := range cases {
		item.driver.client = client
		err := item.driver.validateSSHKeyId()

		if err != nil && !item.err {
			t.Errorf("[%d] unexpected error: %#v", i, err)
		}
	}
}
