// Copyright 2017 The go-oceanconnect authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package oceanconnect

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httputil"
	"strconv"
	"sync"
	"time"
)

// Config struct for client configuration
type Config struct {
	CertFile    string `yaml:"cert_file"` // CertFile is the path to the PEM client certificate
	CertKeyFile string `yaml:"key_file"`  // CertKeyFile is the path to the PEM client certificate public key
	URL         string `yaml:"url"`       // URL where the Oceanconnect API is present
	AppID       string `yaml:"app_id"`    // AppID is the application Identifier
	Secret      string `yaml:"secret"`

	ManufacturerName string `yaml:"manufacturer_name"`
	ManufacturerID   string `yaml:"manufacturer_id"`
	EndUserID        string `yaml:"end_user_id"`
	Location         string `yaml:"location"`
	DeviceType       string `yaml:"device_type"`
	Model            string `yaml:"model"`
}

// Client struct that contains pointer to http client
type Client struct {
	c            *http.Client
	cfg          Config
	token        string
	tokenExpires time.Time
	reqLock      sync.Mutex
}

// GetDevicesStruct struct for function GetDevices
type GetDevicesStruct struct {
	GatewayID string
	NodeType  string
	PageNo    int
	PageSize  int
	Status    string
	StartTime string
	EndTime   string
	Sort      string
}

// NewClient creates new client with certification
func NewClient(c Config) (*Client, error) {
	cert, err := tls.LoadX509KeyPair(c.CertFile, c.CertKeyFile)
	if err != nil {
		return nil, err
	}

	// Setup HTTPS client
	tlsConfig := &tls.Config{
		Certificates:       []tls.Certificate{cert},
		InsecureSkipVerify: true,
	}
	tlsConfig.BuildNameToCertificate()

	return &Client{
		c:   &http.Client{Transport: &http.Transport{TLSClientConfig: tlsConfig}},
		cfg: c,
	}, nil
}

func (c *Client) request(method, urlStr string, body io.Reader) (*http.Response, error) {
	r, err := http.NewRequest(method, c.cfg.URL+urlStr, body)
	if err != nil {
		return nil, err
	}
	return c.doRequest(r)
}

func (c *Client) doRequest(req *http.Request) (*http.Response, error) {
	c.reqLock.Lock()
	defer c.reqLock.Unlock()
	if c.tokenExpires.Before(time.Now().Add(time.Minute * 5)) {
		err := c.Login()
		if err != nil {
			return nil, err
		}
	}
	req.Header.Add("app_key", c.cfg.AppID)
	req.Header.Add("Authorization", c.token)
	req.Header.Add("Content-Type", "application/json")
	return c.c.Do(req)
}

func (c *Client) GetDevice(deviceID string) (*Device, error) {
	resp, err := c.request(http.MethodGet, "/iocm/app/dm/v1.1.0/devices/"+deviceID, nil)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("invalid response code: " + resp.Status)
	}

	// save device response
	d := &Device{client: c}
	if err := json.NewDecoder(resp.Body).Decode(d); err != nil {
		return nil, err
	}
	return d, nil
}

// GetDevices returns struct with devices
func (c *Client) GetDevices(dev GetDevicesStruct) ([]Device, error) {
	resp, err := c.request(http.MethodGet, c.getQueryStringForDeviceGet(dev), nil)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("invalid response code: " + resp.Status)
	}

	// save device response
	d := deviceResponse{}
	if err := json.NewDecoder(resp.Body).Decode(&d); err != nil {
		return nil, err
	}
	var retdevs []Device
	for _, dev := range d.Devices {
		dev.client = c
		retdevs = append(retdevs, dev)
	}
	return retdevs, err
}

// SendCommand send command to target device
func (c *Client) SendCommand(deviceID string, serviceID string, method string, idata interface{}, timeoutSec int64) error {
	type devCmdBodyCommand struct {
		ServiceID string      `json:"serviceId"`
		Method    string      `json:"method"`
		Params    interface{} `json:"paras"`
	}
	type devCmdBody struct {
		DeviceID    string            `json:"deviceId"`
		Command     devCmdBodyCommand `json:"command"`
		CallbackURL string            `json:"callbackUrl"`
		ExpireTime  int64             `json:"expireTime"`
	}

	cmd := devCmdBody{
		DeviceID: deviceID,
		Command: devCmdBodyCommand{
			ServiceID: serviceID,
			Method:    method,
			Params:    idata,
		},
		ExpireTime: timeoutSec,
	}

	body, err := json.Marshal(cmd)
	if err != nil {
		return err
	}

	resp, err := c.request(http.MethodPost, "/iocm/app/cmd/v1.4.0/deviceCommands", bytes.NewBuffer(body))
	if err != nil {
		return err
	}

	httputil.DumpResponse(resp, true)

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return errors.New("invalid response code: " + resp.Status)
	}

	return nil
}

func (c *Client) getQueryStringForDeviceGet(dev GetDevicesStruct) string {
	s := "/iocm/app/dm/v1.1.0/devices?"
	if dev.GatewayID != "" {
		s += "gatewayId=" + dev.GatewayID + "&"
	}
	if dev.NodeType != "" {
		s += "nodeType=" + dev.NodeType + "&"
	}

	s += "pageNo=" + strconv.Itoa(dev.PageNo) + "&"

	if dev.PageSize != 0 {
		s += "pageSize=" + strconv.Itoa(dev.PageSize) + "&"
	}
	if dev.StartTime != "" {
		s += "startTime=" + dev.StartTime + "&"
	}
	if dev.EndTime != "" {
		s += "endTime=" + dev.EndTime + "&"
	}
	if dev.Status != "" {
		s += "status=" + dev.Status + "&"
	}
	if dev.Sort != "" {
		s += "sort=" + dev.Sort
	}
	return s
}
