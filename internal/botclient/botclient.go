// Package botclient is the client side of Drift's API for programs that act
// as people: the simulator and the load tester. It signs guests up, sets
// their settings and finds the socket address, the way the web app does.
package botclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

var httpClient = &http.Client{Timeout: 15 * time.Second}

// Guest is the card a bot signs up with.
type Guest struct {
	Name     string   `json:"name"`
	Gender   string   `json:"gender"`
	Age      int      `json:"age"`
	Tags     []string `json:"tags"`
	About    string   `json:"about"`
	Location string   `json:"location"`
}

// SignUp makes a guest card and returns its session token.
func SignUp(ctx context.Context, server string, g Guest) (string, error) {
	return SignUpFrom(ctx, server, "", g)
}

// SignUpFrom signs up as if from clientIP, sent as X-Forwarded-For. Only
// a server started with BEHIND_PROXY=true believes it; load tests use it to
// stand in for many people without tripping the per-IP sign-up limit.
func SignUpFrom(ctx context.Context, server, clientIP string, g Guest) (string, error) {
	var out struct {
		Token string `json:"token"`
	}
	if err := call(ctx, http.MethodPost, server+"/auth/guest", clientIP, "", g, http.StatusCreated, &out); err != nil {
		return "", fmt.Errorf("sign up %s: %w", g.Name, err)
	}
	return out.Token, nil
}

// SetAura picks a glow, as a person does in the card editor.
func SetAura(ctx context.Context, server, token, aura string) error {
	if err := call(ctx, http.MethodPatch, server+"/me", "", token, map[string]string{"aura": aura}, http.StatusOK, nil); err != nil {
		return fmt.Errorf("set aura: %w", err)
	}
	return nil
}

// SocketURL is where a session's websocket connects.
func SocketURL(server, token string) (string, error) {
	u, err := url.Parse(server)
	if err != nil {
		return "", err
	}
	u.Scheme = strings.Replace(u.Scheme, "http", "ws", 1)
	u.Path = "/ws"
	u.RawQuery = url.Values{"token": {token}}.Encode()
	return u.String(), nil
}

func call(ctx context.Context, method, target, clientIP, token string, in any, want int, out any) error {
	body, err := json.Marshal(in)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, method, target, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if clientIP != "" {
		req.Header.Set("X-Forwarded-For", clientIP)
	}
	res, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = res.Body.Close() }()
	if res.StatusCode != want {
		var e struct {
			Error string `json:"error"`
		}
		_ = json.NewDecoder(res.Body).Decode(&e)
		return fmt.Errorf("%s: %s", res.Status, e.Error)
	}
	if out == nil {
		return nil
	}
	return json.NewDecoder(res.Body).Decode(out)
}

// ReadFrames reads one websocket message and returns the frames in it. The
// server joins frames that queued up for a client into one message, one
// per line, so a message holds one or more.
func ReadFrames(conn *websocket.Conn) ([][]byte, error) {
	_, msg, err := conn.ReadMessage()
	if err != nil {
		return nil, err
	}
	return bytes.Split(msg, []byte{'\n'}), nil
}
