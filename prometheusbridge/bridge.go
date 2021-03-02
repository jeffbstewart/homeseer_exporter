// Package prometheusbridge fetches HS4 device values via JSON and exports
// them to the prometheus monitoring package.
package prometheusbridge

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/golang/glog"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/jeffbstewart/homeseer_exporter/devstatus"
)

var (
	devstatusget = devstatus.Get
	register     = prometheus.Register
	handle       = http.Handle
)

func gaugeOpts(opts Options, name string, help string) prometheus.GaugeOpts {
	return prometheus.GaugeOpts{
		Namespace: opts.Namespace,
		Subsystem: opts.Subsystem,
		Name:      name,
		Help:      help,
	}
}

func newGaugeVec(opts Options, name string, help string) (*prometheus.GaugeVec, error) {
	r := prometheus.NewGaugeVec(
		gaugeOpts(opts, name, help),
		[]string{
			opts.Location2,
			opts.Location1,
			"device",
			"parentDevice",
		})
	if err := register(r); err != nil {
		return nil, err
	}
	return r, nil
}

func now(opts Options) (prometheus.Gauge, error) {
	r := prometheus.NewGauge(gaugeOpts(opts, "current_unix_time", "seconds elapsed since Jan 1, 1970 UTC"))
	return r, register(r)
}

func lastUpdateUnixTime(opts Options) (*prometheus.GaugeVec, error) {
	return newGaugeVec(opts, "last_update_unix_time",
		"Seconds since Jan 1, 1970 UTC when this device last received an update")
}

func temperature(opts Options) (*prometheus.GaugeVec, error) {
	// TODO(jeffstewart): deal with units.
	return newGaugeVec(opts, "temperature_degreesf",
		"A temperature reading in degrees Fahrenheit")
}

func relativeHumidity(opts Options) (*prometheus.GaugeVec, error) {
	return newGaugeVec(opts, "relative_humidity_percent", "Relative Humidity, 0 to 100%")
}

func luminance(opts Options) (*prometheus.GaugeVec, error) {
	return newGaugeVec(opts, "luminance_lux", "A measure of light intensity")
}

func battery(opts Options) (*prometheus.GaugeVec, error) {
	return newGaugeVec(opts, "battery_percent", "Percent of charge remaining in a battery")
}

func watts(opts Options) (*prometheus.GaugeVec, error) {
	return newGaugeVec(opts, "power_watts", "Instantaneous power consumption")
}

func kwhours(opts Options) (*prometheus.GaugeVec, error) {
	return newGaugeVec(opts, "cumulative_power_kwhours", "Total power consumption over time")
}

func ultraviolet(opts Options) (*prometheus.GaugeVec, error) {
	return newGaugeVec(opts, "ultraviolet_index", "A measure of ultraviolet light exposure")
}

func sensorBinary(opts Options) (*prometheus.GaugeVec, error) {
	return newGaugeVec(opts, "sensor_binary", "A sensor that can be either on or off")
}
func switchBinary(opts Options) (*prometheus.GaugeVec, error) {
	return newGaugeVec(opts, "switch_binary", "A switch that is either on or off")
}

func switchMultilevel(opts Options) (*prometheus.GaugeVec, error) {
	return newGaugeVec(opts, "switch_multilevel", "A dimmable switch")
}

func volts(opts Options) (*prometheus.GaugeVec, error) {
	return newGaugeVec(opts, "potential_volts", "A measure of electrical potential")
}

func amperes(opts Options) (*prometheus.GaugeVec, error) {
	return newGaugeVec(opts, "current_amperes", "Instantaneous electrical current")
}

// Options configures the exporter
type Options struct {
	// HostPort is the host:port of the homeseer 4 server.
	HostPort string
	// Username is the identity to present as authentication.  Empty for none.
	Username string
	// Password is the credential to present as authentication.
	Password string
	// OnError will be informed of fatal errors.
	OnError func(error)
	// Namespace metrics will be exported under
	Namespace string
	// Subsystem metrics will be exportered under
	Subsystem string

	// Location1 will be the namespace key in prometheus for HS4's Location1.
	// Example: "floor"
	Location1 string

	// Location1 will be the namespace key in prometheus for HS4's Location2.
	// Example: "room"
	Location2 string
}

// New creates and starts a monitor for the given target.
// onError will be called if monitoring the target fails.
// onError will be called only once.
func New(opts Options) error {
	_, err := internalNew(opts)
	return err
}

