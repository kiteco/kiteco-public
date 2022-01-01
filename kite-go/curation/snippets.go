package curation

import (
	"bytes"
	"fmt"
	"time"

	"github.com/jinzhu/gorm"
	"github.com/kiteco/kiteco/kite-go/web/webutils"
)

// Error codes that can be returned by the CuratedSnippetManager
const (
	ErrCodeDBError         = 1
	ErrCodeSnippetExists   = 2
	ErrCodeSnippetNotExist = 3

	ErrCodeCommentExists   = 4
	ErrCodeCommentNotExist = 5

	ErrCodeBadSnippetID   = 6
	ErrCodeBadSnippetBody = 7

	ErrCodeNeedEditLock = 8
)

// Valid snippet statuses.
const (
	SnippetStatusInProgress     = "in_progress"
	SnippetStatusPendingReview  = "pending_review"
	SnippetStatusNeedsAttention = "needs_attention"
	SnippetStatusApproved       = "approved"
	SnippetStatusDeleted        = "deleted"
)

// CuratedSnippetManager is responsible for the CuratedSnippet model, and defines methods that
// perform operations on CuratedSnippet via the DB. Note that CuratedSnippets are stored in an
// append-only form, so a particluar snippet ID maps to the latest snapshot of that snippet.
type CuratedSnippetManager struct {
	db   gorm.DB
	runs *RunManager
}

// NewCuratedSnippetManager returns a curated snippet manager.
func NewCuratedSnippetManager(db gorm.DB, runs *RunManager) *CuratedSnippetManager {
	return &CuratedSnippetManager{db: db, runs: runs}
}

// Migrate will auto-migrate relevant tables in the db.
func (c *CuratedSnippetManager) Migrate() error {
	if err := c.db.AutoMigrate(&CuratedSnippet{}, &Comment{}, &SupportingFile{}).Error; err != nil {
		return fmt.Errorf("error creating tables in db: %v", err)
	}
	return nil
}

// Create takes in a filled out CuratedSnippet and inserts it into the database.
func (c *CuratedSnippetManager) Create(snippet *CuratedSnippet) error {
	var err error
	if snippet.SnippetID > 0 {
		return webutils.ErrorCodef(ErrCodeSnippetExists, "trying to create snippet that already has an id")
	}

	latestID, err := c.latestSnippetID()
	if err != nil {
		return err
	}

	now := time.Now().Unix()

	snippet.SnippetID = latestID + 1
	snippet.SnapshotTimestamp = now
	if err := c.db.Create(snippet).Error; err != nil {
		return webutils.ErrorCodef(ErrCodeDBError, "error inserting new snippet: %v", err)
	}

	if err := c.updateOrCreateComments(snippet); err != nil {
		return err
	}

	return c.createSupportingFiles(snippet)
}

// GetBySnapshotID will return the snippet with SnapshotID equal to the provided id.
// Note that since the comments and the run output do not apply to
// snippets that are not the last snippet for the set of snippets
// with the same snippet id, the returned snippet do not have those
// fields filled.
func (c *CuratedSnippetManager) GetBySnapshotID(snapshotID int64) (*CuratedSnippet, error) {
	var snippet CuratedSnippet
	err := c.db.Where("SnapshotID = ?", snapshotID).Find(&snippet).Error
	if err != nil {
		switch err {
		case gorm.RecordNotFound:
			return nil, webutils.ErrorCodef(ErrCodeSnippetNotExist, "no snippet with snapshotID=%d", snapshotID)
		default:
			return nil, webutils.ErrorCodef(ErrCodeDBError, "error selecting snippet with snapshotID %d: %v", snapshotID, err)
		}
	}
	return &snippet, nil
}

