package xui

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strconv"
	"strings"

	"github.com/google/uuid"
)

// Client is the interface for interacting with a 3x-ui panel.
type Client interface {
	// Login authenticates and stores the session cookie.
	Login(ctx context.Context) error
	// GetInbound returns a single inbound by its ID.
	GetInbound(ctx context.Context, inboundID int) (*Inbound, error)
	// CreateClient creates a new VLESS-Reality client on the given inbound.
	// email must be unique within the inbound. tgId and comment are stored on
	// the client for later retrieval (owner lookup and display label).
	CreateClient(ctx context.Context, inboundID int, email string, tgId int64, comment string) (*XUIClient, error)
	// SetClientEnabled enables or disables a client within an inbound.
	SetClientEnabled(ctx context.Context, inboundID int, clientUUID string, enable bool) error
	// DeleteClient removes a client from an inbound permanently.
	DeleteClient(ctx context.Context, inboundID int, clientUUID string) error
}

type httpClient struct {
	baseURL    string
	username   string
	password   string
	httpClient *http.Client
}

// NewHTTPClient creates a new 3x-ui HTTP client. Call Login before using other methods.
// baseURL must include the panel web base path, e.g. https://panel.example.com:2053/mypath
func NewHTTPClient(baseURL, username, password string) Client {
	jar, _ := cookiejar.New(nil)
	return &httpClient{
		baseURL:  strings.TrimRight(baseURL, "/"),
		username: username,
		password: password,
		httpClient: &http.Client{
			Jar: jar,
		},
	}
}

// Login authenticates via POST /login/ using multipart/form-data.
func (c *httpClient) Login(ctx context.Context) error {
	body := &bytes.Buffer{}
	w := multipart.NewWriter(body)
	w.WriteField("username", c.username) //nolint:errcheck
	w.WriteField("password", c.password) //nolint:errcheck
	w.Close()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/login/", body)
	if err != nil {
		return fmt.Errorf("xui login: build request: %w", err)
	}
	req.Header.Set("Content-Type", w.FormDataContentType())

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("xui login: do request: %w", err)
	}
	defer resp.Body.Close()

	var result apiResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("xui login: decode response: %w", err)
	}
	if !result.Success {
		return fmt.Errorf("xui login: %s", result.Msg)
	}
	return nil
}

// GetInbound fetches all inbounds and returns the one with the given ID.
func (c *httpClient) GetInbound(ctx context.Context, inboundID int) (*Inbound, error) {
	raw, err := c.doGet(ctx, "/panel/api/inbounds/list")
	if err != nil {
		return nil, fmt.Errorf("xui get inbound: list: %w", err)
	}

	var inbounds []Inbound
	if err := json.Unmarshal(raw, &inbounds); err != nil {
		return nil, fmt.Errorf("xui get inbound: decode: %w", err)
	}

	for i := range inbounds {
		if inbounds[i].ID == inboundID {
			return &inbounds[i], nil
		}
	}
	return nil, fmt.Errorf("xui get inbound: inbound %d not found", inboundID)
}

// CreateClient creates a new VLESS-Reality client on the given inbound.
// Uses POST /panel/api/inbounds/addClient with multipart/form-data.
func (c *httpClient) CreateClient(ctx context.Context, inboundID int, email string, tgId int64, comment string) (*XUIClient, error) {
	client := XUIClient{
		ID:      uuid.New().String(),
		Flow:    "xtls-rprx-vision",
		Email:   email,
		Enable:  true,
		TgId:    FlexInt64(tgId),
		Comment: comment,
	}

	if err := c.doAddClient(ctx, inboundID, client); err != nil {
		if loginErr := c.Login(ctx); loginErr == nil {
			err = c.doAddClient(ctx, inboundID, client)
		}
		if err != nil {
			return nil, err
		}
	}
	return &client, nil
}

