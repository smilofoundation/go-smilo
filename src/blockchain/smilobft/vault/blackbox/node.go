// Copyright 2019 The go-smilo Authors
// Copyright 2016 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package blackbox

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/tv42/httpunix"
)

func unixTransport(socketPath string) *httpunix.Transport {
	t := &httpunix.Transport{
		DialTimeout:           1 * time.Second,
		RequestTimeout:        5 * time.Second,
		ResponseHeaderTimeout: 5 * time.Second,
	}
	t.RegisterLocation("blackbox", socketPath)
	return t
}

func unixClient(socketPath string) *http.Client {
	return &http.Client{
		Transport: unixTransport(socketPath),
	}
}

func RunNode(socketPath string) error {
	c := unixClient(socketPath)
	res, err := c.Get("http+unix://blackbox/upcheck")
	if err != nil {
		return err
	}
	if res.StatusCode == 200 {
		return nil
	}
	return errors.New("blackbox Node API did not respond to upcheck request")
}

func (c *Client) PostData(pl []byte, b64From string, b64To []string) ([]byte, error) {
	buf := bytes.NewBuffer([]byte(base64.StdEncoding.EncodeToString(pl)))
	req, err := http.NewRequest("POST", "http+unix://blackbox/sendraw", buf)
	if err != nil {
		return nil, err
	}
	if b64From != "" {
		req.Header.Set("bb0x-from", b64From)
	}
	req.Header.Set("bb0x-to", strings.Join(b64To, ","))
	req.Header.Set("Content-Type", "application/octet-stream")
	res, err := c.httpClient.Do(req)

	if res != nil {
		defer res.Body.Close()
	}
	if err != nil {
		return nil, err
	}
	if res.StatusCode != 200 {
		return nil, fmt.Errorf("non-200 status code: %+v", res)
	}

	return ioutil.ReadAll(base64.NewDecoder(base64.StdEncoding, res.Body))
}

func (c *Client) PostDataRawTransaction(signedPayload []byte, b64To []string) ([]byte, error) {
	buf := bytes.NewBuffer(signedPayload)
	req, err := http.NewRequest("POST", "http+unix://blackbox/sendsignedtx", buf)
	if err != nil {
		return nil, err
	}

	req.Header.Set("bb0x-to", strings.Join(b64To, ","))
	req.Header.Set("Content-Type", "application/octet-stream")
	res, err := c.httpClient.Do(req)

	if res != nil {
		defer res.Body.Close()
	}
	if err != nil {
		return nil, err
	}
	if res.StatusCode != 200 {
		return nil, fmt.Errorf("non-200 status code: %+v", res)
	}

	return ioutil.ReadAll(base64.NewDecoder(base64.StdEncoding, res.Body))
}

func (c *Client) GetData(key []byte) ([]byte, error) {
	req, err := http.NewRequest("GET", "http+unix://blackbox/receiveraw", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("bb0x-key", base64.StdEncoding.EncodeToString(key))
	res, err := c.httpClient.Do(req)

	if res != nil {
		defer res.Body.Close()
	}
	if err != nil {
		return nil, err
	}
	if res.StatusCode != 200 {
		return nil, fmt.Errorf("non-200 status code: %+v", res)
	}

	return ioutil.ReadAll(base64.NewDecoder(base64.StdEncoding, res.Body))
}
