package curation

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCuratedSnippetCreateGet(t *testing.T) {
	manager := setupCuratedSnippetManager()
	numToCreate := 10

	// Create and Get a series of snippet objects. Check critical properties.
	var lastSnippetID, lastSnapshotID int64
	for i := 0; i < numToCreate; i++ {
		snippet1 := makeFakeSnippet(i)
		err := manager.Create(snippet1)
		assert.NoError(t, err, "expected create to succeed")
		assert.True(t, snippet1.SnippetID > lastSnippetID, "expected SnippetID to be greater than last SnippetID")
		assert.True(t, snippet1.SnapshotID > lastSnapshotID, "expected SnapshotID to be greater than last SnapshotID")
		assert.NotEqual(t, 0, snippet1.SnapshotTimestamp, "expected SnapshotTimestamp to be nonzero")

		snippet2, err := manager.GetByID(snippet1.SnippetID)
		assert.NoError(t, err, "expected get to succeed")
		assertSnippetsEqual(t, snippet1, snippet2)

		lastSnippetID = snippet2.SnippetID
		lastSnapshotID = snippet2.SnapshotID
	}
}

func TestCuratedSnippetUpdateGet(t *testing.T) {
	manager := setupCuratedSnippetManager()

	// Create a new snippet
	snippet1 := makeFakeSnippet(0)
	err := manager.Create(snippet1)
	assert.NoError(t, err, "expected create to succeed")

	// Get the snippet by ID, assert its what we just inserted
	snippet2, err := manager.GetByID(snippet1.SnippetID)
	assert.NoError(t, err, "expected get to succeed")
	assertSnippetsEqual(t, snippet1, snippet2)

	// Create a different snippet, but with ID set to existing snippet id
	snippet3 := makeFakeSnippet(1)
	snippet3.SnippetID = snippet1.SnippetID
	err = manager.Update(snippet3)
	assert.NoError(t, err, "expected update to succeed")

	// Grab the snippet via SnippetID. Test that we got the updated snippet.
	snippet4, err := manager.GetByID(snippet1.SnippetID)
	assert.NoError(t, err, "expected GetById to succeed")
	assert.Equal(t, snippet2.SnippetID, snippet4.SnippetID, "SnippetID's should be equal")
	assertSnippetsEqual(t, snippet3, snippet4)

	assert.True(t, snippet4.SnapshotID > snippet2.SnapshotID, "SnapshotID is not incrementing")
}

func TestCuratedSnippetUpdate_WithComments(t *testing.T) {
	manager := setupCuratedSnippetManager()

	// Create a snippet
	snippet1 := makeFakeSnippet(0)
	err := manager.Create(snippet1)
	assert.NoError(t, err, "expected create to succeed")

	// Create two new comments
	numComments := 2
	var comments []*Comment
	for i := 0; i < numComments; i++ {
		fakeComment := makeFakeComment(i)
		comments = append(comments, fakeComment)
		snippet1.Comments = append(snippet1.Comments, fakeComment)
	}

	// Update the snippet with new comments
	err = manager.Update(snippet1)
	assert.NoError(t, err, "expected update to succeed")

	// Retrieve the snippet
	snippet2, err := manager.GetByID(snippet1.SnippetID)
	assert.NoError(t, err, "expected GetByID to succeed")

	// Make sure snippets are equal (snippetEqual checks comments and supporting files too)
	assertSnippetsEqual(t, snippet1, snippet2)

	// Update the first two comments, add two new comments
	for _, comment := range snippet1.Comments {
		comment.Text = comment.Text + "UPDATED"
	}

	// Update the snippet
	err = manager.Update(snippet1)
	assert.NoError(t, err, "expected update to succeed")

	for i := 0; i < numComments; i++ {
		fakeComment := makeFakeComment(10 * i)
		comments = append(comments, fakeComment)
		snippet1.Comments = append(snippet1.Comments, makeFakeComment(i))
	}

	// Update the snippet
	err = manager.Update(snippet1)
	assert.NoError(t, err, "expected update to succeed")

	// Retrieve the snippet
	snippet3, err := manager.GetByID(snippet1.SnippetID)
	assert.NoError(t, err, "expected GetByID to succeed")

	// Make sure snippets are equal (snippetEqual checks comments and supporting files too)
	assertSnippetsEqual(t, snippet1, snippet3)
}

