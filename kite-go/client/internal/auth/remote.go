package auth

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/kiteco/kiteco/kite-go/client/component"
	"github.com/kiteco/kiteco/kite-golib/domains"
	"github.com/kiteco/kiteco/kite-golib/remotectrl"
)

var rcKiteTarget = "https://" + domains.RemoteConfig

func (c *Client) initializeRemote(opts component.InitializerOptions) {
	if opts.Notifs == nil {
		return
	}

	logFilePath := filepath.Join(opts.Platform.KiteRoot, "message.log")

	// read message set from disk
	func() {
		f, err := os.Open(logFilePath)
		if err != nil {
			// file must not exist
			return
		}
		defer f.Close()
		lines := bufio.NewScanner(f)
		for lines.Scan() {
			c.remoteMsgSet.Store(strings.TrimSpace(lines.Text()), nil)
		}
	}()

	// create logger to log newly seen messages to disk
	var logFile io.Writer
	logFile, err := os.OpenFile(logFilePath, os.O_APPEND|os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		logFile = ioutil.Discard
	}
	c.remoteMsgLog = log.New(logFile, "", 0)
	c.remoteHandlers = append(c.remoteHandlers, opts.Notifs)
	c.remoteHandlers = append(c.remoteHandlers, opts.Settings)
	c.remoteHandlers = append(c.remoteHandlers, opts.Cohort)
}

func (c *Client) handleRemoteMessage(msg remotectrl.Message) {
	if _, seen := c.remoteMsgSet.LoadOrStore(msg.ID, nil); seen {
		return
	}
	c.remoteMsgLog.Println(msg.ID)

	for _, h := range c.remoteHandlers {
		if err := h.HandleRemoteMessage(msg); err != nil {
			log.Println(err)
		}
	}
}

func (c *Client) resetRemoteListenerLocked() {
	if !c.hasNonNilRemoteHandler() {
		return
	}

	newChannel := c.platform.InstallID
	if c.user != nil {
		newChannel = c.user.IDString()
	}
	if c.remoteChannel == newChannel {
		return
	}

	if c.remoteListener != nil {
		c.remoteListener.Close()
	}

	c.remoteChannel = newChannel
	u, err := url.Parse(rcKiteTarget)
	if err != nil {
		panic("invalid URL")
	}
	u.Path = fmt.Sprintf("/receive/%s", newChannel)

	// make a copy of client with no timeout, as that's incompatible with websockets
	client := *c.client
	client.Timeout = 0
	c.remoteListener = remotectrl.Listen(u.String(), &client, c.handleRemoteMessage)
}

func (c *Client) hasNonNilRemoteHandler() bool {
	for _, h := range c.remoteHandlers {
		if h != nil {
			return true
		}
	}
	return false
}
