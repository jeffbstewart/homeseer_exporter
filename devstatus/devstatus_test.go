package devstatus

import (
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/davecgh/go-spew/spew"
)

func TestFetch(t *testing.T) {
	save := httpgetwithbasicauth
	defer func() {
		httpgetwithbasicauth = save
	}()
	httpgetwithbasicauth = func(url string, username string, password string) ([]byte, error) {
		return []byte(`
{"Name":"HomeSeer Devices","Version":"1.0","Devices":[{"ref":392,"name":"Device Name","location":"Room Name","location2":"1st Floor","value":2000,"status":"Status Text","device_type_string":"Z-Wave Central Scene","last_change":"\/Date(1463147447280)\/","relationship":4,"hide_from_view":false,"associated_devices":[391],"device_type":{"Device_API":4,"Device_API_Description":"Plug-In API","Device_Type":0,"Device_Type_Description":"Plug-In Type 0","Device_SubType":91,"Device_SubType_Description":""},"device_image":"","UserNote":"","UserAccess":"Any","status_image":"/images/HomeSeer/status/Scene-Pressed-1.png"}]}`), nil
	}
	got, err := Get("", "", "")
	if err != nil {
		t.Fatalf("Get(): got %v, want nil error", err)
	}
	want := &StatusReport{
		Name:    "HomeSeer Devices",
		Version: "1.0",
		Devices: []Device{
			{
				Reference:         392,
				Name:              "Device Name",
				Location:          "Room Name",
				Location2:         "1st Floor",
				Value:             2000,
				Status:            "Status Text",
				DeviceType:        "Z-Wave Central Scene",
				LastChange:        time.Unix(1463147447, 280000000),
				LastChangeDate:    `/Date(1463147447280)/`,
				Relationship:      Child,
				HideFromView:      false,
				AssociatedDevices: []int{391},
				Type: DeviceType{
					API:                4,
					APIDescription:     "Plug-In API",
					Type:               0,
					TypeDescription:    "Plug-In Type 0",
					SubType:            91,
					SubTypeDescription: "",
				},
				DeviceImage: "/images/HomeSeer/status/Scene-Pressed-1.png",
			},
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Get(): got %s, want %s", spew.Sdump(got), spew.Sdump(want))
	}
}

func TestMalformed(t *testing.T) {
	save := httpgetwithbasicauth
	defer func() {
		httpgetwithbasicauth = save
	}()
	addr := ""
	httpgetwithbasicauth = func(url string, username string, password string) ([]byte, error) {
		addr = url
		return []byte("This is not JSON"), nil
	}
	_, err := Get("addr", "", "")
	wantErr := "invalid character 'T' looking for beginning of value"
	if err == nil || err.Error() != wantErr {
		t.Errorf("Get(...): got %v, want %s", err, wantErr)
	}
	wantAddr := "http://addr/JSON?request=getstatus"
	if addr != wantAddr {
		t.Errorf("Get saw url %q, want %q", addr, wantAddr)
	}
}

func TestGetFail(t *testing.T) {
	save := httpgetwithbasicauth
	defer func() {
		httpgetwithbasicauth = save
	}()
	httpgetwithbasicauth = func(url string, username string, password string) ([]byte, error) {
		return nil, errors.New("gremlins")
	}
	_, err := Get("", "", "")
	wantErr := "gremlins"
	if err == nil || err.Error() != wantErr {
		t.Errorf("Get(...): got %v, want %q", err, wantErr)
	}
}

func TestConvertDate(t *testing.T) {
	input := "/Date(1613971427719-0500)/"
	got, err := convertLastChange(input)
	if err != nil {
		t.Fatalf("convertLastChange(%q): got _, %v, want _, nil", input, err)
	}
	want := time.Unix(1613971427719 / 1000, (1613971427719 % 1000) * 1000000) // .Add(-5*time.Hour)
	if !got.Equal(want) {
		t.Errorf("convertLastChange(%q), got %s, _, want %s, _", input, got.Format("Mon Jan 2 15:04:05 -0700 MST 2006"), want.Format("Mon Jan 2 15:04:05 -0700 MST 2006"))
	}
}

func TestConvertLargeNegativeDate(t *testing.T) {
	input := "/Date(-62135596800000)/"
	got, err := convertLastChange(input)
	if err != nil {
		t.Fatalf("convertLastChange(%q): got _, %v, want _, nil", input, err)
	}
	var want time.Time
	if !got.Equal(want) {
		t.Errorf("convertLastChange(%q): got %s, _, want %s, _", input, got.Format("Mon Jan 2 15:04:05 -0700 MST 2006"), want.Format("Mon Jan 2 15:04:05 -0700 MST 2006"))
	}
}
