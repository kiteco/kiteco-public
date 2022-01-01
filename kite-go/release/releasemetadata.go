package release

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/kiteco/kiteco/kite-golib/errors"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

const majorVersion = 0

// Platform represents an OS platform for the client
type Platform int

const (
	// Cross represents the cross-platform javascript app
	Cross Platform = iota
	// Mac represents the native Mac wrapper app
	Mac
	// Windows represents the native Windows wrapper app
	Windows
	// Linux represents the native Linux wrapper app
	Linux
)

// majorVersion returns the prefix used in version numbers which is used for the platforms
func (p Platform) majorVersion() int8 {
	switch p {
	case Mac:
		return 0
	case Windows:
		return 1
	case Linux:
		return 2
	default:
		return 0
	}
}

// String implements fmt.Stringer
func (p Platform) String() string {
	switch p {
	case Mac:
		return "mac"
	case Windows:
		return "windows"
	case Linux:
		return "linux"
	default:
		return ""
	}
}

// ParsePlatform parses a Platform from a string
func ParsePlatform(s string) (Platform, error) {
	switch s {
	case "windows":
		return Windows, nil
	case "linux":
		return Linux, nil
	case "mac":
		return Mac, nil
	}
	return -1, errors.Errorf("unrecognized platform %s", s)
}

func version(client Platform, t time.Time, nthRelease int) string {
	if client == Windows {
		return fmt.Sprintf("1.%04d.%d%02d.%d", t.Year(), t.Month(), t.Day(), nthRelease)
	}
	releaseDate := dateInt(t)
	return fmt.Sprintf("%d.%d.%d", client.majorVersion(), releaseDate, nthRelease)
}

// normalize takes a version number and turns it into a
// version string of form yyyymmdd.nthrelease
// It only does minimal validation
func normalize(client Platform, version string) (string, error) {
	if client == Windows {
		// Windows version format: majorVersion.yyyy.monthdd.nthrelease, eg 1.2021.111.1
		var ignore, yyyy, monthdd, nth int
		n, err := fmt.Sscanf(version, "%1d.%4d.%d.%d\n", &ignore, &yyyy, &monthdd, &nth)
		if err != nil {
			return "", errors.Errorf("could not parse version string: ", err)
		} else if n != 4 {
			return "", errors.Errorf("could not parse version string: expected four parts, parsed ", n)
		}

		return strings.Join([]string{
			// 111 is January 11th, padding with a 0 results in 0111 for MMDD
			strconv.FormatInt(int64(yyyy), 10) + fmt.Sprintf("%04d", monthdd),
			strconv.FormatInt(int64(nth), 10),
		}, "."), nil
	}

	// Linux and Mac only need to truncate the client.majorVersion from the front
	// majorVersion.yearmonthday.nthrelease, eg 1.20210111.1
	var ignore, yyyymmdd, nth int
	n, err := fmt.Sscanf(version, "%1d.%8d.%d\n", &ignore, &yyyymmdd, &nth)
	if err != nil {
		return "", errors.Errorf("could not parse version string: ", err)
	} else if n != 3 {
		return "", errors.Errorf("could not parse version string: expected four parts, parsed ", n)
	}

	return strings.Join([]string{
		strconv.FormatInt(int64(yyyymmdd), 10),
		strconv.FormatInt(int64(nth), 10),
	}, "."), nil
}

var (
	// ErrVersionNotFound occurs when there are no releases for the requested version
	ErrVersionNotFound = errors.New("no release with given version found")
)

// Metadata stores information about each release, saved in the database
// Fields added here should be included in latest_idx since we select *
type Metadata struct {
	ID int64 `gorm:"primary_key"`
	// GORM sets this to the current time
	CreatedAt         time.Time `gorm:"not null"`
	Client            Platform  `gorm:"not null"`
	Version           string    `gorm:"type:varchar(15);not null"`
	NormalizedVersion string    `gorm:"type:varchar(15);not null"`
	DSASignature      string    `gorm:"type:varchar(255)"`
	GitHash           string    `gorm:"type:varchar(40);not null"`
	// ReleasePercentage is the relative amount of users who
	// receive an update ((0, 100) as canary)
	//   0: no users
	// 100: all users
	ReleasePercentage uint8 `gorm:"not null"`
	Public            bool  `gorm:"not null"`
}

