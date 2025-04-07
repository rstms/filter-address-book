package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/spf13/viper"
	"io/ioutil"
	"net/http"
	"regexp"
)

type Response struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

type BooksResponse struct {
	Response
	Books []string `json:"books"`
}

type FilterControlClient struct {
	url    string
	client *http.Client
}

func NewFilterControlClient() *FilterControlClient {
	c := FilterControlClient{
		url:    viper.GetString("url"),
		client: &http.Client{},
	}
	return &c
}

var ADDR_PATTERN = regexp.MustCompile("^.*<([^>]*)>.*$")

func (c *FilterControlClient) ScanAddressBooks(username, address string) ([]string, error) {
	var response BooksResponse

	matches := ADDR_PATTERN.FindStringSubmatch(address)
	if matches == nil || len(matches) < 2 {
		return []string{}, fmt.Errorf("no email address found in: '%v'\n", address)
	}
	err := c.get(fmt.Sprintf("/scan/%s/%s/", username, matches[1]), &response)
	if err != nil {
		return []string{}, err
	}
	if !response.Success {
		return []string{}, fmt.Errorf("scan request failed: %v\n", response.Message)
	}
	return response.Books, nil
}

func (c *FilterControlClient) request(method, path string, data *[]byte) (*http.Request, error) {
	var body *bytes.Buffer
	if data == nil {
		body = bytes.NewBuffer([]byte{})
	} else {
		body = bytes.NewBuffer(*data)
	}
	req, err := http.NewRequest(method, c.url+path, body)
	if err != nil {
		return nil, fmt.Errorf("failed creating %s request: %v", method, err)
	}
	// we connect directly to filterctld on localhost, bypassing the nginx reverse proxy
	// so set the header indicating a validated client certificate
	req.Header.Set("X-Client-Cert-Dn", "CN=filterctl")
	if data != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return req, nil
}

func (c *FilterControlClient) get(path string, ret interface{}) error {
	req, err := c.request("GET", path, nil)
	if err != nil {
		return fmt.Errorf("failed creating GET request: %v", err)
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("GET %s failed: %v", path, err)
	}
	defer resp.Body.Close()
	return c.handleResponse("GET", path, resp, ret)
}

func (c *FilterControlClient) handleResponse(method, path string, resp *http.Response, ret interface{}) error {
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("%s %s failed reading response body: %v", method, path, err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("Error: %s %s '%s'\n%s", method, path, resp.Status, FormatIfJSON(body))
	}
	if len(body) == 0 {
		return nil
	}
	err = json.Unmarshal(body, ret)
	if err != nil {
		return fmt.Errorf("failed decoding response: %v\n%v", err, string(body))
	}
	return nil
}

func FormatIfJSON(body []byte) string {
	if len(body) == 0 {
		return ""
	}
	decoded := map[string]interface{}{}
	err := json.Unmarshal(body, &decoded)
	if err != nil {
		return string(body)
	}
	formatted, err := json.MarshalIndent(&decoded, "", "  ")
	if err != nil {
		return string(body)
	}
	return string(formatted)
}
