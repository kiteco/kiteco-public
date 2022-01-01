package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/jinzhu/gorm"
	"github.com/kiteco/kiteco/kite-go/annotate"
	"github.com/kiteco/kiteco/kite-go/community"
	"github.com/kiteco/kiteco/kite-go/curation"
	"github.com/kiteco/kiteco/kite-go/curation/handlers"
	"github.com/kiteco/kiteco/kite-go/curation/titleparser"
	"github.com/kiteco/kiteco/kite-go/lang"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythoncuration"
	"github.com/kiteco/kiteco/kite-go/web/webutils"
	"github.com/kiteco/kiteco/kite-golib/templateset"
)

// AppOptions represents parameters passed as input to the app constructor
type AppOptions struct {
	CodeExampleDB   gorm.DB
	AuthDB          gorm.DB
	Templates       *templateset.Set
	TitleValidator  *titleparser.TitleValidator
	ReferenceServer string
	DefaultUnified  bool
	FrontendDev     bool
	PythonImage     string
}

// App is houses all the components for the curation app.
type App struct {
	Users *community.UserManager
	Auth  *community.UserValidation

	Snippets *curation.CuratedSnippetManager
	Runs     *curation.RunManager
	Access   *accessManager

	userHandlers    *handlers.UserHandlers
	snippetHandlers *curatedSnippetHandlers

	db              gorm.DB
	templates       *templateset.Set
	referenceServer string
	pythonImage     string

	defaultUnified bool
	frontendDev    bool

	titleValidator *titleparser.TitleValidator
}

const authRedir = "/login"

// NewApp builds a new app from db objects and templates
func NewApp(opts AppOptions) *App {
	users := community.NewUserManager(opts.AuthDB, nil)
	auth := community.NewUserValidation(users)

	runs := curation.NewRunManager(opts.CodeExampleDB)
	snippets := curation.NewCuratedSnippetManager(opts.CodeExampleDB, runs)
	access := newAccessManager(opts.CodeExampleDB)

	return &App{
		Users:    users,
		Auth:     auth,
		Snippets: snippets,
		Runs:     runs,
		Access:   access,

		userHandlers:    handlers.NewUserHandlers(users, opts.Templates),
		snippetHandlers: newCuratedSnippetHandlers(snippets, access),

		db:              opts.CodeExampleDB,
		templates:       opts.Templates,
		referenceServer: opts.ReferenceServer,
		defaultUnified:  opts.DefaultUnified,
		pythonImage:     opts.PythonImage,
		titleValidator:  opts.TitleValidator,
		frontendDev:     opts.FrontendDev,
	}
}

// Migrate migrates underlying GORM databases
func (a *App) Migrate() error {
	if err := a.Users.Migrate(); err != nil {
		return err
	}
	if err := a.Snippets.Migrate(); err != nil {
		return err
	}
	if err := a.Runs.Migrate(); err != nil {
		return err
	}
	if err := a.Access.Migrate(); err != nil {
		return fmt.Errorf("error migrating access: %v", err)
	}
	return nil
}

