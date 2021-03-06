// Copyright 2017 The go-oceanconnect authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package oceanconnect

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
)

type deviceResponse struct {
	Totalcount int
	PageNo     int
	Pagesize   int
	Devices    []Device
}

// Subscribe to notifications
func (c *Client) Subscribe(url string) (*Server, error) {
	b := struct {
		NotifyType  string `json:"notifyType"`
		CallbackURL string `json:"callbackurl"`
	}{
		NotifyType:  "deviceDataChanged",
		CallbackURL: url,
	}
	body, err := json.Marshal(b)
	if err != nil {
		return nil, err
	}
	resp, err := c.request(http.MethodPost, "/iocm/app/sub/v1.2.0/subscribe", bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusCreated {
		return nil, errors.New("invalid response code: " + resp.Status)
	}
	return &Server{}, nil
}

// RegistrationReply for RegisterDevice
type RegistrationReply struct {
	VerifyCode string `json:"verifyCode"`
	DeviceID   string `json:"deviceId"`
	Timeout    uint   `json:"timeout"`
	Psk        string `json:"psk"`
}

// RegisterDevice registers a device with a corresponding IMEI number
func (c *Client) RegisterDevice(imei string, timeoutV ...uint) (*RegistrationReply, error) {
	type regDevice struct {
		VerifyCode string `json:"verifyCode"`
		NodeID     string `json:"nodeId"`
		Timeout    uint   `json:"timeout"`
		EndUserID  string `json:"endUserId"`
	}

	var timeout uint

	if len(timeoutV) > 0 {
		timeout = timeoutV[0]
	}

	b := regDevice{
		VerifyCode: imei,
		NodeID:     imei,
		Timeout:    timeout,
		EndUserID:  c.cfg.EndUserID,
	}
	body, err := json.Marshal(b)
	if err != nil {
		return nil, err
	}
	resp, err := c.request(http.MethodPost, "/iocm/app/reg/v1.2.0/devices?appId="+c.cfg.AppID, bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("invalid response code: " + resp.Status)
	}
	d := RegistrationReply{}
	if err := json.NewDecoder(resp.Body).Decode(&d); err != nil {
		return nil, err
	}
	return &d, nil
}

func (c *Client) SetDeviceInfo(deviceID, name string) error {
	b := struct {
		Name             string `json:"name"`
		Mute             string `json:"mute"`
		ManufacturerID   string `json:"manufacturerId"`
		ManufacturerName string `json:"manufacturerName"`
		Location         string `json:"location"`
		DeviceType       string `json:"deviceType"`
		ProtocolType     string `json:"protocolType"`
		Model            string `json:"model"`
	}{
		Name:             name,
		Mute:             "FALSE",
		ManufacturerID:   c.cfg.ManufacturerID,
		ManufacturerName: c.cfg.ManufacturerName,
		Location:         c.cfg.Location,
		DeviceType:       c.cfg.DeviceType,
		ProtocolType:     "CoAP",
		Model:            c.cfg.Model,
	}

	body, err := json.Marshal(b)
	if err != nil {
		return err
	}
	resp, err := c.request(http.MethodPut, "/iocm/app/dm/v1.2.0/devices/"+deviceID+"?appId="+c.cfg.AppID, bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusNoContent {
		return errors.New("invalid response code: " + resp.Status)
	}
	return nil
}

func (c *Client) DeleteDevice(deviceID string) error {

	resp, err := c.request(http.MethodDelete, "/iocm/app/dm/v1.1.0/devices/"+deviceID, nil)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusNoContent {
		return errors.New("invalid response code: " + resp.Status)
	}

	return nil
}
