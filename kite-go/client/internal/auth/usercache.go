package auth

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/kiteco/kiteco/kite-go/community"
)

// cacheUser caches a user object at the file indicated by UserCacheFile
func (c *Client) cacheUser(usr *community.User) error {
	c.userCacheMutex.Lock()
	defer c.userCacheMutex.Unlock()

	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(usr); err != nil {
		return fmt.Errorf("error encoding for user caching: %v", err)
	}

	if err := ioutil.WriteFile(c.userCacheFile, buf.Bytes(), os.ModePerm); err != nil {
		return fmt.Errorf("error caching user to file: %v", err)
	}

	return nil
}

func (c *Client) removeCachedUser() error {
	return os.RemoveAll(c.userCacheFile)
}

// getCachedUser reads from file a user object cached at UserCacheFile
func (c *Client) getCachedUser() (*community.User, error) {
	c.userCacheMutex.RLock()
	defer c.userCacheMutex.RUnlock()

	if _, err := os.Stat(c.userCacheFile); os.IsNotExist(err) {
		return nil, err
	}

	usrBytes, err := ioutil.ReadFile(c.userCacheFile)
	if err != nil {
		return nil, err
	}

	var usr community.User
	err = json.NewDecoder(bytes.NewBuffer(usrBytes)).Decode(&usr)
	if err != nil {
		return nil, err
	}

	if usr.ID == 0 {
		return nil, fmt.Errorf("invalid user")
	}

	return &usr, nil
}