// Copy must be updated when Metadata changes
func (m *Metadata) Copy() MetadataCreateArgs {
	return MetadataCreateArgs{
		Client:            m.Client,
		Version:           m.Version,
		DSASignature:      m.DSASignature,
		GitHash:           m.GitHash,
		ReleasePercentage: m.ReleasePercentage,
		Public:            m.Public,
	}
}

// MetadataCreateArgs is used to create a new entry in the Metadata table
type MetadataCreateArgs struct {
	Client            Platform
	Version           string
	DSASignature      string
	GitHash           string
	ReleasePercentage uint8
	Public            bool
}

// String implments stringer
func (m *Metadata) String() string {
	return strings.Join([]string{
		fmt.Sprintf("ID=%d", m.ID),
		fmt.Sprintf("CreatedAt=%s", m.CreatedAt.Format(time.RFC3339)),
		fmt.Sprintf("Client=%s", m.Client),
		fmt.Sprintf("Version=%s", m.Version),
		fmt.Sprintf("NormalizedVersion=%s", m.NormalizedVersion),
		fmt.Sprintf("DSASignatuure=%s", m.DSASignature),
		fmt.Sprintf("GitHash=%s", m.GitHash),
		fmt.Sprintf("ReleasePercentage=%d", m.ReleasePercentage),
		fmt.Sprintf("Public=%t", m.Public),
	}, "\n")
}

// Delta stores information about each delta update, saved in the database
type Delta struct {
	ID int64 `gorm:"primary_key"`
	// GORM sets this to the current time
	CreatedAt    time.Time `gorm:"not null"`
	Client       Platform  `gorm:"not null"`
	FromVersion  string    `gorm:"type:varchar(15);not null"`
	ToVersion    string    `gorm:"type:varchar(15);not null"`
	DSASignature string    `gorm:"type:varchar(255)"`
}

// String implments stringer
func (d *Delta) String() string {
	return strings.Join([]string{
		fmt.Sprintf("ID=%d", d.ID),
		fmt.Sprintf("CreatedAt=%s", d.CreatedAt.Format(time.RFC3339)),
		fmt.Sprintf("Client=%s", d.Client),
		fmt.Sprintf("FromVersion=%s", d.FromVersion),
		fmt.Sprintf("ToVersion=%s", d.ToVersion),
		fmt.Sprintf("DSASignatuure=%s", d.DSASignature),
	}, "\n")
}

// MetadataManager specifies the operations one can take on release metadata stores.
// We create an interface primarily so that we can sometimes use a MockMetadataManager.
type MetadataManager interface {
	Migrate() error
	// Write
	Create(MetadataCreateArgs) (*Metadata, error)
	Publish(client Platform, version string, releasePercentage uint8) error
	// Read
	LatestCanary(client Platform) (*Metadata, error)
	LatestNonCanary(client Platform) (*Metadata, error)
	// Delta Updates
	CreateDelta(client Platform, fromVersion, toVersion, dsaSignature string) (*Delta, error)
	DeltasToVersion(client Platform, version string) ([]*Delta, error)
}

// MetadataManagerImpl implements a MetadataManager backed by a database.
type MetadataManagerImpl struct {
	db *gorm.DB
	// If readsPublic is false, this manager only reads private
	readsPublic bool
}

// DB makes a new database for the MetadataManager to use
func DB(driver, uri string) *gorm.DB {
	db, err := gorm.Open(postgres.New(postgres.Config{
		DriverName:           driver,
		DSN:                  uri,
		PreferSimpleProtocol: true,
	}), &gorm.Config{
		NamingStrategy: schema.NamingStrategy{
			SingularTable: true,
		},
	})
	if err != nil {
		log.Fatal(err)
	}

	d, err := db.DB()
	if err != nil {
		log.Fatal(err)
	}
	d.SetConnMaxLifetime(time.Second * 60)
	return db
}