// GetByID will return the latest snapshot of the snippet with SnippetID equal to the provided id.
func (c *CuratedSnippetManager) GetByID(id int64) (*CuratedSnippet, error) {
	var snippet CuratedSnippet
	err := c.db.Where("SnippetID = ?", id).Order("SnapshotID DESC").Limit(1).Find(&snippet).Error
	if err != nil {
		switch err {
		case gorm.RecordNotFound:
			return nil, webutils.ErrorCodef(ErrCodeSnippetNotExist, "no snippet with id %d", id)
		default:
			return nil, webutils.ErrorCodef(ErrCodeDBError, "error selecting latest snippet with id %d: %v", id, err)
		}
	}

	if snippet.Status == SnippetStatusDeleted {
		return nil, webutils.ErrorCodef(ErrCodeSnippetNotExist, "no snippet with id %d", id)
	}

	if err := c.db.Where("SnippetID = ?", id).Order("Created ASC").Find(&snippet.Comments).Error; err != nil {
		return nil, webutils.ErrorCodef(ErrCodeDBError, "error selecting comments for snippet id %d: %v", id, err)
	}

	if err := c.db.Model(&snippet).
		Related(&snippet.SupportingFiles, "SnapshotID").Error; err != nil {
		return nil, webutils.ErrorCodef(ErrCodeDBError,
			"error selecting supporting files for snapshot id %d: %v",
			snippet.SnapshotID, err)
	}

	if err := c.fillSnippetOutput(&snippet); err != nil {
		return nil, err
	}

	return &snippet, nil
}

// Update will update the snippet with id matching the SnippetID of the provided snippet.
func (c *CuratedSnippetManager) Update(snippet *CuratedSnippet) error {
	latest, err := c.GetByID(snippet.SnippetID)
	if err != nil {
		return err
	}

	now := time.Now().Unix()

	// also checks if supporting files have changed
	if snippetChanged(latest, snippet) {
		snippet.SnapshotID = 0
		snippet.SnapshotTimestamp = now

		// TODO(tarak): This shouldn't be necessary - should be sent to backend from the UI
		snippet.SnippetID = latest.SnippetID
		snippet.Package = latest.Package
		snippet.Language = latest.Language

		if err := c.db.Create(snippet).Error; err != nil {
			return webutils.ErrorCodef(ErrCodeDBError, "error updating (creating) snippet: %v", err)
		}

		if err := c.createSupportingFiles(snippet); err != nil {
			return err
		}
	}

	if commentsChanged(latest.Comments, snippet.Comments) {
		if err := c.updateOrCreateComments(snippet); err != nil {
			return err
		}
	}

	return nil
}

// ListAll will return all snippets in the db.
func (c *CuratedSnippetManager) ListAll() ([]*CuratedSnippet, error) {
	var latestSnapshotIDs []int64
	rows, err := c.db.Table("CuratedSnippet").Select("MAX(SnapshotID)").Group("SnippetID").Rows()
	if err != nil {
		return nil, fmt.Errorf("error selecting snippets: %v", err)
	}

	for rows.Next() {
		var snapshotID int64
		if err := rows.Scan(&snapshotID); err != nil {
			return nil, fmt.Errorf("error selecting snippets: %v", err)
		}
		latestSnapshotIDs = append(latestSnapshotIDs, snapshotID)
	}
	rows.Close()

	var snippets []*CuratedSnippet

	// if latestSnapshotIDs is empty, the SQL query below fails
	if len(latestSnapshotIDs) == 0 {
		return snippets, nil
	}

	if err := c.db.Where("SnapshotID IN (?) AND Status <> ?", latestSnapshotIDs, "deleted").Order("SnippetID ASC").Find(&snippets).Error; err != nil {
		return nil, fmt.Errorf("error selecting snippets: %v", err)
	}

	return snippets, nil
}

// List will return all snippets for a particular language/pkg pair.
func (c *CuratedSnippetManager) List(language, pkg string) ([]*CuratedSnippet, error) {
	var latestSnapshotIDs []int64

	rows, err := c.db.Table("CuratedSnippet").Select("MAX(SnapshotID)").Where("Language=? AND Package=?", language, pkg).Group("SnippetID").Rows()
	if err != nil {
		return nil, webutils.ErrorCodef(ErrCodeDBError, "error selecting snippets: %v", err)
	}

	for rows.Next() {
		var snapshotID int64
		if err := rows.Scan(&snapshotID); err != nil {
			return nil, webutils.ErrorCodef(ErrCodeDBError, "error selecting snippets: %v", err)
		}
		latestSnapshotIDs = append(latestSnapshotIDs, snapshotID)
	}
	rows.Close()

	snippets := []*CuratedSnippet{}

	// if latestSnapshotIDs is empty, the SQL query below fails
	if len(latestSnapshotIDs) == 0 {
		return snippets, nil
	}

	if err := c.db.Where("SnapshotID IN (?) AND Status <> ?", latestSnapshotIDs, "deleted").Order("SnippetID ASC").Find(&snippets).Error; err != nil {
		return nil, webutils.ErrorCodef(ErrCodeDBError, "error selecting snippets: %v", err)
	}

	for _, snippet := range snippets {
		snippet.Comments = []*Comment{}
		c.fillSnippetOutput(snippet)

		if err := c.db.Where("SnippetID = ?", snippet.SnippetID).Order("Created ASC").Find(&snippet.Comments).Error; err != nil {
			return nil, webutils.ErrorCodef(ErrCodeDBError,
				"unable to find comments for snippet %d: %v", snippet.SnippetID, err)
		}

		if err := c.db.Model(snippet).
			Related(&snippet.SupportingFiles, "SnapshotID").Error; err != nil {
			return nil, webutils.ErrorCodef(ErrCodeDBError,
				"error selecting supporting files for snapshot id %d: %v",
				snippet.SnapshotID, err)
		}
	}

	return snippets, nil
}

