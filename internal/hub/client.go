package hub

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// Client talks to a hub for one player
type Client struct {
	BaseURL string // Such as "https://hub.example.com"
	Token   string
	HTTP    *http.Client
}

// clientTimeout is how long a request to the hub may take
const clientTimeout = 20 * time.Second

// ConflictError is returned when the hub has another version of a pet
type ConflictError struct {
	Version int // The hub's version
}

func (e ConflictError) Error() string {
	return fmt.Sprintf("the hub has version %d of this pet", e.Version)
}

// NormalizeURL checks a hub address and returns it without a trailing
// slash; "http://" is added if there is no scheme
func NormalizeURL(address string) (string, error) {
	address = strings.TrimSpace(address)
	if !strings.Contains(address, "://") {
		address = "http://" + address
	}
	u, err := url.Parse(address)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return "", errors.New("enter the hub's address, such as https://hub.example.com or 192.168.1.10:8080")
	}
	return strings.TrimRight(u.Scheme+"://"+u.Host+u.Path, "/"), nil
}

// NewClient returns a client for a hub
func NewClient(baseURL, token string) *Client {
	return &Client{BaseURL: baseURL, Token: token, HTTP: &http.Client{Timeout: clientTimeout}}
}

// do sends a request and decodes a JSON reply into out (if not nil)
func (c *Client) do(ctx context.Context, method, path string, body, out interface{}) error {
	var reader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(data)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.BaseURL+path, reader)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return fmt.Errorf("could not reach the hub: %w", err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, MaxSaveBytes+1<<20))
	if err != nil {
		return err
	}

	if resp.StatusCode >= 300 {
		var e errorReply
		json.Unmarshal(data, &e)
		switch {
		case resp.StatusCode == http.StatusConflict && e.Version > 0:
			return ConflictError{Version: e.Version}
		case resp.StatusCode == http.StatusUnauthorized && e.Error == ErrUnauthorized.Error():
			return ErrUnauthorized
		case resp.StatusCode == http.StatusNotFound && e.Error == ErrNotFound.Error():
			return ErrNotFound
		case resp.StatusCode == http.StatusNotFound && e.Error == ErrNoPlaydate.Error():
			return ErrNoPlaydate
		case resp.StatusCode == http.StatusNotFound && e.Error == ErrNotInPlaydate.Error():
			return ErrNotInPlaydate
		case e.Error != "":
			return errors.New(e.Error)
		}
		return fmt.Errorf("the hub answered %s", resp.Status)
	}
	if out != nil {
		if err := json.Unmarshal(data, out); err != nil {
			return fmt.Errorf("unexpected reply from the hub: %w", err)
		}
	}
	return nil
}

// Status checks that the address is a Gochi hub
func (c *Client) Status(ctx context.Context) (Status, error) {
	var status Status
	if err := c.do(ctx, http.MethodGet, "/hub/v1/status", nil, &status); err != nil {
		return status, err
	}
	if status.Hub != "gochi" {
		return status, errors.New("that address isn't a Gochi hub")
	}
	return status, nil
}

// Register creates an account and returns its token
func (c *Client) Register(ctx context.Context, username, joinCode string) (string, error) {
	var reply struct {
		Token string `json:"token"`
	}
	err := c.do(ctx, http.MethodPost, "/hub/v1/register", map[string]string{"username": username, "join_code": joinCode}, &reply)
	return reply.Token, err
}

// ListPets returns the player's pets on the hub (uploadPetData's
// counterpart for every pet)
func (c *Client) ListPets(ctx context.Context) ([]PetInfo, error) {
	var pets []PetInfo
	err := c.do(ctx, http.MethodGet, "/hub/v1/pets", nil, &pets)
	return pets, err
}

// DownloadPetData returns a pet's information and save
func (c *Client) DownloadPetData(ctx context.Context, key string) (PetInfo, json.RawMessage, error) {
	var reply petReply
	err := c.do(ctx, http.MethodGet, "/hub/v1/pets/"+url.PathEscape(key), nil, &reply)
	return reply.Info, reply.Save, err
}