func (c *httpClient) doAddClient(ctx context.Context, inboundID int, client XUIClient) error {
	settingsBytes, err := json.Marshal(InboundSettings{Clients: []XUIClient{client}})
	if err != nil {
		return fmt.Errorf("xui create client: marshal settings: %w", err)
	}

	body := &bytes.Buffer{}
	w := multipart.NewWriter(body)
	w.WriteField("id", strconv.Itoa(inboundID))     //nolint:errcheck
	w.WriteField("settings", string(settingsBytes)) //nolint:errcheck
	w.Close()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/panel/api/inbounds/addClient", body)
	if err != nil {
		return fmt.Errorf("xui create client: build request: %w", err)
	}
	req.Header.Set("Content-Type", w.FormDataContentType())

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("xui create client: do request: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	var result apiResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return fmt.Errorf("xui create client: decode: %w", err)
	}
	if !result.Success {
		return fmt.Errorf("xui create client: %s", result.Msg)
	}
	return nil
}

// SetClientEnabled enables or disables a client.
// Uses POST /panel/api/inbounds/updateClient/{uuid} with JSON body.
func (c *httpClient) SetClientEnabled(ctx context.Context, inboundID int, clientUUID string, enable bool) error {
	// Fetch the inbound to get the current client settings.
	inbound, err := c.GetInbound(ctx, inboundID)
	if err != nil {
		return fmt.Errorf("xui set enabled: get inbound: %w", err)
	}

	var settings InboundSettings
	if err := json.Unmarshal([]byte(inbound.Settings), &settings); err != nil {
		return fmt.Errorf("xui set enabled: parse settings: %w", err)
	}

	var target *XUIClient
	for i := range settings.Clients {
		if settings.Clients[i].ID == clientUUID {
			settings.Clients[i].Enable = enable
			target = &settings.Clients[i]
			break
		}
	}
	if target == nil {
		return fmt.Errorf("xui set enabled: client %s not found in inbound %d", clientUUID, inboundID)
	}

	// updateClient expects only the single client in settings, not the full list.
	settingsBytes, err := json.Marshal(InboundSettings{Clients: []XUIClient{*target}})
	if err != nil {
		return fmt.Errorf("xui set enabled: marshal settings: %w", err)
	}

	payload := updateClientBody{
		InboundID: inboundID,
		Settings:  string(settingsBytes),
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("xui set enabled: marshal payload: %w", err)
	}

	endpoint := fmt.Sprintf("/panel/api/inbounds/updateClient/%s", url.PathEscape(clientUUID))
	if err := c.doPost(ctx, endpoint, payloadBytes); err != nil {
		return fmt.Errorf("xui set enabled: %w", err)
	}
	return nil
}

// DeleteClient removes a client from an inbound.
// Uses POST /panel/api/inbounds/{inboundId}/delClient/{uuid}.
func (c *httpClient) DeleteClient(ctx context.Context, inboundID int, clientUUID string) error {
	endpoint := fmt.Sprintf("/panel/api/inbounds/%d/delClient/%s", inboundID, url.PathEscape(clientUUID))
	if err := c.doPost(ctx, endpoint, []byte("{}")); err != nil {
		return fmt.Errorf("xui delete client: %w", err)
	}
	return nil
}

// doGet performs an authenticated GET and returns the raw "obj" field.
// On decode failure (e.g. session expired) it re-logs in and retries once.
func (c *httpClient) doGet(ctx context.Context, path string) (json.RawMessage, error) {
	raw, err := c.doGetOnce(ctx, path)
	if err != nil {
		if loginErr := c.Login(ctx); loginErr == nil {
			raw, err = c.doGetOnce(ctx, path)
		}
	}
	return raw, err
}

func (c *httpClient) doGetOnce(ctx context.Context, path string) (json.RawMessage, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result apiResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("decode: %w", err)
	}
	if !result.Success {
		return nil, fmt.Errorf("api error: %s", result.Msg)
	}
	return result.Obj, nil
}

// doPost performs an authenticated POST with a JSON body.
// On decode failure (e.g. session expired) it re-logs in and retries once.
func (c *httpClient) doPost(ctx context.Context, path string, body []byte) error {
	err := c.doPostOnce(ctx, path, body)
	if err != nil {
		if loginErr := c.Login(ctx); loginErr == nil {
			err = c.doPostOnce(ctx, path, body)
		}
	}
	return err
}

func (c *httpClient) doPostOnce(ctx context.Context, path string, body []byte) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var result apiResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return fmt.Errorf("decode: %w", err)
	}
	if !result.Success {
		return fmt.Errorf("api error: %s", result.Msg)
	}
	return nil
}