func TestCuratedSnippetUpdate_WithSupportingFiles(t *testing.T) {
	manager := setupCuratedSnippetManager()

	// Create a snippet
	snippet1 := makeFakeSnippet(0)
	err := manager.Create(snippet1)
	assert.NoError(t, err, "expected create to succeed")

	// Create new supporting file
	fakeSupportingFile := makeFakeSupportingFile(0)
	snippet1.SupportingFiles = append(snippet1.SupportingFiles, fakeSupportingFile)

	// Update the snippet with a new supporting file
	err = manager.Update(snippet1)
	assert.NoError(t, err, "expected update to succeed")

	// Retrieve the snippet
	snippet2, err := manager.GetByID(snippet1.SnippetID)
	assert.NoError(t, err, "expected GetByID to succeed")

	// Make sure snippets are equal (snippetEqual checks supporting files too)
	assertSnippetsEqual(t, snippet1, snippet2)

	// Update the first supporting file
	for _, file := range snippet1.SupportingFiles {
		file.Contents = []byte(string(file.Contents) + " UPDATED")
	}

	// Update the snippet
	err = manager.Update(snippet1)
	assert.NoError(t, err, "expected update to succeed")

	// Add one more supporting file
	fakeSupportingFile = makeFakeSupportingFile(1)
	snippet1.SupportingFiles = append(snippet1.SupportingFiles, fakeSupportingFile)

	// Update the snippet
	err = manager.Update(snippet1)
	assert.NoError(t, err, "expected update to succeed")

	// Retrieve the snippet
	snippet3, err := manager.GetByID(snippet1.SnippetID)
	assert.NoError(t, err, "expected GetByID to succeed")

	// Make sure snippets are equal (snippetEqual checks supporting files too)
	assertSnippetsEqual(t, snippet1, snippet3)

	// Create another snippet with a SupportingFile already
	snippet4 := makeFakeSnippet(1)
	fakeSupportingFile = makeFakeSupportingFile(2)
	snippet4.SupportingFiles = append(snippet4.SupportingFiles, fakeSupportingFile)
	err = manager.Create(snippet4)
	assert.NoError(t, err, "expected create to succeed")

	// Retrieve the latest snippet
	snippet5, err := manager.GetByID(snippet4.SnippetID)
	assert.NoError(t, err, "expected GetByID to succeed")

	// Make sure snippets are equal (snippetEqual checks supporting files too)
	assertSnippetsEqual(t, snippet4, snippet5)

	allSnippetsWithFiles, err := manager.List(testLang, testPackage)
	assert.NoError(t, err, "expected List to succeed")

	assertSnippetsEqual(t, allSnippetsWithFiles[0], snippet1)
	assertSnippetsEqual(t, allSnippetsWithFiles[1], snippet4)
}

func TestCuratedSnippetUpdate_CommentsProtectCreated(t *testing.T) {
	manager := setupCuratedSnippetManager()

	// Create a snippet
	snippet := makeFakeSnippet(0)
	err := manager.Create(snippet)
	assert.NoError(t, err, "expected create to succeed")

	// Create a comment
	comment := &Comment{
		Text: "trust, but verify",
	}
	snippet.Comments = append(snippet.Comments, comment)

	// Update the snippet with new comment
	err = manager.Update(snippet)
	assert.NoError(t, err, "expected update to succeed")

	// Update the comment with fields that Update should ignore
	comment.Text = "the end of an era"
	comment.CreatedBy = "Captain America"
	comment.Created = 1337
	err = manager.Update(snippet)
	assert.NoError(t, err, "expected update to succeed")

	// Retrieve the snippet
	snippet, err = manager.GetByID(snippet.SnippetID)
	assert.NoError(t, err, "expected GetByID to succeed")

	// Check that the text was updated
	assert.Equal(t, comment.Text, snippet.Comments[0].Text)
	// Check that the untrusted fields (CreatedBy and Created) were not updated
	assert.NotEqual(t, comment.CreatedBy, snippet.Comments[0].CreatedBy)
	assert.NotEqual(t, comment.Created, snippet.Comments[0].Created)
}

