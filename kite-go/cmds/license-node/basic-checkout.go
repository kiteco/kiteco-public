//go:generate bash -c "go-bindata $BINDATAFLAGS -o bindata.go templates/..."

package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/kiteco/kiteco/kite-golib/errors"

	"github.com/kiteco/kiteco/kite-go/community"

	stripekite "github.com/kiteco/kiteco/kite-golib/stripe"

	assetfs "github.com/elazarl/go-bindata-assetfs"

	"github.com/kiteco/kiteco/kite-golib/templateset"

	"github.com/gorilla/mux"
)

var (
	currentUserID       = int64(0)
	currentUserStripeID = ""
	currentUserMail     = ""
)

type data struct {
	Message     string
	CurrentUser string
}

type checkoutApp struct {
	templates *templateset.Set
	users     *community.UserManager
	plans     stripekite.Plans
}

func buildApp(plans stripekite.Plans, users *community.UserManager) checkoutApp {
	staticfs := &assetfs.AssetFS{Asset: Asset, AssetDir: AssetDir, AssetInfo: AssetInfo}

	return checkoutApp{
		templates: templateset.NewSet(staticfs, "templates", nil),
		users:     users,
		plans:     plans,
	}
}

func (a checkoutApp) handleSuccess(w http.ResponseWriter, r *http.Request) {

	a.renderRootTemplate(w, "Congrats, you successfully paid for Kite PRO")
}

func (a checkoutApp) handleCancel(w http.ResponseWriter, r *http.Request) {
	a.renderRootTemplate(w, "Checkout operation got cancelled")
}

func (a checkoutApp) handleRoot(w http.ResponseWriter, r *http.Request) {
	a.renderRootTemplate(w, "")
}

func (a checkoutApp) renderRootTemplate(w http.ResponseWriter, message string) {

	err := a.templates.Render(w, "root.html",
		data{
			CurrentUser: getUserInfo(),
			Message:     message,
		})
	if err != nil {
		log.Println(err)
	}
}

func getUserInfo() string {
	if currentUserMail == "" {
		return "Anonymous user"
	}
	if currentUserStripeID == "" {
		return fmt.Sprintf("User %s (no stripe account)", currentUserMail)
	}
	return fmt.Sprintf("User %s (stripe ID : %s)", currentUserMail, currentUserStripeID)
}

func (a checkoutApp) login(email string, w http.ResponseWriter, r *http.Request) error {
	_, session, err := a.users.Login(email, testPassword)

	if err != nil {
		return err
	}
	community.SetSession(session, w, r)
	return nil
}

func (a checkoutApp) handleSelectUser(w http.ResponseWriter, r *http.Request) {
	user, ok := r.URL.Query()["user"]
	if !ok {
		a.renderRootTemplate(w, fmt.Sprint("Please specify user when using /selectUser endpoint"))
		return
	}
	var err error
	switch user[0] {
	case "userA":
		err = a.login(userAEmail, w, r)

	case "userB":
		err = a.login(userBEmail, w, r)
	case "userStudent":
		err = a.login(studentEmail, w, r)
	default:
		err = errors.Errorf("%s doesn't corresponds to a known test user", user[0])
	}
	if err != nil {
		a.renderRootTemplate(w, fmt.Sprintf("Error while login the user : %v", err))
	} else {
		a.renderRootTemplate(w, fmt.Sprintf("%s User selected", user[0]))
	}
}

func (a checkoutApp) setupRoutes(r *mux.Router) {

	r.HandleFunc("/", a.handleRoot)
	r.HandleFunc("/success", a.handleSuccess)
	r.HandleFunc("/cancel", a.handleCancel)
	r.HandleFunc("/selectUser", a.handleSelectUser)
}