// UploadPetData stores a new version of a pet and returns its version. If
// the hub's version isn't baseVersion, it returns a ConflictError.
func (c *Client) UploadPetData(ctx context.Context, info PetInfo, save json.RawMessage, baseVersion int) (int, error) {
	var reply struct {
		Version int `json:"version"`
	}
	err := c.do(ctx, http.MethodPut, "/hub/v1/pets/"+url.PathEscape(info.Key),
		putRequest{BaseVersion: baseVersion, Info: info, Save: save}, &reply)
	return reply.Version, err
}

// DeletePetData marks a pet deleted on the hub
func (c *Client) DeletePetData(ctx context.Context, key string, baseVersion int) (int, error) {
	var reply struct {
		Version int `json:"version"`
	}
	err := c.do(ctx, http.MethodDelete, "/hub/v1/pets/"+url.PathEscape(key)+"?base_version="+strconv.Itoa(baseVersion), nil, &reply)
	return reply.Version, err
}

// GetProfile returns the player's social settings
func (c *Client) GetProfile(ctx context.Context) (Profile, error) {
	var profile Profile
	err := c.do(ctx, http.MethodGet, "/hub/v1/profile", nil, &profile)
	return profile, err
}

// EnableSocialFeatures sets the player's area and whether their pets appear
// to nearby players and they can share achievements
func (c *Client) EnableSocialFeatures(ctx context.Context, area string, sharing bool) (Profile, error) {
	var profile Profile
	err := c.do(ctx, http.MethodPut, "/hub/v1/profile", map[string]interface{}{"area": area, "sharing": sharing}, &profile)
	return profile, err
}

// GetNearbyPets returns other players' pets in the player's area
func (c *Client) GetNearbyPets(ctx context.Context) ([]NearbyPet, error) {
	var pets []NearbyPet
	err := c.do(ctx, http.MethodGet, "/hub/v1/nearby", nil, &pets)
	return pets, err
}

// ShareAchievement posts an achievement to the hub's feed
func (c *Client) ShareAchievement(ctx context.Context, a SharedAchievement) error {
	return c.do(ctx, http.MethodPost, "/hub/v1/achievements", a, nil)
}

// Feed returns recently shared achievements, newest first
func (c *Client) Feed(ctx context.Context) ([]SharedAchievement, error) {
	var feed []SharedAchievement
	err := c.do(ctx, http.MethodGet, "/hub/v1/achievements", nil, &feed)
	return feed, err
}

// playdatePath returns the path of a playdate request
func playdatePath(code, action string) string {
	return "/hub/v1/playdates/" + url.PathEscape(NormalizePlaydateCode(code)) + "/" + action
}

// StartPlaydate starts a playdate with the player's pet (its card)
func (c *Client) StartPlaydate(ctx context.Context, card json.RawMessage) (PlaydateState, error) {
	var state PlaydateState
	err := c.do(ctx, http.MethodPost, "/hub/v1/playdates", playdateRequest{Card: card}, &state)
	return state, err
}

// JoinPlaydate joins a playdate with the player's pet
func (c *Client) JoinPlaydate(ctx context.Context, code string, card json.RawMessage) (PlaydateState, error) {
	var state PlaydateState
	err := c.do(ctx, http.MethodPost, playdatePath(code, "join"), playdateRequest{Card: card}, &state)
	return state, err
}

// UpdatePlaydate sends the player's pet (its card, if not nil) and its
// position, and returns the playdate with the events after seq
func (c *Client) UpdatePlaydate(ctx context.Context, code string, card json.RawMessage, position Offset, after int64) (PlaydateState, error) {
	var state PlaydateState
	err := c.do(ctx, http.MethodPost, playdatePath(code, "update"),
		playdateRequest{Card: card, Position: position, After: after}, &state)
	return state, err
}

// PostPlaydateEvent adds an event to a playdate
func (c *Client) PostPlaydateEvent(ctx context.Context, code string, e PlaydateEvent) (PlaydateEvent, error) {
	var event PlaydateEvent
	err := c.do(ctx, http.MethodPost, playdatePath(code, "events"), e, &event)
	return event, err
}

// LeavePlaydate leaves a playdate
func (c *Client) LeavePlaydate(ctx context.Context, code string) error {
	return c.do(ctx, http.MethodPost, playdatePath(code, "leave"), struct{}{}, nil)
}