func TestCuratedSnippetUpdate_Delete(t *testing.T) {
	manager := setupCuratedSnippetManager()

	// Create a snippet
	snippet1 := makeFakeSnippet(0)
	err := manager.Create(snippet1)
	assert.NoError(t, err, "expected create to succeed")

	// Retrieve the snippet
	snippet2, err := manager.GetByID(snippet1.SnippetID)
	assert.NoError(t, err, "expected GetByID to succeed")

	// Update the snippet status to be deleted
	snippet2.Status = SnippetStatusDeleted
	err = manager.Update(snippet2)
	assert.NoError(t, err, "expected update to succeed")

	// Retrieve the snippet, expect error
	_, err = manager.GetByID(snippet1.SnippetID)
	assert.Error(t, err, "expected GetByID to fail")
}

func TestCuratedSnippetUpdate_NoChange(t *testing.T) {
	manager := setupCuratedSnippetManager()

	// Create a new snippet
	snippet1 := makeFakeSnippet(0)
	err := manager.Create(snippet1)
	assert.NoError(t, err, "expected create to succeed")

	// Get the snippet by ID, assert its what we just inserted
	snippet2, err := manager.GetByID(snippet1.SnippetID)
	assert.NoError(t, err, "expected get to succeed")
	assertSnippetsEqual(t, snippet1, snippet2)

	// TODO(tarak): We have to sleep between creation and update because snapshot timestamps only record seconds
	// instead of nanoseconds. Update this to use nanoseconds.
	time.Sleep(2 * time.Second)

	// Create the same snippet, but with ID set to existing snippet id
	snippet3 := makeFakeSnippet(0)
	snippet3.SnippetID = snippet1.SnippetID
	err = manager.Update(snippet3)
	assert.NoError(t, err, "expected update to succeed")

	// Grab the snippet via SnippetID. Test that we got the updated snippet.
	snippet4, err := manager.GetByID(snippet1.SnippetID)
	assert.NoError(t, err, "expected GetById to succeed")
	assert.Equal(t, snippet2.SnippetID, snippet4.SnippetID, "SnippetID's should be equal")
	assertSnippetsEqual(t, snippet4, snippet2)

	assert.Equal(t, snippet2.SnapshotID, snippet4.SnapshotID, "SnapshotID should be unchanged")
	assert.Equal(t, snippet2.SnapshotTimestamp, snippet4.SnapshotTimestamp, "SnapshotTimestamp should be unchanged")
}

func TestCuratedSnippetList(t *testing.T) {
	manager := setupCuratedSnippetManager()
	numToCreate := 2

	// Create a few snippets...
	var snippets []*CuratedSnippet
	for i := 0; i < numToCreate; i++ {
		snippet1 := makeFakeSnippet(i)
		err := manager.Create(snippet1)
		assert.NoError(t, err, "expected create to succeed")
		assert.NotEqual(t, 0, snippet1.SnippetID, "expected SnippetID to be nonzero")
		assert.NotEqual(t, 0, snippet1.SnapshotID, "expected SnapshotID to be nonzero")
		assert.NotEqual(t, 0, snippet1.SnapshotTimestamp, "expected SnapshotTimestamp to be nonzero")
		snippets = append(snippets, snippet1)
	}

	list, err := manager.List(testLang, testPackage)
	assert.NoError(t, err, "expected list to succeed")
	assert.Equal(t, len(list), len(snippets), "got mismatched number of snippets")

	for i := 0; i < len(list); i++ {
		assertSnippetsEqual(t, snippets[i], list[i])
	}
}

func TestCuratedSnippetList_AfterUpdate(t *testing.T) {
	manager := setupCuratedSnippetManager()
	snippetIndexToUpdate := 2
	numToCreate := 4

	// Create a few snippets...
	var snippets []*CuratedSnippet
	for i := 0; i < numToCreate; i++ {
		snippet1 := makeFakeSnippet(i)
		err := manager.Create(snippet1)
		assert.NoError(t, err, "expected create to succeed")
		assert.NotEqual(t, 0, snippet1.SnippetID, "expected SnippetID to be nonzero")
		assert.NotEqual(t, 0, snippet1.SnapshotID, "expected SnapshotID to be nonzero")
		assert.NotEqual(t, 0, snippet1.SnapshotTimestamp, "expected SnapshotTimestamp to be nonzero")
		snippets = append(snippets, snippet1)
	}

	updatedSnippet := makeFakeSnippet(100)
	updatedSnippet.SnippetID = snippets[snippetIndexToUpdate].SnippetID
	snippets[snippetIndexToUpdate] = updatedSnippet

	err := manager.Update(updatedSnippet)
	assert.NoError(t, err, "expected update to succeed")

	list, err := manager.List(testLang, testPackage)
	assert.NoError(t, err, "expected list to succeed")
	assert.Equal(t, len(snippets), len(list), "got mismatched number of snippets")

	for i := 0; i < len(list); i++ {
		assertSnippetsEqual(t, snippets[i], list[i])
	}
}