// SnippetQuery stores a query to be run across all snippets (not limited to a particular package)
type SnippetQuery struct {
	Statuses []string
}

// Query takes a SnippetQuery and returns the latest snapshots of all snippets which match
// (not limited to a particular language or package).
func (c *CuratedSnippetManager) Query(q SnippetQuery) ([]*CuratedSnippet, error) {
	snippets := []*CuratedSnippet{}
	rows, err := c.db.Table("CuratedSnippet").
		Select("MAX(SnapshotID)").
		Group("SnippetID").
		Rows()
	defer rows.Close()
	if err != nil {
		return nil, webutils.ErrorCodef(ErrCodeDBError, "error selecting snippets: %s", err)
	}

	var latestSnapshotIDs []int64
	for rows.Next() {
		var snapshotID int64
		if err := rows.Scan(&snapshotID); err != nil {
			return nil, webutils.ErrorCodef(ErrCodeDBError, "error reading row: %v", err)
		}
		latestSnapshotIDs = append(latestSnapshotIDs, snapshotID)
	}

	if len(latestSnapshotIDs) == 0 {
		return snippets, nil
	}

	err = c.db.
		Where("SnapshotID in (?) AND Status in (?)", latestSnapshotIDs, q.Statuses).
		Order("SnippetID ASC").
		Find(&snippets).Error

	if err != nil {
		return nil, webutils.ErrorCodef(ErrCodeDBError, "unable to selecting snapshots: %v", err)
	}

	// TODO: return comments too!

	return snippets, nil
}

// --

// CreateComment takes a filled in Comment and inserts it into the database.
func (c *CuratedSnippetManager) CreateComment(comment *Comment) error {
	if !c.db.NewRecord(comment) {
		return webutils.ErrorCodef(ErrCodeCommentExists, "trying to create comment that already has an id")
	}

	comment.Created = time.Now().Unix()

	if err := c.db.Create(comment).Error; err != nil {
		return webutils.ErrorCodef(ErrCodeDBError, "error inserting new comment: %v", err)
	}

	return nil
}

// GetByIDComment takes an ID of a comment and retrieves it from the database.
func (c *CuratedSnippetManager) GetByIDComment(id int64) (*Comment, error) {
	var comment Comment
	if err := c.db.First(&comment, id).Error; err != nil {
		switch err {
		case gorm.RecordNotFound:
			return nil, webutils.ErrorCodef(ErrCodeCommentNotExist, "no comment with id %d", id)
		default:
			return nil, webutils.ErrorCodef(ErrCodeDBError, "error selecting comment with id %d: %v", id, err)
		}
	}
	return &comment, nil
}

// UpdateComment takes an updated Comment struct and updates the corresponding
// entry in the database with the same ID.
func (c *CuratedSnippetManager) UpdateComment(comment *Comment) error {
	if comment.Dismissed == 1 {
		comment.Dismissed = time.Now().Unix()
	} else {
		comment.Modified = time.Now().Unix()
	}

	if err := c.db.Save(comment).Error; err != nil {
		return webutils.ErrorCodef(ErrCodeDBError, "error updating existing comment: %v", err)
	}
	return nil
}