func internalNew(opts Options) (*monitor, error) {
	if opts.Username != "" && opts.Password == "" {
		return nil, fmt.Errorf("when Username is provided you must also provide a password")
	}
	if opts.Location2 == opts.Location1 {
		return nil, fmt.Errorf("options Location1 cannot be the same as Location2")
	}
	glog.Infof("Monitoring homeseer at %s", opts.HostPort)
	rval := &monitor{
		opts:        opts,
		close:       make(chan interface{}, 1),
		promHandler: promhttp.Handler(),
	}

	var err error
	if rval.temperature, err = temperature(opts); err != nil {
		return nil, err
	}
	if rval.relativeHumidity, err = relativeHumidity(opts); err != nil {
		return nil, err
	}
	if rval.luminance, err = luminance(opts); err != nil {
		return nil, err
	}
	if rval.battery, err = battery(opts); err != nil {
		return nil, err
	}
	if rval.watts, err = watts(opts); err != nil {
		return nil, err
	}
	if rval.kwhours, err = kwhours(opts); err != nil {
		return nil, err
	}
	if rval.ultraviolet, err = ultraviolet(opts); err != nil {
		return nil, err
	}
	if rval.sensorBinary, err = sensorBinary(opts); err != nil {
		return nil, err
	}
	if rval.switchBinary, err = switchBinary(opts); err != nil {
		return nil, err
	}
	if rval.switchMultilevel, err = switchMultilevel(opts); err != nil {
		return nil, err
	}
	if rval.volts, err = volts(opts); err != nil {
		return nil, err
	}
	if rval.amperes, err = amperes(opts); err != nil {
		return nil, err
	}
	if rval.now, err = now(opts); err != nil {
		return nil, err
	}

	if rval.lastUpdateUnixTime, err = lastUpdateUnixTime(opts); err != nil {
		return nil, err
	}
	handle("/", http.RedirectHandler("/metrics", 302))
	handle("/metrics", rval)
	return rval, nil
}

// monitor monitors an instance of HS3.
type monitor struct {
	ticker      *time.Ticker
	pulse       chan interface{}
	close       chan interface{}
	wg          sync.WaitGroup
	promHandler http.Handler

	opts Options

	temperature        *prometheus.GaugeVec
	relativeHumidity   *prometheus.GaugeVec
	luminance          *prometheus.GaugeVec
	battery            *prometheus.GaugeVec
	watts              *prometheus.GaugeVec
	kwhours            *prometheus.GaugeVec
	ultraviolet        *prometheus.GaugeVec
	sensorBinary       *prometheus.GaugeVec
	switchBinary       *prometheus.GaugeVec
	switchMultilevel   *prometheus.GaugeVec
	volts              *prometheus.GaugeVec
	amperes            *prometheus.GaugeVec
	now                prometheus.Gauge
	lastUpdateUnixTime *prometheus.GaugeVec
}

func (m *monitor) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	if err := m.pollOnce(); err != nil {
		glog.Errorf("pollOnce(): %v", err)
		rw.WriteHeader(500)
		return
	}
	m.promHandler.ServeHTTP(rw, req)
}

func (m *monitor) pollOnce() error {
	st, err := devstatusget(m.opts.HostPort, m.opts.Username, m.opts.Password)
	if err != nil {
		return fmt.Errorf("devstatus.Get(%q, %q, elided): %v", m.opts.HostPort, m.opts.Username, err)
	}
	m.now.Set(float64(time.Now().Unix()))
	want := map[string]*prometheus.GaugeVec{
		"Z-Wave Temperature":       m.temperature,
		"Z-Wave Relative Humidity": m.relativeHumidity,

		"Z-Wave Battery": m.battery,

		"Z-Wave Luminance":   m.luminance,
		"Z-Wave Ultraviolet": m.ultraviolet,

		"Z-Wave Watts":    m.watts,
		"Z-Wave kW Hours": m.kwhours,
		"Z-Wave Volts":    m.volts,
		"Z-Wave Amperes":  m.amperes,

		"Z-Wave Sensor Binary": m.sensorBinary,

		"Z-Wave Switch":            m.switchBinary,
		"Z-Wave Switch Multilevel": m.switchMultilevel,

		// TODO: Nest special handling
	}
	now := time.Now()
	m.now.Set(float64(now.Unix()))
	deviceNames := make(map[int]string)
	for _, d := range st.Devices {
		deviceNames[d.Reference] = d.Name
	}
	for _, d := range st.Devices {
		device := ""
		parent := ""
		t := d.DeviceType
		if t == "Z-Wave Electric Meter" {
			t = "Z-Wave " + d.Name
		}

		if got, ok := want[t]; ok {
			if d.DeviceType == "Z-Wave Switch Binary" || d.DeviceType == "Z-Wave Switch" {
				// convert 0/255 to 0/1
				if d.Value != 0 {
					d.Value = 1
				}
			}
			device = d.Name
			if len(d.AssociatedDevices) == 1 {
				parent = deviceNames[d.AssociatedDevices[0]]
			}
			labels := prometheus.Labels{
				m.opts.Location2: d.Location2,
				m.opts.Location1: d.Location,
				"device":         device,
				"parentDevice":   parent,
			}
			got.With(labels).Set(d.Value)
			m.lastUpdateUnixTime.With(labels).Set(float64(d.LastChange.Unix()))
		}
	}
	return nil
}
