package localfiles

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/kiteco/kiteco/kite-go/community"
)

// SettingsAPI is an http wrapper around the file-server application for use as an api
type SettingsAPI struct {
	store *ContentStore
	auth  *community.UserValidation
}

// FolderAPI holds all the information about a machines's local folder
type FolderAPI struct {
	Name    string                `valid:"required" json:"name"`
	Folders map[string]*FolderAPI `valid:"required" json:"folders"`
	Files   []*FileAPI            `valid:"required" json:"files"`
}

// FileAPI holds all the information about a machine's local file.
type FileAPI struct {
	Name          string `valid:"required" json:"name"`
	HashedContent string `json:"hashed_content"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// FileSystemAPI holds information about a machine's filesystem
type FileSystemAPI struct {
	Separator string     `valid:"required" json:"separator"`
	Root      *FolderAPI `valid:"required" json:"root"`
}

// NewSettingsAPI creates a new server with the provided content store.
func NewSettingsAPI(store *ContentStore, auth *community.UserValidation) *SettingsAPI {
	s := &SettingsAPI{
		store: store,
		auth:  auth,
	}

	return s
}

// NewFolderAPI creates a new FolderAPI with the provided folder name
func NewFolderAPI(name string) *FolderAPI {
	f := &FolderAPI{
		Name:    name,
		Folders: make(map[string]*FolderAPI),
	}
	return f
}

// NewFileAPI creates a new FileAPI
func NewFileAPI(name string, hashedContent string, createdAt time.Time, updatedAt time.Time) *FileAPI {
	l := &FileAPI{
		Name:          name,
		HashedContent: hashedContent,
		CreatedAt:     createdAt,
		UpdatedAt:     updatedAt,
	}
	return l
}

// NewFileSystemAPI creates a new FileSystemAPI from the root FolderAPI
func NewFileSystemAPI(root *FolderAPI) *FileSystemAPI {
	separator := `/`

	// Only convert to windows paths if the only folder is "windows"
	if root.Folders["windows"] != nil && len(root.Folders) == 1 && len(root.Files) == 0 {
		separator = `\`
		windows := root.Folders["windows"]
		root = NewFolderAPI("")

		// convert `/windows/c -> C:\`
		for k, f := range windows.Folders {
			if k != "unc" {
				name := strings.ToUpper(f.Name) + ":"
				f.Name = name
				root.Folders[name] = f
			}
		}

		// `/windows/unc/example -> \\example`
		unc := windows.Folders["unc"]
		if unc != nil {
			for _, f := range unc.Folders {
				name := `\\` + f.Name
				f.Name = name
				root.Folders[name] = f
			}
		}

	}

	l := &FileSystemAPI{
		Separator: separator,
		Root:      root,
	}

	return l
}

// SetupRoutes prepares handlers for the file-server API in the given Router.
func (s *SettingsAPI) SetupRoutes(mux *mux.Router) {
	mux.HandleFunc("/machines", s.auth.Wrap(s.HandleMachines)).Methods("GET")
	mux.HandleFunc("/machines/{machine}/files", s.auth.Wrap(s.HandleAPIFiles)).Methods("GET")
	mux.HandleFunc("/machines/{machine}", s.auth.Wrap(s.HandlePurgeFilesAPI)).Methods("DELETE")
}

// HandleMachines shows all machines.
func (s *SettingsAPI) HandleMachines(w http.ResponseWriter, r *http.Request) {
	machines, err := s.store.Files.Machines(community.GetUser(r).ID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	if err := json.NewEncoder(w).Encode(machines); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// HandleAPIFiles shows all existing file objects.
func (s *SettingsAPI) HandleAPIFiles(w http.ResponseWriter, r *http.Request) {
	machine := mux.Vars(r)["machine"]

	files, err := s.store.Files.List(community.GetUser(r).ID, machine)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	tree := CreateFolderTree(files)

	fs := NewFileSystemAPI(tree)

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(fs); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// HandlePurgeFilesAPI deletes all files for the provided user id and machine id.
func (s *SettingsAPI) HandlePurgeFilesAPI(w http.ResponseWriter, r *http.Request) {
	machine := mux.Vars(r)["machine"]
	err := s.store.Files.DeleteUserMachine(community.GetUser(r).ID, machine)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// CreateFolderTree creates a FolderAPI tree from a list of Files with full pathnames
func CreateFolderTree(files []*File) *FolderAPI {
	root := NewFolderAPI("")

	for _, f := range files {
		current := root
		parents := strings.Split(f.Name, "/")
		// remove the filename and the first empty string
		filename := parents[len(parents)-1]
		if len(parents) > 1 {
			parents = parents[1 : len(parents)-1]
		} else {
			parents = make([]string, 0)
		}

		for _, c := range parents {
			next := current.Folders[c]
			if next == nil {
				next = NewFolderAPI(c)
				current.Folders[c] = next
			}
			current = next
		}
		l := NewFileAPI(filename, f.HashedContent, f.CreatedAt, f.UpdatedAt)
		current.Files = append(current.Files, l)
	}

	return root
}
