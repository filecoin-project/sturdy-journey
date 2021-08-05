package circleci

// This package is a lightly modified version from a project by jszwedko
// https://github.com/jszwedko/go-circleci

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	logging "github.com/ipfs/go-log/v2"
)

var log = logging.Logger("sturdy-journey/circleci")

var (
	defaultBaseURL = &url.URL{Host: "circleci.com", Scheme: "https", Path: "/api/v2/"}
)

type APIError struct {
	HTTPStatusCode int
	Message        string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("%d: %s", e.HTTPStatusCode, e.Message)
}

type Client struct {
	BaseURL    *url.URL
	Token      string
	HTTPClient *http.Client
	Project    string
}

func (c *Client) client() *http.Client {
	if c.HTTPClient == nil {
		return http.DefaultClient
	}

	return c.HTTPClient
}

func (c *Client) baseURL() *url.URL {
	if c.BaseURL == nil {
		return defaultBaseURL
	}

	return c.BaseURL
}

func (c *Client) request(method, path string, bodyStruct, responseStruct interface{}) error {
	u := c.baseURL().ResolveReference(&url.URL{Path: path})
	req, err := http.NewRequest(method, u.String(), nil)
	if err != nil {
		return err
	}

	if bodyStruct != nil {
		b, err := json.Marshal(bodyStruct)
		if err != nil {
			return err
		}

		req.Body = io.NopCloser(bytes.NewBuffer(b))
	}

	req.Header.Add("Accept", "application/json")
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Circle-Token", c.Token)

	out, err := httputil.DumpRequestOut(req, true)
	if err != nil {
		log.Debugf("error debugging request %+v: %s", req, err)
	}
	log.Debugf("request:\n%+v", strings.Replace(string(out), c.Token, "**REDACTED**", -1))

	resp, err := c.client().Do(req)
	if err != nil {
		return err
	}

	out, err = httputil.DumpResponse(resp, true)
	if err != nil {
		log.Debugf("error debugging response %+v: %s", resp, err)
	}
	log.Debugf("response:\n%+v", string(out))

	if resp.StatusCode >= 300 {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return &APIError{HTTPStatusCode: resp.StatusCode, Message: "unable to parse response: %s"}
		}

		if len(body) > 0 {
			message := struct {
				Message string `json:"message"`
			}{}
			err = json.Unmarshal(body, &message)
			if err != nil {
				return &APIError{
					HTTPStatusCode: resp.StatusCode,
					Message:        fmt.Sprintf("unable to parse API response: %s", err),
				}
			}
			return &APIError{HTTPStatusCode: resp.StatusCode, Message: message.Message}
		}

		return &APIError{HTTPStatusCode: resp.StatusCode}
	}

	if responseStruct != nil {
		err = json.NewDecoder(resp.Body).Decode(responseStruct)
		if err != nil {
			return err
		}
	}

	return nil
}

type PipelineCreateRequest struct {
	Branch     string                 `json:"branch,omitempty"`
	Tag        string                 `json:"tag,omitempty"`
	Parameters map[string]interface{} `json:"parameters"`
}

type PipelineCreateResponse struct {
	ID        string     `json:"id"`
	State     string     `json:"state"`
	Number    int        `json:"number"`
	CreatedAt *time.Time `json:"created_at"`
}

func (c *Client) CreatePipeline(branch string, parameters map[string]interface{}) (*PipelineCreateResponse, error) {
	req := &PipelineCreateRequest{
		Branch:     branch,
		Parameters: parameters,
	}

	resp := &PipelineCreateResponse{}

	err := c.request(http.MethodPost, fmt.Sprintf("%s/%s/%s", "project/gh", c.Project, "pipeline"), req, resp)
	if err != nil {
		return nil, err
	}

	return resp, nil
}
