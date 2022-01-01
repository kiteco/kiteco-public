package auth

import (
	"log"
	"os"
	"testing"
	"time"

	"github.com/kiteco/kiteco/kite-go/community"
	"github.com/stretchr/testify/assert"
)

func Test_CacheUser(t *testing.T) {
	client := NewTestClient(5 * time.Second)
	defer os.Remove(client.userCacheFile)

	user1 := &community.User{
		ID:    123,
		Name:  "test",
		Email: "test@test.com",
	}

	//CacheUser
	err := client.cacheUser(user1)
	assert.NoError(t, err)

	user2, err := client.getCachedUser()
	assert.NoError(t, err)
	assert.Equal(t, user1.ID, user2.ID)
	assert.Equal(t, user1.Name, user2.Name)
	assert.Equal(t, user1.Email, user2.Email)
}

func Test_CacheUser_NotPresent(t *testing.T) {
	client := NewTestClient(5 * time.Second)
	defer os.Remove(client.userCacheFile)

	_, err := client.getCachedUser()
	assert.Error(t, err)
}

func Test_NilUserCache(t *testing.T) {
	client := NewTestClient(5 * time.Second)
	defer os.Remove(client.userCacheFile)

	err := client.cacheUser(nil)
	assert.NoError(t, err)

	user, err := client.getCachedUser()
	log.Println(user)
	assert.Error(t, err)
}
