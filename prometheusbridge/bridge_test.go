package prometheusbridge

import (
	"errors"
	"testing"
	"time"

	"github.com/golang/glog"
	"github.com/jeffbstewart/homeseer_exporter/devstatus"
)

func TestPoll(t *testing.T) {
	save, savet := devstatusget, maketicker
	defer func() {
		devstatusget = save
		maketicker = savet
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
	ch := make(chan time.Time)
	ti := &time.Ticker{
		C: ch,
	}
	maketicker = func() *time.Ticker {
		return ti
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
	c, err := New(opts)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	glog.Info("Forcing a poll now")
	ch <- time.Now()
	if err := c.Close(); err != nil {
		t.Errorf("Close: %v", err)
	}

	if gotErr != nil {
		t.Fatalf("gotErr: got %v, want nil", gotErr)
	}
}

func TestPollFails(t *testing.T) {
	save, savet := devstatusget, maketicker
	defer func() {
		devstatusget = save
		maketicker = savet
	}()
	devstatusget = func(hostPort string, user string, pass string) (*devstatus.StatusReport, error) {
		return nil, errors.New("gremlins")
	}
	ch := make(chan time.Time)
	ti := &time.Ticker{
		C: ch,
	}
	maketicker = func() *time.Ticker {
		return ti
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
	c, err := New(opts)
	if err != nil {
		t.Fatalf("New(): %v", err)
	}
	if err := c.Close(); err != nil {
		t.Errorf("Close: %v", err)
	}
	wantErr := `devstatus.Get("1.2.3.4:80", "Tim", elided): gremlins`
	if gotErr == nil || gotErr.Error() != wantErr {
		t.Errorf("gotErr: got\n%v, want\n%s", gotErr, wantErr)
	}
}