// SetupRoutes registers handlers to the provided mux.
func (a *App) SetupRoutes(r *mux.Router) {
	r.HandleFunc("/login", a.userHandlers.HandleLogin)
	r.HandleFunc("/logout", a.userHandlers.HandleLogout)

	r.HandleFunc("/api/{language}/execute", a.Auth.WrapRedirect(authRedir, a.handleExecute)).Methods("POST")

	r.HandleFunc("/", a.Auth.WrapRedirect(authRedir, a.handlePackages)).Methods("GET")
	r.HandleFunc("/{language}/{package}/author", a.Auth.WrapRedirect(authRedir, a.handleAuthor)).Methods("GET")

	r.HandleFunc("/moderate", a.Auth.WrapRedirect(authRedir, a.handleModeratorView))
	r.HandleFunc("/examples/{language}/{package}", a.Auth.WrapRedirect(authRedir, a.handleUnifiedAuthor))

	// Get and update a snippet
	exampleRouter := r.PathPrefix("/api/example").Subrouter()
	exampleRouter.HandleFunc("/{snippetID}", a.Auth.WrapRedirect(authRedir, a.snippetHandlers.handleGet)).Methods("GET")
	exampleRouter.HandleFunc("/{snippetID}", a.Auth.WrapRedirect(authRedir, a.snippetHandlers.handleUpdate)).Methods("PUT")

	// Get comments for snippet and create new comment for snippet
	exampleRouter.HandleFunc("/{snippetID}/comments", a.Auth.WrapRedirect(authRedir, a.snippetHandlers.handleCreateComment)).Methods("POST")
	exampleRouter.HandleFunc("/{snippetID}/comments", a.Auth.WrapRedirect(authRedir, a.snippetHandlers.handleListComments)).Methods("GET")

	// Get snippets in language-package and create new snippet in language-package
	examplesRouter := r.PathPrefix("/api/{language}/{package}").Subrouter()
	examplesRouter.HandleFunc("/examples", a.Auth.WrapRedirect(authRedir, a.snippetHandlers.handleList)).Methods("GET")
	examplesRouter.HandleFunc("/examples", a.Auth.WrapRedirect(authRedir, a.snippetHandlers.handleCreate)).Methods("POST")
	examplesRouter.HandleFunc("/lockAndList", a.Auth.WrapRedirect(authRedir, a.handleLockAndListExamples))

	r.HandleFunc("/api/examples/query", a.Auth.WrapRedirect(authRedir, a.snippetHandlers.handleQuery))

	// Get, update, dismiss comment
	commentRouter := r.PathPrefix("/api/comment").Subrouter()
	commentRouter.HandleFunc("/{commentID}", a.Auth.WrapRedirect(authRedir, a.snippetHandlers.handleGetComment)).Methods("GET")
	commentRouter.HandleFunc("/{commentID}", a.Auth.WrapRedirect(authRedir, a.snippetHandlers.handleUpdateComment)).Methods("PUT")
}

// --

func (a *App) handleUnifiedAuthor(w http.ResponseWriter, r *http.Request) {
	user := community.GetUser(r)
	pkg := mux.Vars(r)["package"]

	// Don't let candidate curators access packages other than those
	// that start with `kite_interview`
	if user.Email == "curator_candidate@kite.com" && !strings.HasPrefix(pkg, "kite_interview") {
		webutils.ReportNotFound(w, "page not found")
		return
	}

	err := a.templates.Render(w, "unified.html", map[string]interface{}{
		"View":               "author",
		"UserEmail":          user.Email,
		"CodeExampleUrlBase": a.referenceServer,
		"Language":           mux.Vars(r)["language"],
		"Package":            pkg,
		"Dev":                a.frontendDev,
	})
	if err != nil {
		webutils.ReportError(w, err.Error())
	}
}

func (a *App) handleModeratorView(w http.ResponseWriter, r *http.Request) {
	err := a.templates.Render(w, "unified.html", map[string]interface{}{
		"View":               "moderate",
		"UserEmail":          community.GetUser(r).Email,
		"CodeExampleUrlBase": a.referenceServer,
		"Dev":                a.frontendDev,
	})
	if err != nil {
		webutils.ReportError(w, err.Error())
	}
}

func (a *App) handleAuthor(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	language := vars["language"]
	pkg := vars["package"]

	if language == "" {
		webutils.ReportError(w, "no language specified")
		return
	}

	if pkg == "" {
		webutils.ReportError(w, "no package specified")
		return
	}

	// Record access to this package
	user := community.GetUser(r)
	accessor := a.Access.currentAccessor(language, pkg)
	conflict := accessor != "" && accessor != user.Email
	if !conflict {
		if err := a.Access.record(language, pkg, user.Email); err != nil {
			webutils.ReportError(w, "error recording access log: %v", err)
			return
		}
	}

	// Execute template
	err := a.templates.Render(w, "author.html", map[string]interface{}{
		"Language":        language,
		"Package":         pkg,
		"AccessConflict":  conflict,
		"CurrentAccessor": accessor,
	})

	if err != nil {
		webutils.ReportError(w, err.Error())
	}
}

