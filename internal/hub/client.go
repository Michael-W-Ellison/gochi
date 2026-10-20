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
// slash; "https://" is added if there is no scheme. Plain HTTP is refused
// unless the hub is on this computer.
func NormalizeURL(address string) (string, error) {
	address = strings.TrimSpace(address)
	if !strings.Contains(address, "://") {
		address = "https://" + address
	}
	u, err := url.Parse(address)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return "", errors.New("enter the hub's address, such as hub.example.com or 192.168.1.10:8443")
	}
	if u.Scheme == "http" && !loopbackHost(u) {
		return "", errors.New("hubs must use https:// so account keys and saves can't be read on the way")
	}
	return strings.TrimRight(u.Scheme+"://"+u.Host+u.Path, "/"), nil
}

// NewClient returns a client for a hub. pin is the fingerprint of the
// hub's certificate to require (see ProbeCertificate), or "" to check the
// certificate as usual.
func NewClient(baseURL, token, pin string) *Client {
	return &Client{BaseURL: baseURL, Token: token, HTTP: &http.Client{Timeout: clientTimeout, Transport: transport(pin)}}
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
	if errors.Is(err, ErrCertificateChanged) {
		return ErrCertificateChanged
	}
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

// Register creates an account and returns its token and the handle the
// hub gave it
func (c *Client) Register(ctx context.Context, joinCode string) (token, handle string, err error) {
	var reply registration
	err = c.do(ctx, http.MethodPost, "/hub/v1/register", map[string]string{"join_code": joinCode}, &reply)
	return reply.Token, reply.Handle, err
}

// ListPets returns the player's pets on the hub (uploadPetData's
// counterpart for every pet)
func (c *Client) ListPets(ctx context.Context) ([]PetInfo, error) {
	var pets []PetInfo
	err := c.do(ctx, http.MethodGet, "/hub/v1/pets", nil, &pets)
	return pets, err
}

// DownloadPetData returns a pet's information and save, decrypted (see
// SealSave)
func (c *Client) DownloadPetData(ctx context.Context, key string) (PetInfo, json.RawMessage, error) {
	var reply petReply
	if err := c.do(ctx, http.MethodGet, "/hub/v1/pets/"+url.PathEscape(key), nil, &reply); err != nil {
		return reply.Info, nil, err
	}
	if reply.Save == nil {
		return reply.Info, nil, nil // Deleted
	}
	save, err := OpenSave(c.Token, key, reply.Save)
	return reply.Info, save, err
}

// UploadPetData encrypts a pet's save (see SealSave) and stores it as a
// new version, returning its version. If the hub's version isn't
// baseVersion, it returns a ConflictError.
func (c *Client) UploadPetData(ctx context.Context, info PetInfo, save json.RawMessage, baseVersion int) (int, error) {
	if !json.Valid(save) {
		return 0, errors.New("the save must be valid JSON")
	}
	sealed, err := SealSave(c.Token, info.Key, save)
	if err != nil {
		return 0, err
	}
	if len(sealed) > MaxSaveBytes {
		return 0, fmt.Errorf("the pet's save is too large to upload (%d MB at most, encrypted)", MaxSaveBytes>>20)
	}
	var reply struct {
		Version int `json:"version"`
	}
	err = c.do(ctx, http.MethodPut, "/hub/v1/pets/"+url.PathEscape(info.Key),
		putRequest{BaseVersion: baseVersion, Info: info, Save: sealed}, &reply)
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

// EnableSocialFeatures sets whether the player's pets' achievements go
// to the hub's feed
func (c *Client) EnableSocialFeatures(ctx context.Context, shareAchievements bool) (Profile, error) {
	var profile Profile
	err := c.do(ctx, http.MethodPut, "/hub/v1/profile", map[string]bool{"share_achievements": shareAchievements}, &profile)
	return profile, err
}

// ShareAchievement posts an achievement, by its ID, to the hub's feed with
// the species of the pet that earned it
func (c *Client) ShareAchievement(ctx context.Context, achievementID, species string) error {
	return c.do(ctx, http.MethodPost, "/hub/v1/achievements", shareRequest{ID: achievementID, Species: species}, nil)
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

// StartPlaydate starts a playdate with the player's pet
func (c *Client) StartPlaydate(ctx context.Context, card CardRequest) (PlaydateState, error) {
	var state PlaydateState
	err := c.do(ctx, http.MethodPost, "/hub/v1/playdates", playdateRequest{Card: &card}, &state)
	return state, err
}

// JoinPlaydate joins a playdate with the player's pet
func (c *Client) JoinPlaydate(ctx context.Context, code string, card CardRequest) (PlaydateState, error) {
	var state PlaydateState
	err := c.do(ctx, http.MethodPost, playdatePath(code, "join"), playdateRequest{Card: &card}, &state)
	return state, err
}

// UpdatePlaydate sends the player's pet (if card isn't nil) and its
// position, and returns the playdate with the events after seq
func (c *Client) UpdatePlaydate(ctx context.Context, code string, card *CardRequest, position Offset, after int64) (PlaydateState, error) {
	var state PlaydateState
	err := c.do(ctx, http.MethodPost, playdatePath(code, "update"),
		playdateRequest{Card: card, Position: position, After: after}, &state)
	return state, err
}

// ReportPlay reports that the player's pet played with another member's
// pet; the hub writes what the members see
func (c *Client) ReportPlay(ctx context.Context, code string, report PlayReport) (PlaydateEvent, error) {
	var event PlaydateEvent
	err := c.do(ctx, http.MethodPost, playdatePath(code, "play"), report, &event)
	return event, err
}

// LeavePlaydate leaves a playdate
func (c *Client) LeavePlaydate(ctx context.Context, code string) error {
	return c.do(ctx, http.MethodPost, playdatePath(code, "leave"), nil, nil)
}
