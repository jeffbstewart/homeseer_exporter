package prometheusbridge

import (
	"errors"
	"github.com/jeffbstewart/homeseer_exporter/devstatus"
	"testing"
)

func TestPoll(t *testing.T) {
	save := devstatusget
	defer func() {
		devstatusget = save
	}()
	devstatusget = func(hostPort string, user string, pass string) (*devstatus.StatusReport, error) {
		return &devstatus.StatusReport{
			Devices: []devstatus.Device{
				{
					Name:       "Main Thermostat Temperature",
					Location:   "Living Room",
					Location2:  "Ground Floor",
					Value:      72.0,
					DeviceType: "Z-Wave Temperature",
				},
			},
		}, nil
	}
	var gotErr error
	onError := func(err error) {
		gotErr = err
	}
	opts := Options{
		OnError:   onError,
		Namespace: t.Name(),
		Location1: "Floor",
		Location2: "Room",
	}
	mon, err := internalNew(opts)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	_ = mon.pollOnce()
	if gotErr != nil {
		t.Fatalf("gotErr: got %v, want nil", gotErr)
	}
}

func TestPollFails(t *testing.T) {
	save := devstatusget
	defer func() {
		devstatusget = save
	}()
	devstatusget = func(hostPort string, user string, pass string) (*devstatus.StatusReport, error) {
		return nil, errors.New("gremlins")
	}
	var gotErr error
	onError := func(err error) {
		gotErr = err
	}
	opts := Options{
		OnError:   onError,
		Namespace: t.Name(),
		HostPort:  "1.2.3.4:80",
		Username:  "Tim",
		Password:  "What is your Quest?",
		Location1: "l1",
		Location2: "l2",
	}
	mon, err := internalNew(opts)
	if err != nil {
		t.Fatalf("New(): %v", err)
	}
	_ = mon.pollOnce()
	wantErr := `devstatus.Get("1.2.3.4:80", "Tim", elided): gremlins`
	if gotErr == nil || gotErr.Error() != wantErr {
		t.Errorf("gotErr: got\n%v, want\n%s", gotErr, wantErr)
	}
}
