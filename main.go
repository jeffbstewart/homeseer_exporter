// promexporter converts select Z-Wave device statuses in homeseer into a format
// legible by the Prometheus time series and monitoring service.
// Homeseer version 4 (hs4) disables the JSON interface by default.  You have to turn it on
// to use this exporter.  I recomment setting up a distinct user for the monitoring.
// That user requires only "Device Control" permisssions.
// This program will not control any of your devices.  It will merely read them.
package main

import (
	"flag"
	"fmt"
	"net/http"

	"github.com/golang/glog"

	"github.com/jeffbstewart/homeseer_exporter/prometheusbridge"
)

var (
	hs4       = flag.String("hs4", "127.0.0.1:8080", "host:port for the homeseer to export")
	port      = flag.Int("port", 6789, "TCP port to export the exporter on")
	user      = flag.String("user", "", "if non empty, the username to present to homeseer")
	pass      = flag.String("pass", "", "if non empty, the password to present to homeseer")
	location1 = flag.String("location1", "room", "prometheus label for Location1")
	location2 = flag.String("location2", "floor", "prometheus label for Location2")
)

func main() {
	flag.Parse()
	if err := prometheusbridge.New(prometheusbridge.Options{
		HostPort: *hs4,
		Username: *user,
		Password: *pass,
		OnError: func(err error) {
			glog.Errorf("prometheusbridge: %v", err)
		},
		Location1: *location1,
		Location2: *location2,
	}); err != nil {
		glog.Fatalf("prometheusbridge.New: %v", err)
	}
	fp := fmt.Sprintf(":%d", *port)
	glog.Infof("Serving HTTP at %s", fp)
	if err := http.ListenAndServe(fp, nil); err != nil {
		glog.Fatalf("http.ListenAndServer(%q): %v", fp, err)
	}
}
