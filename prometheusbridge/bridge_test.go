package prometheusbridge

import (
	"errors"
	"testing"
	"time"

	"github.com/golang/glog"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_model/go"

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
		OnError: onError,
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

	m := &io_prometheus_client.Metric{}
	tg, err := temperature(opts)
	if err != nil {
		t.Fatalf("temperature: %v", err)
	}
	tm := tg.With(prometheus.Labels{
		"floor":  "Ground Floor",
		"room":   "Living Room",
		"device": "Main Thermostat Temperature"})
	if err := tm.Write(m); err != nil {
		t.Fatalf("tm.Write: got %v, want nil error", err)
	}
	got := m.String()
	want := `label:<name:"device" value:"Main Thermostat Temperature" > label:<name:"floor" value:"Ground Floor" > label:<name:"room" value:"Living Room" > gauge:<value:72 > `
	if got != want {
		t.Errorf("tm.Write: got %q, want %q", got, want)
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
		OnError: onError,
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
	wantErr := `devstatus.Get(""): gremlins`
	if gotErr == nil || gotErr.Error() != wantErr {
		t.Errorf("gotErr: got %v, want %q", gotErr, wantErr)
	}
}
