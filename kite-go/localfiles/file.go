package localfiles

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/jinzhu/gorm"
	"github.com/jmoiron/sqlx"
)

// FileManager wraps the file db object.
type FileManager struct {
	dbx *sqlx.DB
}

// NewFileManager creates a new FileManager.
func NewFileManager(db *sqlx.DB) *FileManager {
	return &FileManager{
		dbx: db,
	}
}

// Migrate migrates the files db.
func (fm *FileManager) Migrate() error {
	db, err := gorm.Open("postgres", fm.dbx.DB)
	if err != nil {
		return fmt.Errorf("error creating tables in db: %v", err)
	}
	db.SingularTable(true)
	if err := db.AutoMigrate(&File{}).Error; err != nil {
		return fmt.Errorf("error creating tables in db: %v", err)
	}
	return nil
}

var (
	eventsPerBatch = 10000 // this is based on postgres's max of 65535 parameters. each event takes 6 params
	upsertQuery    = `
	INSERT INTO file (user_id, machine, name, hashed_content, created_at, updated_at)
	VALUES %s
	ON CONFLICT (user_id, machine, name) DO UPDATE SET
	hashed_content = excluded.hashed_content,
	updated_at = $%d
	`
)

// BatchCreateOrUpdate does a batch upsert to update the files table with the provided file events
func (fm *FileManager) BatchCreateOrUpdate(remaining []*FileEvent) error {
	for len(remaining) > 0 {
		var batch []*FileEvent
		if len(remaining) > eventsPerBatch {
			batch = remaining[:eventsPerBatch]
			remaining = remaining[eventsPerBatch:]
		} else {
			batch = remaining
			remaining = nil
		}

		err := fm.bulkUpsertEvents(batch)
		if err != nil {
			return err
		}
	}

	return nil
}

func insertQuery(events []*FileEvent) (string, []interface{}) {
	var stmt string
	valueStrings := make([]string, 0, len(events))
	valueArgs := make([]interface{}, 0, len(events)*6)
	now := string(formatTimestamp(time.Now().UTC()))
	var i int
	for _, f := range events {
		// TODO(hrysoula): Find longterm fix
		// This is a temporary fix for exceeding the var char limit of 255
		if len(f.Name) > 255 {
			continue
		}
		valueStrings = append(valueStrings, fmt.Sprintf("($%d, $%d, $%d, $%d, $%d, $%d)", i*6+1, i*6+2, i*6+3, i*6+4, i*6+5, i*6+6))
		valueArgs = append(valueArgs, f.UserID)
		valueArgs = append(valueArgs, f.Machine)
		valueArgs = append(valueArgs, f.Name)
		valueArgs = append(valueArgs, f.HashedContent)
		valueArgs = append(valueArgs, now)
		valueArgs = append(valueArgs, now)
		i++
	}
	if len(valueArgs) > 0 {
		// add one more arg for updated_at in the ON CONFLICT clause
		valueArgs = append(valueArgs, now)
		stmt = fmt.Sprintf(upsertQuery, strings.Join(valueStrings, ","), i*6+1)
	}
	return stmt, valueArgs
}

func (fm *FileManager) bulkUpsertEvents(events []*FileEvent) error {
	query, args := insertQuery(events)
	rows, err := fm.dbx.Query(query, args...)
	if err != nil {
		return err
	}

	defer rows.Close()
	return nil
}

// Get returns the file with the input id.
func (fm *FileManager) Get(id int64) (*File, error) {
	file := &File{}
	query := "SELECT * FROM file WHERE id=$1"
	err := fm.dbx.Get(file, query, id)
	file.fixName()
	return file, err
}

// Delete removes the file from the db.
func (fm *FileManager) Delete(f *FileEvent) error {
	query := "DELETE FROM file WHERE user_id=$1 AND machine=$2 AND name=$3"
	_, err := fm.dbx.Exec(query, f.UserID, f.Machine, f.Name)
	return err
}

// DeleteUser removes all files for a given user.
func (fm *FileManager) DeleteUser(user int64) error {
	query := "DELETE FROM file WHERE user_id=$1"
	_, err := fm.dbx.Exec(query, user)
	return err
}

// DeleteUserMachine removes all files for a given user and machine.
func (fm *FileManager) DeleteUserMachine(user int64, machine string) error {
	query := "DELETE FROM file WHERE user_id=$1 AND machine=$2"
	_, err := fm.dbx.Exec(query, user, machine)
	return err
}