// --

func TestCommentCreateGet(t *testing.T) {
	manager := setupCuratedSnippetManager()
	numToCreate := 10

	var lastCommentID int64
	for i := 0; i < numToCreate; i++ {
		// Create a new comment
		comment := makeFakeComment(i)
		comment.SnippetID = 1
		err := manager.CreateComment(comment)
		assert.NoError(t, err, "error creating comment %d", i)
		assert.True(t, comment.ID > lastCommentID, "expected ID to be greater than last comment's ID")
		assert.NotEqual(t, "", comment.Created, "expected Created field to be set")

		// Get same comment by ID
		retrieved, err := manager.GetByIDComment(comment.ID)
		require.NoError(t, err, "expected get comment to succeed")

		assertCommentEqual(t, comment, retrieved)

		lastCommentID = comment.ID

		// Sleep for half a second so that Created timestamps are different
		time.Sleep(500 * time.Millisecond)
	}
}

func TestCommentUpdateGet(t *testing.T) {
	manager := setupCuratedSnippetManager()

	// Create a new comment
	comment1 := makeFakeComment(0)
	comment1.SnippetID = 1
	err := manager.CreateComment(comment1)
	assert.NoError(t, err, "expected create comment to succeed")

	// Update fields
	comment1.Text = comment1.Text + "UPDATED"
	err = manager.UpdateComment(comment1)
	assert.NoError(t, err, "expected update comment to succeed")

	comment2, err := manager.GetByIDComment(comment1.ID)
	require.NoError(t, err, "expected get by id to succeed")
	assert.Equal(t, comment1.Text, comment2.Text, "expected Text fields to be same")
	assert.Equal(t, comment1.Modified, comment2.Modified, "expected Modified field to be set by Update operation")
}

func TestCommentList(t *testing.T) {
	manager := setupCuratedSnippetManager()
	numToCreate := 5

	var lastCommentID int64
	var created []*Comment
	for i := 0; i < numToCreate; i++ {
		// Create a new comment
		comment := makeFakeComment(i)
		comment.SnippetID = 1
		err := manager.CreateComment(comment)
		assert.NoError(t, err, "error creating comment %d", i)
		assert.True(t, comment.ID > lastCommentID, "expected ID to be greater than last comment's ID")

		created = append(created, comment)

		lastCommentID = comment.ID

		// Sleep for half a second so that Created timestamps are different
		time.Sleep(500 * time.Millisecond)
	}

	comments, err := manager.ListComments(1)
	require.NoError(t, err, "expected ListComments to succeed")

	assertCommentsEqual(t, created, comments)
}

func TestCommentList_AfterUpdateDismiss(t *testing.T) {
	manager := setupCuratedSnippetManager()
	commentIndexToUpdate := 2
	commentIndexToDismiss := 3
	numToCreate := 5

	var lastCommentID int64
	var created []*Comment
	for i := 0; i < numToCreate; i++ {
		// Create a new comment
		comment := makeFakeComment(i)
		comment.SnippetID = 1
		err := manager.CreateComment(comment)
		assert.NoError(t, err, "error creating comment %d", i)
		assert.True(t, comment.ID > lastCommentID, "expected ID to be greater than last comment's ID")

		created = append(created, comment)

		lastCommentID = comment.ID

		// Sleep for half a second so that Created timestamps are different
		time.Sleep(500 * time.Millisecond)
	}

	// Update a comment's Text
	created[commentIndexToUpdate].Text += "UPDATED"
	err := manager.UpdateComment(created[commentIndexToUpdate])
	assert.NoError(t, err, "expected update to succeed")

	retrieved, err := manager.GetByIDComment(created[commentIndexToDismiss].ID)
	assert.NoError(t, err, "expected GetByIDComment to succeed")
	created[commentIndexToDismiss] = retrieved

	comments, err := manager.ListComments(1)
	require.NoError(t, err, "expected ListComments to succeed")

	assertCommentsEqual(t, created, comments)
}

