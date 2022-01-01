package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/kiteco/kiteco/kite-go/community"
	"github.com/kiteco/kiteco/kite-golib/errors"
)

func (c *Client) handleAccountAuth(w http.ResponseWriter, r *http.Request, urlPath string) {
	ctx, cancel := context.WithTimeout(r.Context(), c.httpTimeout)
	defer cancel()

	err := r.ParseMultipartForm(1024)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	resp, err := c.postForm(ctx, urlPath, r.Form)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		w.Header().Set("Content-Type", resp.Header.Get("Content-Type"))
		w.WriteHeader(resp.StatusCode)
		io.Copy(w, resp.Body)
		return
	}

	c.token.UpdateFromHeader(resp.Header)
	err = c.saveAuth()
	if err != nil {
		log.Println(err)
	}

	w.Header().Set("Content-Type", resp.Header.Get("Content-Type"))
	w.WriteHeader(resp.StatusCode)
	bodyCopy := io.TeeReader(resp.Body, w)

	var user community.User
	if err := json.NewDecoder(bodyCopy).Decode(&user); err != nil {
		log.Printf("ERR decoding user from %s response: %v\n", urlPath, err)
		return
	}

	//the current user will be set by LoggedIn (component UserAuth)
	//we don't want more than one place where a login is handled
	c.loggedInChan <- &user
}

func (c *Client) handleLogin(w http.ResponseWriter, r *http.Request) {
	c.handleAccountAuth(w, r, "/api/account/login-desktop")
}

func (c *Client) handleCreateAccount(w http.ResponseWriter, r *http.Request) {
	c.handleAccountAuth(w, r, "/api/account/create-web")
}

func (c *Client) handleCreatePasswordlessAccount(w http.ResponseWriter, r *http.Request) {
	c.handleAccountAuth(w, r, "/api/account/createPasswordless")
}

func (c *Client) handleAuthenticate(w http.ResponseWriter, r *http.Request) {
	c.handleAccountAuth(w, r, "/api/account/authenticate")
}

// writes the information about the current user into the response
// returns status 401 if no user is logged in
func (c *Client) handleUser(w http.ResponseWriter, r *http.Request) {
	user, err := c.GetUser()
	if err != nil {
		http.Error(w, "no user is currently logged in", http.StatusUnauthorized)
		return
	}

	buf, err := json.Marshal(user)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(buf)
}

func (c *Client) handleLogout(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), c.httpTimeout)
	defer cancel()

	resp, err := c.Get(ctx, "/api/account/logout")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		c.token.UpdateFromHeader(resp.Header)
	}

	c.Logout()

	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

// Logout clears data related to the current user
func (c *Client) Logout() error {
	c.token.Clear()
	jar := newCookieJar()
	if jar == nil {
		return errors.New("error creating cookiejar")
	}

	//remove the current user
	func() {
		c.mu.Lock()
		defer c.mu.Unlock()

		c.client.Jar = jar
		c.user = nil

		// clear the license store and plan
		if c.licenseStore == nil {
			return
		}

		c.licenseStore.ClearUser()
		if err := c.licenseStore.SaveFile(c.licenseFilepath); err != nil {
			log.Println(err)
		}

		c.resetRemoteListenerLocked()
	}()

	if err := c.saveAuth(); err != nil {
		log.Println(err)
	}

	//this is handled in the loginLoop and triggers LoggedOut
	c.loggedOutChan <- struct{}{}

	return nil
}

// FetchUser fetches the remote user using the auth cookie
func (c *Client) FetchUser(ctx context.Context) (*community.User, error) {
	if !c.HasAuthCookie() {
		return nil, fmt.Errorf("no authentication set")
	}

	ctx, cancel := context.WithTimeout(ctx, c.httpTimeout)
	defer cancel()

	user, _, err := c.fetchUserRemote(ctx)
	if err != nil {
		return nil, err
	}

	return user, nil
}

// CachedUser returns the cached user object if it exists
func (c *Client) CachedUser() (*community.User, error) {
	user, _, err := c.fetchUserLocal()
	return user, err
}

// --

func (c *Client) fetchUserLocal() (*community.User, int, error) {
	var user *community.User
	user, err := c.getCachedUser()
	if err != nil {
		return nil, http.StatusUnauthorized, ErrNotAuthenticated
	}

	return user, http.StatusOK, nil
}

func (c *Client) fetchUserRemote(ctx context.Context) (*community.User, int, error) {
	resp, err := c.getNoHMAC(ctx, "/api/account/user")
	if err != nil {
		return nil, -1, fmt.Errorf("error accessing /api/account/user: %s", err)
	}
	defer resp.Body.Close()

	switch {
	case resp.StatusCode == http.StatusOK:
		var user community.User
		if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
			return nil, resp.StatusCode, fmt.Errorf("error unmarshaling user from %s: %v", "/api/account/user", err)
		}
		return &user, resp.StatusCode, nil
	case resp.StatusCode == http.StatusUnauthorized:
		// Only return ErrNotAuthenticated if we get http.StatusUnauthorized
		return nil, resp.StatusCode, ErrNotAuthenticated
	default:
		return nil, resp.StatusCode, fmt.Errorf("got response code %d checking for sessioned user", resp.StatusCode)
	}
}

func (c *Client) checkAuthenticated(ctx context.Context) error {
	if !c.HasAuthCookie() {
		return ErrNotAuthenticated
	}

	ctx, cancel := context.WithTimeout(ctx, c.httpTimeout)
	defer cancel()
	resp, err := c.getNoHMAC(ctx, "/api/account/authenticated")
	if err != nil {
		return fmt.Errorf("error accessing /api/account/authenticated: %s", err)
	}

	defer resp.Body.Close()

	_, err = io.Copy(ioutil.Discard, resp.Body)
	if err != nil {
		return fmt.Errorf("error reading response from /api/account/authenticated: %s", err)
	}

	switch {
	case resp.StatusCode == http.StatusOK:
		return nil
	case resp.StatusCode == http.StatusUnauthorized:
		// Only return ErrNotAuthenticated if we get http.StatusUnauthorized
		return ErrNotAuthenticated
	default:
		return fmt.Errorf("got response code %d checking for authentication", resp.StatusCode)
	}
}
