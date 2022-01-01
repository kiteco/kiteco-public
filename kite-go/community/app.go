//go:generate go-bindata -pkg community -o bindata.go templates static/...

package community

import (
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"

	"github.com/kiteco/kiteco/kite-go/community/student"

	assetfs "github.com/elazarl/go-bindata-assetfs"
	"github.com/jinzhu/gorm"
	"github.com/kiteco/kiteco/kite-golib/rollbar"
	"github.com/kiteco/kiteco/kite-golib/templateset"
)

// App wraps community database operations. It contains a handle to the
// database, and managers to handle different groups of community models.
type App struct {
	DB            gorm.DB
	Auth          *UserValidation
	Users         *UserManager
	Settings      SettingsProvider
	Templates     *templateset.Set
	EmailVerifier EmailVerifier
	Signups       *SignupManager

	fileSystem http.FileSystem
}

// NewApp returns a pointer to a newly initialized App.
func NewApp(db gorm.DB, settings SettingsProvider, studentDomains *student.DomainLists) *App {
	users := NewUserManager(db, studentDomains)
	auth := NewUserValidation(users)
	signupManager := NewSignupManager(db)
	emailVerifier := newEmailVerificationManager(db)
	fs, templates := loadTemplates()

	return &App{
		DB:            db,
		Auth:          auth,
		Users:         users,
		EmailVerifier: emailVerifier,
		Signups:       signupManager,
		Settings:      settings,
		Templates:     templates,
		fileSystem:    fs,
	}
}

// Migrate sets up the proper table columns in the database.
func (a *App) Migrate() error {
	err := a.Users.Migrate()
	if err != nil {
		return fmt.Errorf("error migrating Users: %s", err)
	}

	err = a.Signups.Migrate()
	if err != nil {
		return fmt.Errorf("error migrating Signups: %s", err)
	}

	err = a.EmailVerifier.Migrate()
	if err != nil {
		return fmt.Errorf("error migrating Email Verifications: %s", err)
	}

	err = a.Settings.Load()
	if err != nil {
		return fmt.Errorf("error loading Settings: %s", err)
	}

	return nil
}

// loadTemplates is a helper that initializes the template files used by the app.
func loadTemplates() (http.FileSystem, *templateset.Set) {
	staticfs := &assetfs.AssetFS{Asset: Asset, AssetDir: AssetDir, AssetInfo: AssetInfo}
	templates := templateset.NewSet(staticfs, "templates", template.FuncMap{})
	err := templates.Validate()
	if err != nil {
		log.Fatal(err)
	}
	templates.ErrHandler = func(w io.Writer, err error) {
		rollbar.Error(err)
		if rw, ok := w.(http.ResponseWriter); ok {
			rw.WriteHeader(http.StatusInternalServerError)
		}
		err = templates.Render(w, "error.html", map[string]string{})
		if err != nil {
			w.Write([]byte("An unexpected error occurred. Please try again later."))
			rollbar.Error(err)
		}
	}
	return staticfs, templates
}
