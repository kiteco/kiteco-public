package curation

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/sessions"
	"github.com/kiteco/kiteco/kite-go/web/authentication"
	gorp "gopkg.in/gorp.v1"
)

const (
	passwordCost         = 10
	preSalt       string = "XXXXXXX" // changing this will invalidate all passwords in the DB
	postSalt      string = "/$`*tgihtg45r89tgXXXXXXX" // changing this will invalidate all passwords in the DB
	sessionSecret string = "XXXXXXX" // changing this will invalidate all active sessions
	sessionName   string = "curation-auth"
)

// User represents a username and password that give access to the web ranking system.
type User struct {
	ID           int64
	Name         string
	PasswordHash string
}

// Authenticator represents a connection to the user database and a session manager that together perform user authentication
type Authenticator struct {
	db    *gorp.DbMap
	store *sessions.CookieStore
}

// NewAuthenticatorOrDie connects to the user DB and initializes a session manager. It panics if the database connection fails.
func NewAuthenticatorOrDie(db *sql.DB, dialect gorp.Dialect) *Authenticator {
	dbmap, err := OpenUserDb(db, dialect)
	if err != nil {
		log.Fatalf("Failed to connect to user db: %v", err)
	}
	return &Authenticator{
		db:    dbmap,
		store: sessions.NewCookieStore([]byte(sessionSecret)),
	}
}

// OpenUserDb opens a database, constructs a db map, and creates any tables that do
// not already exist.
func OpenUserDb(db *sql.DB, dialect gorp.Dialect) (*gorp.DbMap, error) {
	// Set up dialect
	dbmap := gorp.DbMap{
		Db:      db,
		Dialect: dialect,
	}

	// Register tables
	dbmap.AddTable(User{}).SetKeys(true, "ID")

	return &dbmap, nil
}

// ErrNotLoggedIn signifies that there is no user logged in.
var ErrNotLoggedIn = errors.New("user not logged in")

// ErrSessionNotDecodable signifies that there was a problem retrieving the session.
var ErrSessionNotDecodable = errors.New("session could not be decoded")

// CurrentUser gets the name of the user currently logged in (according to cookies in the given request) or
// returns empty string if no user is logged in.
func (auth *Authenticator) CurrentUser(r *http.Request) (string, error) {
	session, err := auth.store.Get(r, sessionName)
	if err != nil {
		return "", ErrSessionNotDecodable
	}
	if username, ok := session.Values["username"].(string); ok {
		return username, nil
	}
	return "", ErrNotLoggedIn
}

// CreateUser creates a user in the db with the given username and password.
// The password is hashed. Used for testing at the moment.
func (auth *Authenticator) CreateUser(username, password string) error {
	if username == "" {
		return errors.New("Missing username")
	}
	if password == "" {
		return errors.New("Missing password")
	}

	hash, err := authentication.PasswordHash(password)
	if err != nil {
		fmt.Println("Failed to hash password: ", err)
		return err
	}

	user := User{Name: username, PasswordHash: hash}
	err = auth.db.Insert(&user)
	if err != nil {
		fmt.Println("Failed to insert user: ", err)
		return err
	}

	return nil
}

// Authenticate checks if the given username exists in the user database and, if it does,
// whether the given password is correct.
func (auth *Authenticator) Authenticate(username, password string) (string, error) {
	if username == "" {
		return "", errors.New("Missing username")
	}
	if password == "" {
		return username, errors.New("Missing password")
	}

	// Look up user
	var user User
	err := auth.db.SelectOne(&user, "SELECT * FROM User WHERE name = ?", username)
	if err == sql.ErrNoRows {
		return username, fmt.Errorf("No such user: %v", err)
	} else if err != nil {
		return username, err
	}

	// Validate password
	if !authentication.PasswordMatches(user.PasswordHash, password) {
		return username, errors.New("Incorrect password")
	}

	return user.Name, nil
}

// RedirectIfNotAuthenticated executes an HTTP redirect if no user is logged in
func (auth *Authenticator) RedirectIfNotAuthenticated(w http.ResponseWriter, r *http.Request, url string) error {
	if _, err := auth.CurrentUser(r); err == ErrNotLoggedIn {
		log.Printf("Received an unauthenticated request to %s, redirecting to %s\n", r.URL.Path, url)
		http.Redirect(w, r, url, http.StatusTemporaryRedirect)
		return err
	} else if err != nil {
		return err
	}
	return nil
}

// ErrorIfNotAuthenticated writes an error the given response stream if no user is logged in
func (auth *Authenticator) ErrorIfNotAuthenticated(w http.ResponseWriter, r *http.Request) error {
	if _, err := auth.CurrentUser(r); err == ErrNotLoggedIn {
		log.Printf("Received an unauthenticated request to %s, returning error\n", r.URL.Path)
		http.Error(w, "you must be logged in", http.StatusUnauthorized)
		return err
	} else if err != nil {
		return err
	}
	return nil
}

// Login attempts to authenticate a user and sets a session cookie if successful
func (auth *Authenticator) Login(username, password string, w http.ResponseWriter, r *http.Request) error {
	username, err := auth.Authenticate(username, password)
	if err != nil {
		log.Printf("user '%s' attempted to log in but failed: %s", username, err.Error())
		return err
	}

	session, err := auth.store.Get(r, sessionName)
	if err != nil {
		return err
	}

	session.Values["username"] = username
	if err = session.Save(r, w); err != nil {
		return err
	}

	log.Println(username, "logged in")
	return nil
}

// Logout clears the authentication cookie for the given user if it exists
func (auth *Authenticator) Logout(w http.ResponseWriter, r *http.Request) error {
	session, err := auth.store.Get(r, sessionName)
	if err != nil {
		return err
	}

	// Causes the client to delete the cookie immediately
	session.Options.MaxAge = -1
	return session.Save(r, w)
}

// --

// CreateTablesIfNotExists is used for testing.
func (auth *Authenticator) CreateTablesIfNotExists() error {
	return auth.db.CreateTablesIfNotExists()
}