// List returns an array of File objects.
func (fm *FileManager) List(user int64, machine string) ([]*File, error) {
	var files []*File

	query := "SELECT * FROM file WHERE user_id=$1 AND machine=$2"
	err := fm.dbx.Select(&files, query, user, machine)

	if err != nil {
		return nil, err
	}

	for _, f := range files {
		f.fixName()
	}

	return files, nil
}

var listChanBuffer = 100

// ListChan returns a channel of File objects.
func (fm *FileManager) ListChan(ctx context.Context, user int64, machine string) <-chan *File {
	ret := make(chan *File, listChanBuffer)

	go func() {
		defer close(ret)

		query := "SELECT * FROM file WHERE user_id=$1 AND machine=$2"
		rows, err := fm.dbx.QueryxContext(ctx, query, user, machine)
		if err != nil {
			log.Printf("error selecting files for (%d, %s): %s", user, machine, err)
			return
		}

		defer rows.Close()

		for rows.Next() {
			select {
			case <-ctx.Done():
				log.Printf("error scanning files for (%d, %s): %s", user, machine, ctx.Err())
				return
			default:
			}

			file := &File{}
			err = rows.StructScan(file)
			if err != nil {
				log.Printf("error scanning files for (%d, %s): %s", user, machine, err)
				return
			}
			file.fixName()
			ret <- file
		}
	}()

	return ret
}

// Machines returns an array of distinct machine ids for a given user.
func (fm *FileManager) Machines(uid int64) ([]string, error) {
	var machines []string
	query := "SELECT DISTINCT machine FROM file WHERE user_id=$1 AND machine IS NOT NULL"
	err := fm.dbx.Select(&machines, query, uid)
	return machines, err
}

// -------------------------------

// File holds all the information about a user's local file. Additional
// "db" tags are for sqlx.
type File struct {
	ID int64 `json:"id"` // Primary key, by GORM convention

	UserID        int64  `valid:"required" json:"user_id" db:"user_id"`
	Machine       string `valid:"required" json:"machine"`
	Name          string `valid:"required" json:"name"`
	HashedContent string `json:"hashed_content" db:"hashed_content"`

	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// lowercases name for windows file
func (f *File) fixName() {
	if strings.HasPrefix(f.Name, "/windows/") {
		f.Name = strings.ToLower(f.Name)
	}
}

// Content holds the content of a file and a hash of that content.
type Content struct {
	ContentHash []byte `json:"content_hash"`
	Content     []byte `json:"content"`
}

// EventType indicates a type of file event.
type EventType int

// Enum of possible file event types.
const (
	UnrecognizedEvent EventType = iota
	ModifiedEvent
	RemovedEvent
)

var eventTypes = []string{
	"None",
	"Modified",
	"Removed",
}

func (et EventType) String() string {
	return eventTypes[et]
}

// FileEvent extends the File data with the contents of the file.
type FileEvent struct {
	*File
	Content []byte `json:"content"`
	Type    EventType
}

// FileEvents is an array of FileEvent objects.
type FileEvents []*FileEvent

// UploadRequest groups file and content data for sending to the file server.
type UploadRequest struct {
	Files    FileEvents
	Contents map[string]*Content

	start   time.Time
	userID  int64
	machine string
}

// ContentUpdate groups file events by their content for writing.
type ContentUpdate struct {
	UserID  int64
	Hash    string
	Content []byte
	Files   FileEvents
	Context context.Context
}

// --

// formatTimestamp was pulled from the latest version of the postgres driver:
// https://github.com/lib/pq/blob/8af01c1982a76c61fb253cbc1f8f0b353d253967/encode.go#L470
// Our current version of pq does not export this method, and I wanted to avoid updating
// the driver since that could cause a bunch of knock-on issues.
func formatTimestamp(t time.Time) []byte {
	// Need to send dates before 0001 A.D. with " BC" suffix, instead of the
	// minus sign preferred by Go.
	// Beware, "0000" in ISO is "1 BC", "-0001" is "2 BC" and so on
	bc := false
	if t.Year() <= 0 {
		// flip year sign, and add 1, e.g: "0" will be "1", and "-10" will be "11"
		t = t.AddDate((-t.Year())*2+1, 0, 0)
		bc = true
	}
	b := []byte(t.Format("2006-01-02 15:04:05.999999999Z07:00"))

	_, offset := t.Zone()
	offset = offset % 60
	if offset != 0 {
		// RFC3339Nano already printed the minus sign
		if offset < 0 {
			offset = -offset
		}

		b = append(b, ':')
		if offset < 10 {
			b = append(b, '0')
		}
		b = strconv.AppendInt(b, int64(offset), 10)
	}

	if bc {
		b = append(b, " BC"...)
	}
	return b
}