// NewMetadataManager creates a new MetadataManager with database
func NewMetadataManager(dbDriver, dbURI string, readsPublic bool) *MetadataManagerImpl {
	return &MetadataManagerImpl{
		db:          DB(dbDriver, dbURI),
		readsPublic: readsPublic,
	}
}

// Migrate auto-migrates relevant tables in the db.
func (m *MetadataManagerImpl) Migrate() error {
	if err := m.db.AutoMigrate(&Metadata{}, &Delta{}); err != nil {
		return errors.Errorf("error creating tables in DB: %v", err)
	}
	// Gorm does not support index tag INCLUDE, so we use raw sql here
	if !m.db.Migrator().HasIndex(&Metadata{}, "latest_idx") {
		res := m.db.Exec(`
      CREATE INDEX
        latest_idx
      ON
        metadata (public, "version", client, created_at DESC)
      INCLUDE
        (id, normalized_version, dsa_signature, git_hash, release_percentage)
    `)
		if res.Error != nil {
			return res.Error
		}
	}
	if !m.db.Migrator().HasIndex(&Delta{}, "deltas_to_version_idx") {
		res := m.db.Exec(`
      CREATE INDEX
        deltas_to_version_idx
      ON
        delta (client, to_version, from_version DESC)
      INCLUDE
        (id, dsa_signature, created_at)
    `)
		if res.Error != nil {
			return res.Error
		}
	}
	return nil
}

// dateInt takes a time.Time and returns an int which represents the date. Example:
// dateInt(time.Date(2015, time.September, 29, 0, 0, 0, 0, time.UTC)) -> 20150929
func dateInt(t time.Time) int64 {
	return int64(t.Year()*10000 + int(t.Month())*100 + t.Day())
}

// Create adds a new release entry in the database.
func (m *MetadataManagerImpl) Create(args MetadataCreateArgs) (*Metadata, error) {
	if args.ReleasePercentage < 0 || args.ReleasePercentage > 100 {
		return nil, errors.Errorf("canary percentage must be in range [0, 100]")
	}

	nver, err := normalize(args.Client, args.Version)
	if err != nil {
		return nil, err
	}
	newEntry := &Metadata{
		Client:            args.Client,
		Version:           args.Version,
		NormalizedVersion: nver,
		DSASignature:      args.DSASignature,
		GitHash:           args.GitHash,
		ReleasePercentage: args.ReleasePercentage,
		Public:            args.Public,
	}
	if err := m.db.Create(newEntry).Error; err != nil {
		return nil, errors.Errorf("database error while saving new release metadata: %s", err)
	}
	// Match the timezone between Create and select method returned Metadata
	newEntry.CreatedAt = newEntry.CreatedAt.UTC()
	return newEntry, nil
}

// LatestCanary returns the most recent release, including canaries, on a given platform.
func (m *MetadataManagerImpl) LatestCanary(client Platform) (*Metadata, error) {
	releases := m.db.Table("(?) as client_versions", m.latestRowsForClientVersion()).
		Select("*").
		Where("client = ?", client).
		Where("release_percentage > 0").
		Order("created_at desc").
		Limit(1)

	latest := &Metadata{}
	err := releases.Take(latest).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	return latest, nil
}

// LatestNonCanary returns the most recent, fully-released version for a given platform
func (m *MetadataManagerImpl) LatestNonCanary(client Platform) (*Metadata, error) {
	releases := m.db.Table("(?) as client_versions", m.latestRowsForClientVersion()).
		Select("*").
		Where("client = ?", client).
		Where("release_percentage = 100").
		Order("created_at desc").
		Limit(1)

	latest := &Metadata{}
	err := releases.Take(latest).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	return latest, nil
}

func (m *MetadataManagerImpl) latestRowsForClientVersion() *gorm.DB {
	return m.db.
		Select("DISTINCT ON (version, client) *").
		Model(&Metadata{}).
		Where("public = ?", m.readsPublic).
		Order("version, client, created_at DESC")
}

