// Package devstatus retrieves details about HomeSeer devices from the /JSON?request=getstatus interface.
package devstatus

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/golang/glog"
	"io/ioutil"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var httpgetwithbasicauth = getWithBasicAuth

func basicAuth(username string, password string) string {
	token := fmt.Sprintf("%s:%s", username, password)
	return fmt.Sprintf("Basic %s", base64.StdEncoding.EncodeToString([]byte(token)))
}

// 2tatusCodeError signals a non-200 response.
type statusCodeError struct {
	url  string
	code int
}

func (s statusCodeError) Error() string {
	return fmt.Sprintf("http.Get(%q): got code %d, want 200", s.url, s.code)
}


func getWithBasicAuth(url string, username string, password string) ([]byte, error) {
	client := &http.Client{
	}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	if username != "" {
		req.Header.Add("Authorization", basicAuth(username, password))
	}
	r, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	if r.StatusCode != 200 {
		return nil, statusCodeError{url: url, code: r.StatusCode}
	}
	defer func() {
		if err := r.Body.Close(); err != nil {
			glog.Errorf("r.Body.Close(): %v", err)
		}
	}()
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}
	return body, nil
}

// Get retrieves all devices from the given HS3 instance.
func Get(hostPort string, username string, password string) (*StatusReport, error) {
	url := fmt.Sprintf("http://%s/JSON?request=getstatus", hostPort)
	payload, err := httpgetwithbasicauth(url, username, password)
	if err != nil {
		return nil, err
	}
	rval := &StatusReport{}
	err = json.NewDecoder(bytes.NewReader(payload)).Decode(rval)
	if err != nil {
		return nil, err
	}
	if strings.HasPrefix(rval.Response, "Error") {
		return nil, fmt.Errorf("homeseer error: %q", rval.Response)
	}
	if err := rval.convertLastChange(); err != nil {
		return nil, err
	}
	return rval, nil
}

var lastChangePattern = regexp.MustCompile(`^/Date\((-?\d+)([+\\-]\d+)?\)/$`)

func convertLastChange(in string) (time.Time, error) {
	var never time.Time
	parts := lastChangePattern.FindStringSubmatch(in)
	if len(parts) != 3 && len(parts) != 2 {
		return never, fmt.Errorf("malformed date: %q", in)
	}
	ifrm, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return never, err
	}
	if ifrm < 0 {
		// some devices report large negative last changed times.
		// It's unclear why, but I don't want to fail the parse over it.
		return never, nil
	}
	// might have the timezone wrong here, but I'm not using it, so I don't care yet.
	/*tz, err := time.Parse("Mon Jan 2 15:04:05 2006 -0700", "Mon Jan 2 15:04:05 2006 " + parts[2])
	if err != nil {
		return never, err
	}*/
	base := time.Unix(ifrm/1000, (ifrm%1000) * 1000000) // .In(tz.Location())
	return base, nil
}

type StatusReport struct {
	Name    string
	Version string
	Devices []Device
	Response string
}

func (s *StatusReport) convertLastChange() error {
	for i, d := range s.Devices {
		lc, err := convertLastChange(d.LastChangeDate)
		if err != nil {
			return err
		}
		s.Devices[i].LastChange = lc
	}
	return nil
}

type Device struct {
	Reference  int     `json:"ref"`
	Name       string  `json:"name"`
	Location   string  `json:"location"`
	Location2  string  `json:"location2"`
	Value      float64 `json:"value"`
	Status     string  `json:"status"`
	DeviceType string  `json:"device_type_string"`
	LastChange time.Time
	// LastChangeDate is converted to a time.Time in LastChange for convenience.
	LastChangeDate    string     `json:"last_change"`
	Relationship      RelType    `json:"relationship"`
	HideFromView      bool       `json:"hide_from_view"`
	AssociatedDevices []int      `json:"associated_devices"`
	Type              DeviceType `json:"device_type"`
	DeviceImage       string     `json:"status_image"`
}

type DeviceType struct {
	API                int    `json:"Device_API"`
	APIDescription     string `json:"Device_API_Description"`
	Type               int    `json:"Device_Type"`
	TypeDescription    string `json:"Device_Type_Description"`
	SubType            int    `json:"Device_SubType"`
	SubTypeDescription string `json:"Device_SubType_Description"`
}

type RelType int

const (
	RootDevice RelType = 2
	Standalone RelType = 3
	Child      RelType = 4
)
