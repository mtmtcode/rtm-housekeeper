package rtm

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
)

const (
	baseURL = "https://api.rememberthemilk.com/services/rest/"
	authURL = "https://www.rememberthemilk.com/services/auth/"
)

// Client is an RTM API client.
type Client struct {
	APIKey       string
	SharedSecret string
	AuthToken    string
	HTTPClient   *http.Client
}

// NewClient creates a new RTM API client.
func NewClient(apiKey, sharedSecret, authToken string) *Client {
	return &Client{
		APIKey:       apiKey,
		SharedSecret: sharedSecret,
		AuthToken:    authToken,
		HTTPClient:   &http.Client{},
	}
}

// sign generates the API signature for the given parameters.
func (c *Client) sign(params map[string]string) string {
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var buf strings.Builder
	buf.WriteString(c.SharedSecret)
	for _, k := range keys {
		buf.WriteString(k)
		buf.WriteString(params[k])
	}

	return fmt.Sprintf("%x", md5.Sum([]byte(buf.String())))
}

// call makes an API request and returns the parsed response.
func (c *Client) call(method string, extra map[string]string) (*Rsp, error) {
	params := map[string]string{
		"method":   method,
		"api_key":  c.APIKey,
		"format":   "json",
	}
	if c.AuthToken != "" {
		params["auth_token"] = c.AuthToken
	}
	for k, v := range extra {
		params[k] = v
	}
	params["api_sig"] = c.sign(params)

	u, _ := url.Parse(baseURL)
	q := u.Query()
	for k, v := range params {
		q.Set(k, v)
	}
	u.RawQuery = q.Encode()

	resp, err := c.HTTPClient.Get(u.String())
	if err != nil {
		return nil, fmt.Errorf("rtm: request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("rtm: read body failed: %w", err)
	}

	var r Response
	if err := json.Unmarshal(body, &r); err != nil {
		return nil, fmt.Errorf("rtm: unmarshal failed: %w\nbody: %s", err, string(body))
	}

	if r.Rsp.Stat != "ok" {
		if r.Rsp.Err != nil {
			return nil, fmt.Errorf("rtm: API error %s: %s", r.Rsp.Err.Code, r.Rsp.Err.Msg)
		}
		return nil, fmt.Errorf("rtm: unknown error, stat=%s", r.Rsp.Stat)
	}

	return &r.Rsp, nil
}

// GetFrob gets a frob for authentication.
func (c *Client) GetFrob() (string, error) {
	rsp, err := c.call("rtm.auth.getFrob", nil)
	if err != nil {
		return "", err
	}
	return rsp.Frob, nil
}

// GetAuthURL returns the URL the user should visit to authorize the app.
func (c *Client) GetAuthURL(frob string) string {
	params := map[string]string{
		"api_key": c.APIKey,
		"perms":   "delete",
		"frob":    frob,
	}
	params["api_sig"] = c.sign(params)

	u, _ := url.Parse(authURL)
	q := u.Query()
	for k, v := range params {
		q.Set(k, v)
	}
	u.RawQuery = q.Encode()
	return u.String()
}

// GetToken exchanges a frob for an auth token.
func (c *Client) GetToken(frob string) (string, error) {
	rsp, err := c.call("rtm.auth.getToken", map[string]string{"frob": frob})
	if err != nil {
		return "", err
	}
	if rsp.Auth == nil {
		return "", fmt.Errorf("rtm: no auth in response")
	}
	return rsp.Auth.Token, nil
}

// CreateTimeline creates a timeline for write operations.
func (c *Client) CreateTimeline() (string, error) {
	rsp, err := c.call("rtm.timelines.create", nil)
	if err != nil {
		return "", err
	}
	return rsp.Timeline, nil
}

// GetLists returns all lists.
func (c *Client) GetLists() ([]List, error) {
	rsp, err := c.call("rtm.lists.getList", nil)
	if err != nil {
		return nil, err
	}
	if rsp.Lists == nil {
		return nil, nil
	}
	return rsp.Lists.List, nil
}

// AddList creates a new list.
func (c *Client) AddList(timeline, name string) (*ListResp, error) {
	rsp, err := c.call("rtm.lists.add", map[string]string{
		"timeline": timeline,
		"name":     name,
	})
	if err != nil {
		return nil, err
	}
	return rsp.List, nil
}

// GetTaskList returns tasks matching the filter.
// Optional extra parameters can be passed as key-value pairs.
func (c *Client) GetTaskList(filter string, extra ...map[string]string) ([]TaskList, error) {
	params := map[string]string{
		"filter": filter,
	}
	if len(extra) > 0 {
		for k, v := range extra[0] {
			params[k] = v
		}
	}
	rsp, err := c.call("rtm.tasks.getList", params)
	if err != nil {
		return nil, err
	}
	if rsp.Tasks == nil {
		return nil, nil
	}
	return rsp.Tasks.List, nil
}

// MoveTo moves a task to another list.
func (c *Client) MoveTo(timeline string, info TaskInfo, toListID string) error {
	_, err := c.call("rtm.tasks.moveTo", map[string]string{
		"timeline":      timeline,
		"from_list_id":  info.ListID,
		"to_list_id":    toListID,
		"taskseries_id": info.TaskSeriesID,
		"task_id":       info.TaskID,
	})
	return err
}

// UnarchiveList unarchives a list.
func (c *Client) UnarchiveList(timeline, listID string) error {
	_, err := c.call("rtm.lists.unarchive", map[string]string{
		"timeline": timeline,
		"list_id":  listID,
	})
	return err
}

// DeleteTask deletes a task.
func (c *Client) DeleteTask(timeline string, info TaskInfo) error {
	_, err := c.call("rtm.tasks.delete", map[string]string{
		"timeline":      timeline,
		"list_id":       info.ListID,
		"taskseries_id": info.TaskSeriesID,
		"task_id":       info.TaskID,
	})
	return err
}