// nextVersion returns the next version number to be used for a new release
// It's only used for testing
func (m *MetadataManagerImpl) nextVersion(client Platform) (string, error) {
	now := time.Now()

	latestEntryToday := m.db.Model(&Metadata{}).
		Where("client = ?", client).
		Where("normalized_version LIKE ?", now.Format("20060102")+"%").
		Order("created_at DESC").
		Limit(1)

	latest := &Metadata{}
	err := latestEntryToday.Take(latest).Error
	if err == gorm.ErrRecordNotFound {
		return version(client, now, 0), nil
	} else if err != nil {
		return "", errors.Errorf("database error while getting next version: %s", err)
	}

	// Previous record exists, take the last version and increment
	nv := strings.Split(latest.NormalizedVersion, ".")
	if len(nv) != 2 {
		return "", errors.Errorf("NormalizedVersion %s was not formatted as expected yyyymmdd.nthver", latest.NormalizedVersion)
	}
	lastNthRelease, err := strconv.Atoi(nv[1])
	if err != nil {
		return "", err
	}
	return version(client, now, lastNthRelease+1), nil
}

// fromVersion returns the latest release metadata for the specified version and platform
// disregarding release percentage
func (m *MetadataManagerImpl) fromVersion(client Platform, version string) (*Metadata, error) {
	if version == "*" {
		return m.LatestCanary(client)
	}
	q := m.db.Model(&Metadata{}).
		Where("client = ?", client).
		Where("version = ?", version).
		Order("created_at DESC").
		Limit(1)

	metadata := &Metadata{}
	err := q.Take(metadata).Error
	if err == gorm.ErrRecordNotFound {
		return nil, ErrVersionNotFound
	} else if err != nil {
		return nil, err
	}
	// Match the timezone between Create and select method returned Metadata
	metadata.CreatedAt = metadata.CreatedAt.UTC()
	return metadata, nil
}

// Publish updates the release percentage and makes the release public
func (m *MetadataManagerImpl) Publish(client Platform, version string, releasePercentage uint8) error {
	if releasePercentage < 0 || releasePercentage > 100 {
		return errors.Errorf("canary_percentage must be in range [0, 100]")
	}

	mdata, err := m.fromVersion(client, version)
	if err != nil {
		return err
	}
	cargs := mdata.Copy()
	cargs.ReleasePercentage = releasePercentage
	cargs.Public = true
	_, err = m.Create(cargs)
	return err
}

// CreateDelta adds a delta update to the db for the given client and from->to versions.
func (m *MetadataManagerImpl) CreateDelta(client Platform, fromVersion, toVersion, dsaSignature string) (*Delta, error) {
	var conflicts int64
	err := m.db.Model(&Delta{}).Where("client = ? AND from_version = ? AND to_version = ?", client, fromVersion, toVersion).Count(&conflicts).Error
	if err != nil {
		return nil, err
	}
	if conflicts > 0 {
		return nil, errors.Errorf("for client %d, from_version %s, to_version %s, found %d existing record", client, fromVersion, toVersion, conflicts)
	}

	newDelta := &Delta{
		Client:       client,
		FromVersion:  fromVersion,
		ToVersion:    toVersion,
		DSASignature: dsaSignature,
	}

	err = m.db.Create(newDelta).Error
	if err != nil {
		return nil, errors.Errorf("database error while saving new delta: %s", err)
	}
	// Match the timezone between Create and select method returned Delta
	newDelta.CreatedAt = newDelta.CreatedAt.UTC()
	return newDelta, nil
}

// DeltasToVersion returns deltas that update to the given version.
func (m *MetadataManagerImpl) DeltasToVersion(client Platform, version string) ([]*Delta, error) {
	var deltas []*Delta
	q := m.db.Model(&Delta{}).Where("client = ? AND to_version = ?", client, version).Order("from_version DESC")
	err := q.Find(&deltas).Error
	if err == gorm.ErrRecordNotFound {
		return nil, ErrVersionNotFound
	} else if err != nil {
		return nil, err
	}
	return deltas, nil
}
