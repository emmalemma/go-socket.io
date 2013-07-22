package socketio

import (
	"code.google.com/p/go.net/websocket"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

const (
	ProtocolVersion = 1
)

type Client struct {
	*Session
	*EventEmitter
}

func Dial(url_, origin string) (*Client, error) {
	u, err := url.Parse(url_)
	if err != nil {
		return nil, err
	}
	path := u.Path
	if l := len(path); l > 0 && path[len(path)-1] == '/' {
		path = path[:l-1]
	}
	lastPath := strings.LastIndex(path, "/")
	endpoint := ""
	if lastPath >= 0 {
		path := path[lastPath:]
		if len(path) > 0 {
			endpoint = path
		}
	}
	u.Path = ""

	url_ = fmt.Sprintf("%s/socket.io/%d/", u.String(), ProtocolVersion)
	r, err := http.Get(url_)
	if err != nil {
		return nil, err
	}
	defer r.Body.Close()
	if r.StatusCode != 200 {
		return nil, errors.New("invalid status: " + r.Status)
	}
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}
	parts := strings.SplitN(string(body), ":", 4)
	if len(parts) != 4 {
		return nil, errors.New("invalid handshake: " + string(body))
	}
	if !strings.Contains(parts[3], "websocket") {
		return nil, errors.New("server does not support websockets")
	}
	sessionId := parts[0]
	wsurl := "ws" + url_[4:]
	wsurl = fmt.Sprintf("%swebsocket/%s", wsurl, sessionId)
	ws, err := websocket.Dial(wsurl, "", origin)
	if err != nil {
		return nil, err
	}

	timeout, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return nil, err
	}

	ee := NewEventEmitter()
	session := NewSession(map[string]*EventEmitter{endpoint: ee}, sessionId, int(timeout), false)
	transport := newWebSocket(session)
	transport.conn = ws
	session.transport = transport
	if endpoint != "" {
		session.Of(endpoint).sendPacket(new(connectPacket))
	}

	return &Client{
		Session:      session,
		EventEmitter: ee,
	}, nil
}

func (c *Client) Run() {
	c.Session.loop()
}