func (a *App) handleLockAndListExamples(w http.ResponseWriter, r *http.Request) {
	language := mux.Vars(r)["language"]
	pkg := mux.Vars(r)["package"]

	if language == "" {
		webutils.ReportError(w, "no language specified")
		return
	}

	if pkg == "" {
		webutils.ReportError(w, "no package specified")
		return
	}

	user := community.GetUser(r)
	lock, err := a.Access.acquireAccessLock(language, pkg, user.Email)
	if err != nil {
		webutils.ReportError(w, err.Error())
		return
	}
	exampleList, err := a.Snippets.List(language, pkg)
	if err != nil {
		webutils.ReportError(w, err.Error())
		return
	}
	type lockAndExamples struct {
		AccessLock *accessLock                `json:"accessLock"`
		Examples   []*curation.CuratedSnippet `json:"examples"`
	}
	sendResponse(w, &lockAndExamples{
		AccessLock: lock,
		Examples:   exampleList,
	})
}

func (a *App) handlePackages(w http.ResponseWriter, r *http.Request) {
	user := community.GetUser(r)

	if user.Email == "curator_candidate@kite.com" {
		webutils.ReportNotFound(w, "page not found")
		return
	}

	// Perform database query
	type packageLanguagePair struct {
		Package      string
		Language     string
		SnippetCount int
	}

	var pairs []packageLanguagePair
	rows, err := a.db.Table("CuratedSnippet").
		Select("COUNT(DISTINCT SnippetID) AS SnippetCount, Package, Language").
		Where("Status<>?", "deleted").
		Group("Package, Language").
		Order("SnippetCount DESC").
		Rows()
	if err != nil {
		webutils.ReportError(w, "error while attempting to select package list: %v", err)
		return
	}
	for rows.Next() {
		var pair packageLanguagePair
		rows.Scan(&pair.SnippetCount, &pair.Package, &pair.Language)
		if !strings.HasPrefix(pair.Package, "interview_") {
			pairs = append(pairs, pair)
		}
	}
	rows.Close()

	// Group by language
	type pkg struct {
		Name            string
		SnippetCount    int
		CurrentAccessor string
	}
	type language struct {
		Name         string
		Packages     []pkg
		SnippetTotal int // total over all packages
	}
	packagesByLanguage := make(map[string]*language)
	accessors := a.Access.allAccessors()
	for _, pair := range pairs {
		p := pkg{
			Name:            pair.Package,
			SnippetCount:    pair.SnippetCount,
			CurrentAccessor: accessors[pair.Package],
		}
		if lang, ok := packagesByLanguage[pair.Language]; ok {
			lang.Packages = append(lang.Packages, p)
			lang.SnippetTotal += p.SnippetCount
		} else {
			packagesByLanguage[pair.Language] = &language{
				Name:         pair.Language,
				Packages:     []pkg{p},
				SnippetTotal: p.SnippetCount,
			}
		}
	}

	// Render template
	err = a.templates.Render(w, "packages.html", map[string]interface{}{
		"CurrentUser": user.Email,
		"Languages":   packagesByLanguage,
		"DefaultOld":  !a.defaultUnified,
	})
	if err != nil {
		webutils.ReportError(w, err.Error())
	}
}