// --

const (
	testLang    = "python"
	testPackage = "os"
)

func setupCuratedSnippetManager() *CuratedSnippetManager {
	db := GormDB("sqlite3", ":memory:")
	runs := NewRunManager(db)
	snippets := NewCuratedSnippetManager(db, runs)
	snippets.Migrate()
	runs.Migrate()
	return snippets
}

func makeFakeSnippet(n int) *CuratedSnippet {
	return &CuratedSnippet{
		User:            fmt.Sprintf("user%d", n),
		Status:          SnippetStatusInProgress,
		Language:        testLang,
		Package:         testPackage,
		Title:           fmt.Sprintf("title%d", n),
		Code:            fmt.Sprintf("code%d", n),
		Prelude:         fmt.Sprintf("prelude%d", n),
		Postlude:        fmt.Sprintf("postlude%d", n),
		ParallelProgram: fmt.Sprintf("parallel program %d", n),
		ApparatusSpec: fmt.Sprintf(`--- !yaml%d
key: val`, n),
	}
}

func makeFakeComment(n int) *Comment {
	return &Comment{
		Text: fmt.Sprintf("comment%d", n),
	}
}

func makeFakeSupportingFile(n int) *SupportingFile {
	return &SupportingFile{
		Path:     fmt.Sprintf("file%d.tmpl", n),
		Contents: []byte(fmt.Sprintf("sample content %d", n)),
	}
}

func assertSnippetsEqual(t *testing.T, s1, s2 *CuratedSnippet) {
	assert.Equal(t, s1.User, s2.User, "Snippets different on User field")
	assert.Equal(t, s1.Status, s2.Status, "Snippets different on Status field")
	assert.Equal(t, s1.Language, s2.Language, "Snippets different on Language field")
	assert.Equal(t, s1.Package, s2.Package, "Snippets different on Package field")
	assert.Equal(t, s1.Title, s2.Title, "Snippets different on Title field")
	assert.Equal(t, s1.Code, s2.Code, "Snippets different on Code field")
	assert.Equal(t, s1.Prelude, s2.Prelude, "Snippets different on Prelude field")
	assert.Equal(t, s1.Postlude, s2.Postlude, "Snippets different on Postlude field")
	assert.Equal(t, s1.ApparatusSpec, s2.ApparatusSpec, "Snippets different on ApparatusSpec field")
	assertCommentsEqual(t, s1.Comments, s2.Comments)
	assertSupportingFilesEqual(t, s1.SupportingFiles, s2.SupportingFiles)
}

func assertCommentEqual(t *testing.T, comment1, comment2 *Comment) {
	assert.Equal(t, comment1.SnippetID, comment2.SnippetID, "Comments different on SnippetID")
	assert.Equal(t, comment1.Text, comment2.Text, "Comments different on Text")

	assert.Equal(t, comment1.Created, comment2.Created, "Comments different on Created")
	assert.Equal(t, comment1.CreatedBy, comment2.CreatedBy, "Comments different on CreatedBy")

	assert.Equal(t, comment1.Modified, comment2.Modified, "Comments different on Modified")
	assert.Equal(t, comment1.ModifiedBy, comment2.ModifiedBy, "Comments different on ModifiedBy")

	assert.Equal(t, comment1.Dismissed, comment2.Dismissed, "Comments different on Dismissed")
	assert.Equal(t, comment1.DismissedBy, comment2.DismissedBy, "Comments different on DismissedBy")
}

func assertCommentsEqual(t *testing.T, c1, c2 []*Comment) {
	if !assert.Equal(t, len(c1), len(c2), "Snippets different on number of comments") {
		return
	}

	for i := 0; i < len(c1); i++ {
		assertCommentEqual(t, c1[i], c2[i])
	}
}

func assertSupportingFilesEqual(t *testing.T, f1, f2 []*SupportingFile) {
	if !assert.Equal(t, len(f1), len(f2), "Snippets different on number of supporting files") {
		return
	}

	for i := 0; i < len(f1); i++ {
		assert.Equal(t, f1[i].Path, f2[i].Path, "Snippets different on Path for supporting file at index %d", i)
		assert.True(t, bytes.Equal(f1[i].Contents, f2[i].Contents), "Snippets different on Contents for supporting file at index %d", i)
	}
}

func pprint(data interface{}) {
	buf, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	log.Println(string(buf))
}
