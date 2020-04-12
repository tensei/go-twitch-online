package twitch

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/nicklaw5/helix"
)

type onlineCallbackFunc func(stream Stream)
type offlineCallbackFunc func(channelID string)

// Stream struct containing stream information
type Stream helix.Stream

// Params struct for required values
type Params struct {
	ClientID string
	OAuth    string
}

// Client ...
type Client struct {
	streamIDs []string
	siMu      sync.RWMutex

	params *Params

	hx             *helix.Client
	stopChan       chan struct{}
	forceCheckChan chan struct{}
	interval       time.Duration

	onOnlineCallback  onlineCallbackFunc
	onOfflineCallback offlineCallbackFunc
}

// New creates a new Client
func New(params *Params) (*Client, error) {
	if params == nil {
		return nil, errors.New("missing option")
	}

	h, err := helix.NewClient(&helix.Options{
		ClientID:        params.ClientID,
		UserAccessToken: params.OAuth,
		RateLimitFunc:   rateLimitCallback,
	})

	return &Client{
		params:         params,
		hx:             h,
		stopChan:       make(chan struct{}, 1),
		forceCheckChan: make(chan struct{}, 1),
		interval:       time.Minute,
	}, err
}

// AddStreamer ...
func (c *Client) AddStreamer(userID ...string) {
	c.siMu.Lock()
	for _, uid := range userID {
		for _, si := range c.streamIDs {
			if uid == si {
				return
			}
		}
		c.streamIDs = append(c.streamIDs, uid)
	}
	c.siMu.Unlock()
}

// SetInterval set how often to check if streams are online -
// when you set the interval after .Start() you need to .Stop() and .Start()
// again for it to take effect
func (c *Client) SetInterval(d time.Duration) {
	c.interval = d
}

// CheckNow forces a online check without waiting for the next cycle
func (c *Client) CheckNow() {
	c.forceCheckChan <- struct{}{}
}

// OnOnlineCallback add the func getting called when a stream goes on
func (c *Client) OnOnlineCallback(f func(Stream)) {
	c.onOnlineCallback = f
}

// OnOfflineCallback add the func getting called when a stream goes off
func (c *Client) OnOfflineCallback(f func(string)) {
	c.onOfflineCallback = f
}

// Stop ...
func (c *Client) Stop() {
	c.stopChan <- struct{}{}
}

// Start ...
func (c *Client) Start() {
	c.check()
	tick := time.NewTicker(c.interval)
	for {
		select {
		case <-tick.C:
			c.check()
		case <-c.stopChan:
			tick.Stop()
			return
		case <-c.forceCheckChan:
			c.check()
		}
	}
}

func (c *Client) check() {
	c.siMu.RLock()
	defer c.siMu.RUnlock()

	resp, err := c.hx.GetStreams(&helix.StreamsParams{
		First:   100,
		Type:    "live",
		UserIDs: c.streamIDs,
	})
	if err != nil {
		return
	}
	streams := make(map[string]helix.Stream, len(c.streamIDs))
	for _, stream := range resp.Data.Streams {
		streams[stream.UserID] = stream
	}
	for _, s := range c.streamIDs {
		stream, ok := streams[s]
		if ok && c.onOnlineCallback != nil {
			c.onOnlineCallback(Stream(stream))
		} else if !ok && c.onOfflineCallback != nil {
			c.onOfflineCallback(s)
		}
	}
}

func rateLimitCallback(lastResponse *helix.Response) error {
	if lastResponse.GetRateLimitRemaining() > 0 {
		return nil
	}

	var reset64 int64
	reset64 = int64(lastResponse.GetRateLimitReset())

	currentTime := time.Now().Unix()

	if currentTime < reset64 {
		timeDiff := time.Duration(reset64 - currentTime)
		if timeDiff > 0 {
			fmt.Printf("Waiting on rate limit to pass before sending next request (%d seconds)\n", timeDiff)
			time.Sleep(timeDiff * time.Second)
		}
	}

	return nil
}