func (a *App) handleExecute(w http.ResponseWriter, r *http.Request) {
	// Parse submission
	title := r.PostFormValue("title")
	prelude := r.PostFormValue("prelude")
	code := r.PostFormValue("code")
	postlude := r.PostFormValue("postlude")

	snippetID, err := strconv.Atoi(r.PostFormValue("backendId"))
	if err != nil {
		webutils.ReportError(w, "invalid snippet ID: %s", r.PostFormValue("backendId"))
		return
	}
	log.Println("snippetID:", snippetID)

	// Parse apparatus spec
	spec, err := annotate.NewSpecFromCode(lang.Python, postlude)
	if err != nil {
		webutils.ReportError(w, "error parsing spec: %v", err)
		return
	}

	// Initialize so that json writes "[]" not "null"
	styleViolations := []styleViolation{}
	titleViolations := []*titleparser.TitleViolation{}
	switch lang.FromFilename(spec.SaveAs) {
	case lang.Python:
		// Autoformat code
		log.Println("Running autoformatter...")
		formatted, err := autoformatPythonSegments(prelude, code, postlude)
		if err == nil {
			prelude, code, postlude = formatted[0], formatted[1], formatted[2]
		} else {
			log.Printf("autoformat failed: %v. Ignoring\n", err)
		}

		// Check for style violations
		log.Println("Running style checks...")
		styleViolations, err = checkPythonStyle(prelude, code, annotate.RemoveSpecFromCode(lang.Python, postlude))
		if err != nil {
			log.Printf("style checks failed: %v. Ignoring\n", err)
		}
		if styleViolations == nil {
			styleViolations = make([]styleViolation, 0) // so that json writes "[]" not "null"
		}

		if a.titleValidator != nil {
			titleViolations, err = a.titleValidator.Validate(title, code, prelude)
			if err != nil {
				log.Println("title validator error:", err)
			}
		}
		if titleViolations == nil {
			titleViolations = make([]*titleparser.TitleViolation, 0) // so that json writes "[]" not "null"
		}
	default:
		log.Println("No validation implemented for non-python languages")
	}

	// Execute program
	log.Println("Running code...")
	regions := curation.RegionsFromCode(prelude, code, postlude)
	annotated, err := annotate.RunWithSpec(regions, spec, annotate.Options{
		Language:    lang.Python,
		DockerImage: a.pythonImage,
	})
	if err != nil {
		if annotated == nil || annotated.Stencil == nil {
			log.Println("annotated.Stencil was nil")
		} else {
			log.Println("Original code:\n" + annotated.Stencil.Original)
			log.Println("Runnable code:\n" + annotated.Stencil.Runnable)
			log.Println("Presention code:\n" + annotated.Stencil.Presentation)
		}
		webutils.ReportError(w, "error executing code example: %v", err)
		return
	}

	log.Println("Code example segments are:")
	for _, segment := range annotated.Segments {
		log.Printf("  %T\n", segment)
	}

	// Store the results
	_, err = a.Runs.Create(annotated.Raw, int64(snippetID), time.Now())
	if err != nil {
		webutils.ReportError(w, "error add result to database: %v", err)
		return
	}

	// Extract images (for backwards compatibility only)
	images := []*annotate.ImageAnnotation{}
	for _, annotation := range annotated.Segments {
		if image, ok := annotation.(*annotate.ImageAnnotation); ok {
			images = append(images, image)
		}
	}

	// Process the segments
	segments := curation.SegmentsFromAnnotations(annotated.Segments)
	preludeSegs, mainSegs, postludeSegs := pythoncuration.ResponseFromSegments(segments)

	// Construct result
	log.Println("Constructing result...")
	payload := map[string]interface{}{
		"style_violations":   styleViolations,
		"succeeded":          annotated.Raw.Succeeded,
		"formatted_prelude":  prelude,
		"formatted_code":     code,
		"formatted_postlude": postlude,
		"output":             annotated.Plain(),
		"title_violations":   titleViolations,
		"images":             images,
		"prelude_segments":   preludeSegs,
		"code_segments":      mainSegs,
		"postlude_segments":  postludeSegs,
	}

	buf, _ := json.MarshalIndent(payload, "", "  ")
	log.Println(string(buf))

	// Write json back to client
	encoder := json.NewEncoder(w)
	if err := encoder.Encode(payload); err != nil {
		webutils.ReportError(w, "error encoding json: %v", err)
	}
}