// ListComments retrieves all comments for the given snippetID in ascending order
// by timestamp.
func (c *CuratedSnippetManager) ListComments(snippetID int64) ([]*Comment, error) {
	var comments []*Comment
	if err := c.db.Where(Comment{SnippetID: snippetID}).Order("Created").Find(&comments).Error; err != nil {
		return nil, webutils.ErrorCodef(ErrCodeDBError, "error retrieving comments for snippet %d: %v", snippetID, err)
	}
	return comments, nil
}

// --

func (c *CuratedSnippetManager) fillSnippetOutput(snippet *CuratedSnippet) error {
	r, err := c.runs.LookupLatestForSnippetAggregate(snippet.SnippetID)
	if err != nil {
		return fmt.Errorf("error getting run for snippet %d: %v", snippet.SnippetID, err)
	}
	if r != nil {
		snippet.Output = r.String()
	}
	return nil
}

// updateOrCreateComments will take a snippet and update or create comments based on
// the comments present in the provided snippet. A comment is considered new if its
// ID is not yet set.
func (c *CuratedSnippetManager) updateOrCreateComments(snippet *CuratedSnippet) error {
	now := time.Now().Unix()
	for _, comment := range snippet.Comments {
		comment.SnippetID = snippet.SnippetID
		comment.Modified = now
		switch comment.ID {
		case -1, 0:
			comment.ID = 0
			comment.Created = now
			comment.CreatedBy = snippet.User
			if err := c.db.Create(comment).Error; err != nil {
				return webutils.ErrorCodef(ErrCodeDBError, "error inserting new comment: %v", err)
			}
		default:
			comment.ModifiedBy = snippet.User
			createdBy := comment.CreatedBy
			created := comment.Created
			comment.CreatedBy = "" // never trust anybody else to set these;
			comment.Created = 0    // zero them out so they aren't updated
			if err := c.db.Model(Comment{ID: comment.ID}).Updates(comment).Error; err != nil {
				return webutils.ErrorCodef(ErrCodeDBError, "error updating existing comment: %v", err)
			}
			comment.CreatedBy = createdBy
			comment.Created = created
		}
	}
	return nil
}

func (c *CuratedSnippetManager) createSupportingFiles(snippet *CuratedSnippet) error {
	for _, file := range snippet.SupportingFiles {
		file.ID = 0
		file.SnapshotID = snippet.SnapshotID
		if err := c.db.Create(file).Error; err != nil {
			return webutils.ErrorCodef(ErrCodeDBError, "error inserting supporting file: %v", err)
		}
	}
	return nil
}

// latestSnippetID returns the max snippet id to be used by any snippet.
func (c *CuratedSnippetManager) latestSnippetID() (int64, error) {
	rows, err := c.db.Table("CuratedSnippet").Select("IFNULL(MAX(SnippetID), 0)").Rows()
	if err != nil {
		return 0, webutils.ErrorCodef(ErrCodeDBError, "error generating snippet id: %v", err)
	}

	var max int64
	for rows.Next() {
		rows.Scan(&max)
		break
	}
	rows.Close()
	return max, nil
}

// --

// Tests whether a snippet has changed.
func snippetChanged(s1, s2 *CuratedSnippet) bool {
	return (s1.Title != s2.Title ||
		s1.Prelude != s2.Prelude ||
		s1.Code != s2.Code ||
		s1.Postlude != s2.Postlude ||
		s1.Output != s2.Output ||
		s1.Status != s2.Status ||
		s1.ParallelProgram != s2.ParallelProgram ||
		s1.ApparatusSpec != s2.ApparatusSpec ||
		supportingFilesChanged(s1.SupportingFiles, s2.SupportingFiles))
}

func commentChanged(comment1, comment2 *Comment) bool {
	return comment1.Text != comment2.Text
}

// Tests whether comments have changed.
func commentsChanged(c1, c2 []*Comment) bool {
	if len(c1) != len(c2) {
		return true
	}
	if len(c1) == 0 {
		return false
	}

	for i := 0; i < len(c1); i++ {
		if commentChanged(c1[i], c2[i]) {
			return true
		}
	}

	return false
}

func supportingFilesChanged(f1, f2 []*SupportingFile) bool {
	if len(f1) != len(f2) {
		return true
	}
	if len(f1) == 0 {
		return false
	}

	equal := true
	for i := 0; i < len(f1); i++ {
		equal = equal && (f1[i].Path == f2[i].Path &&
			bytes.Equal(f1[i].Contents, f2[i].Contents))
	}
	return !equal
}
