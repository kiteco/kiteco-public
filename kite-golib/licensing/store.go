package licensing

import (
	"encoding/json"
	"io"
	"os"
	"time"

	"github.com/kiteco/kiteco/kite-golib/errors"
)

type persistentData struct {
	// UserID is the ID of the currently managed user
	UserID string `json:"user"`
	// InstallID is the ID of the currently managed installation
	InstallID string `json:"install"`
	// HasHadPro is true if at least one valid pro license was added to this store for any user
	HasHadPro bool `json:"has_had_pro"`
	// LicenseTokens is the list of licenses, which belong to the current user
	Tokens []string `json:"licenses"`
}

// ProductGetter has a method to return the current product.
type ProductGetter interface {
	GetProduct() Product
}

// StatusGetter has a method to return the
// - Current license expiration date
// - Current license end of subscription date
// - Current license plan
// - Current license product
type StatusGetter interface {
	LicenseStatus() (time.Time, time.Time, Plan, Product)
}

// TrialAvailableGetter has a method to return whether the user can trial
type TrialAvailableGetter interface {
	TrialAvailable() bool
}

// Store manages a set of licenses of a single user
// This is suitable for client-side use, i.e. for a single user with 0..n licenses
// Store isn't go-routine safe, users need to take care of proper synchronization.
type Store struct {
	KiteServer bool

	l Licenses

	validator *Validator

	// mutable data
	hasHadPro bool
	userID    string
	installID string
}

// NewStore creates a license store.
func NewStore(validator *Validator, installID string) *Store {
	return &Store{
		validator: validator,
		installID: installID,
	}
}

// LoadFile loads the license data from the given file
func (s *Store) LoadFile(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}

	defer file.Close()
	return s.Load(file)
}

// SaveFile stores the current license data into the given file
// If the file doesn't exist yet, then it's created.
func (s *Store) SaveFile(path string) error {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}

	defer file.Close()
	return s.Save(file)
}

// Load initializes the store from an input source
// The format of Save is expected. The input is closed after the operation completed.
func (s *Store) Load(r io.Reader) error {
	s.ClearAll()

	var data persistentData
	if err := json.NewDecoder(r).Decode(&data); err != nil {
		return err
	}

	s.userID = data.UserID
	s.installID = data.InstallID
	s.hasHadPro = data.HasHadPro
	for _, tok := range data.Tokens {
		// ignore the error, since we should just filter out invalid tokens at this point
		s.Add(tok)
	}

	return nil
}

// Save stores the current data of the store into the writer, e.g. a file on disk
// An error is returned when an error occurred while writing the data.
func (s *Store) Save(w io.Writer) error {
	var tokens []string
	for lic, next := s.l.Iterate(); lic != nil; lic = next() {
		tokens = append(tokens, lic.Token)
	}
	return json.NewEncoder(w).Encode(&persistentData{
		InstallID: s.installID,
		UserID:    s.userID,
		HasHadPro: s.hasHadPro,
		Tokens:    tokens,
	})
}

// UserID returns the ID of the currently managed user
func (s *Store) UserID() string {
	return s.userID
}

// SetUserID sets the user ID, and removes all licenses not matching the ID or the installID.
func (s *Store) SetUserID(userID string) {
	if s.userID != userID {
		// if we previously didn't have a user ID, filter the licenses based on the user.
		prevL := s.l
		prevUID := s.userID

		s.userID = userID
		s.l = Licenses{}

		if prevUID == "" {
			for lic, next := prevL.Iterate(); lic != nil; lic = next() {
				if (lic.UserID != "" && lic.UserID == userID) || (lic.InstallID != "" && lic.InstallID == s.installID) {
					s.l.Add(lic)
				}
			}
		}
	}
}

// Size returns the number of licenses, which are currently stored
func (s *Store) Size() int {
	return s.l.Len()
}

// ClearAll removes all licenses and resets the user ID to an empty string.
func (s *Store) ClearAll() {
	s.userID = ""
	s.l = Licenses{}
}

// ClearUser removes all licenses that do not match the installID.
func (s *Store) ClearUser() {
	prevL := s.l
	s.ClearAll()

	for lic, next := prevL.Iterate(); lic != nil; lic = next() {
		if lic.InstallID == s.installID {
			s.l.Add(lic)
		}
	}
}

// Add adds a new license. If the license isn't assigned to the currently configured user of this store,
// then an error is returned.
// An error is returned when the license validation failed.
func (s *Store) Add(token string) error {
	license, err := s.validator.Parse(token)
	// ignoring validation errors, the store manages expiration
	// expired license are still an indicator for the trial state
	if err != nil {
		return err
	}

	matchesUser := license.UserID != "" && license.UserID == s.userID
	matchesInstall := license.InstallID != "" && license.InstallID == s.installID
	if !matchesUser && !matchesInstall {
		return errors.Errorf("the license does not match the current user")
	}

	s.l.Add(license)
	if license.GetProduct() == Pro {
		s.hasHadPro = true
	}

	return nil
}

// TrialAvailable returns the trial state for the current machine
func (s *Store) TrialAvailable() bool {
	return !s.hasHadPro && s.l.TrialAvailable()
}

// LicenseStatus implements StatusGetter
func (s *Store) LicenseStatus() (time.Time, time.Time, Plan, Product) {
	if s.KiteServer {
		return time.Time{}, time.Time{}, ProServer, Pro
	}
	best := s.License()
	if best == nil {
		return time.Time{}, time.Time{}, best.GetPlan(), best.GetProduct()
	}
	return best.ExpiresAt, best.PlanEnd, best.GetPlan(), best.GetProduct()
}

// License returns the first active license of the current user, or nil.
func (s *Store) License() *License {
	return s.l.License()
}

// Plan returns the current plan.
// An empty string is returned if there's no valid license.
// If there are multiple valid licenses, then the license with the latest expiration date is returned
func (s *Store) Plan() Plan {
	if s.KiteServer {
		return ProServer
	}
	return s.l.License().GetPlan()
}

// Product returns the current product
// The Free product is returned if there's no valid license.
// If there are multiple valid licenses, then the product of the license with the latest expiration date is returned
func (s *Store) Product() Product {
	if s.KiteServer {
		return Pro
	}
	return s.l.License().GetProduct()
}
