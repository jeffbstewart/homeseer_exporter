# homeseer_exporter
Monitoring Bridge from Homeseer version 4 (hs4) to Prometheus

This program adapts JSON device ouput from HomeSeer's 4th
generation home automation platform, hs4.

This program is not associate with the HomeSeer
company, and it was not authorized by them.

Google, Inc. may have an ownership interest in
this software.

## Permissions

This program uses the /JSON?request=getstatus
interface to read device values.  By default,
this mechanism is disabled in HS4.  To turn it
on, navigate to Setup -> Network, and check
the box next to "Enable Control with JSON".

Next, you need a username and password to use
with the monitoring.  You should setup a
different user and password.  The only required
permission is Device Control.  The example
below assumes you picked the username
"prometheus" and the password "secret".

## Running the monitor

Run homeseer_exporter --help to see all the
flags.  Here's a simple invocation:

```
homeseer_exporter
  --logtostderr
  --hs4=localhost:80
  --user=prometheus
  --password=secret
```


