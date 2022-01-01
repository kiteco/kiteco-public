package filesystem

import (
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/kiteco/kiteco/kite-go/client/internal/localpath"
	"github.com/kiteco/kiteco/kite-go/client/readdir"
)

// LibraryType indicates how the library was discovered
type LibraryType int

// Enum of possible library types
const (
	SysPath LibraryType = iota
	VirtualEnvMatch
	Anaconda
	HomeDirWalk
	KiteLibraries
)

var (
	libTypes = []string{
		"SysPath",
		"VirtualEnvMatch",
		"Anaconda",
		"HomeDirWalk",
		"KiteLibraries",
	}
	virtualEnvLibraryPatterns = map[string][]string{
		"darwin": []string{
			"*/lib/python*/site-packages",
			"*/lib/python*/dist-packages",
		},
		"linux": []string{
			"*/lib/python*/site-packages",
			"*/lib/python*/dist-packages",
		},
		"windows": []string{
			"*\\Lib\\site-packages",
			"*\\Lib\\dist-packages",
		},
	}
	anacondaEnvLibraryPatterns = map[string][]string{
		"darwin": []string{
			"lib/python*/site-packages",
			"lib/python*/dist-packages",
		},
		"linux": []string{
			"lib/python*/site-packages",
			"lib/python*/dist-packages",
		},
		"windows": []string{
			"Lib\\site-packages",
			"Lib\\dist-packages",
		},
	}
)

func (lt LibraryType) String() string {
	return libTypes[lt]
}

// LibraryManager manages library discovery
type LibraryManager struct {
	// contains all library dirs found mapped to their type
	dirs sync.Map
	// contains counts of library types used in a build
	libTypesUsed map[string]int
	m            sync.Mutex

	homeDir    string
	kiteLibDir string
	walkLibs   []string

	debug bool
}

var (
	virtualEnvDirs = []string{
		"Envs",
		".virtualenvs",
	}
)

// NewLibraryManager creates a new LibraryManager
func NewLibraryManager(homeDir, kiteDir string, walkLibs []string) *LibraryManager {
	m := &LibraryManager{
		libTypesUsed: make(map[string]int),
		homeDir:      homeDir,
		kiteLibDir:   filepath.Join(kiteDir, "libraries"),
		walkLibs:     walkLibs,
	}

	// set up libraries dir
	err := os.MkdirAll(m.kiteLibDir, os.ModePerm)
	if err != nil {
		log.Println(err)
	}

	// add libs found in sys.Path
	m.sysPathLibs()

	// add libs found in conda base dir
	condaLibs, err := m.anacondaLibs()
	if err != nil {
		log.Println("error getting anaconda libraries:", err)
	}
	for _, lib := range condaLibs {
		canonPath, err := canonicalizePath(lib)
		if err != nil {
			log.Println(err)
			continue
		}
		m.dirs.Store(canonPath, Anaconda)
		m.logf("Adding conda lib: %s", canonPath)
	}

	// add libs found in home directory
	for _, lib := range m.walkLibs {
		// libs are already canonicalized
		// do not overwrite type if previously found
		m.dirs.LoadOrStore(lib, HomeDirWalk)
		m.logf("Adding home dir lib: %s", lib)
	}

	// add libs specified by the user
	m.kiteLibs()

	return m
}

// AddProject adds virtualenvs to the library manager found from the project paths
func (m *LibraryManager) AddProject(projectPaths []string) {
	var localPaths []string
	projectPathMap := make(map[string]bool)
	for _, p := range projectPaths {
		// project paths are canonicalized
		p = slashed(p)
		localPath, err := localpath.FromUnix(p)
		if err != nil {
			log.Println(err)
			continue
		}
		projectPathMap[localPath] = true
		localPaths = append(localPaths, localPath)
	}
	// add libs found in virtualenvs
	for _, v := range virtualEnvDirs {
		fpath := filepath.Join(m.homeDir, v)
		if _, ok := projectPathMap[fpath]; !ok {
			localPaths = append(localPaths, fpath)
		}
	}
	for _, lib := range findMatchingPaths(localPaths, virtualEnvLibraryPatterns[runtime.GOOS]) {
		canonPath, err := canonicalizePath(lib)
		if err != nil {
			log.Println(err)
			continue
		}
		// overwrite if previously found
		m.dirs.Store(canonPath, VirtualEnvMatch)
		m.logf("Adding virtualenv lib: %s", canonPath)
	}
}

// Dirs implements the LibraryManager interface
func (m *LibraryManager) Dirs() []string {
	var dirs []string
	m.dirs.Range(func(fpath, _ interface{}) bool {
		dirs = append(dirs, fpath.(string))
		return true
	})
	return dirs
}

// Stats implements the LibraryManager interface
func (m *LibraryManager) Stats() map[string]int {
	return m.libTypesUsed
}

// MarkUsed increments the count for the type of library used
func (m *LibraryManager) MarkUsed(fpath string) {
	m.logf("Marking %s used", fpath)
	libType, ok := m.dirs.Load(fpath)
	if !ok {
		return
	}
	m.m.Lock()
	defer m.m.Unlock()
	m.libTypesUsed[libType.(LibraryType).String()]++
}

// --

func (m *LibraryManager) sysPathLibs() {
	paths := make(map[string]bool)

	for _, python := range []string{"python3", "python2", "python"} {
		p, err := exec.LookPath(python)
		if err != nil {
			continue
		}
		cmd := exec.Command(p, "-c", "import sys; print(\",\".join(sys.path))")
		// Try to make sure that a window doesn't pop up when this happens
		cmd.SysProcAttr = attributes
		cmd.Dir = m.homeDir
		out, err := cmd.CombinedOutput()
		if err != nil {
			log.Println(err)
			continue
		}
		for _, lib := range strings.Split(string(out), ",") {
			paths[lib] = true
		}
	}
	for lib := range paths {
		if lib == "" {
			// skip "", which is used by syspath to indicate current working dir
			continue
		}
		canonPath, err := canonicalizePath(lib)
		if err != nil {
			log.Println(err)
			continue
		}
		m.dirs.Store(canonPath, SysPath)
		m.logf("Adding sys path lib: %s", canonPath)
	}
}

func (m *LibraryManager) anacondaLibs() ([]string, error) {
	var paths []string
	p, err := exec.LookPath("conda")
	if err != nil {
		return paths, err
	}
	cmd := exec.Command(p, "info", "--envs")
	// Try to make sure that a window doesn't pop up when this happens
	cmd.SysProcAttr = attributes
	cmd.Dir = m.homeDir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return paths, err
	}
	envs := parseAnacondaEnvs(string(out))
	return findMatchingPaths(envs, anacondaEnvLibraryPatterns[runtime.GOOS]), nil
}

func (m *LibraryManager) kiteLibs() {
	paths := make(map[string]bool)
	for _, p := range readdir.List(m.kiteLibDir) {
		fpath := filepath.Join(m.kiteLibDir, p.Path)
		fi, err := os.Lstat(fpath)
		if err != nil {
			log.Println(err)
			continue
		}
		if fi.Mode()&os.ModeSymlink != 0 {
			linkPath, err := filepath.EvalSymlinks(fpath)
			if err != nil {
				log.Println(err)
				continue
			}
			absPath, err := filepath.Abs(linkPath)
			if err != nil {
				log.Println(err)
				continue
			}
			paths[absPath] = true
		}
	}
	for lib := range paths {
		canonPath, err := canonicalizePath(lib)
		if err != nil {
			log.Println(err)
			continue
		}
		// do not overwrite type if previously found
		m.dirs.LoadOrStore(canonPath, KiteLibraries)
		log.Printf("Adding kite libraries lib: %s", canonPath)
	}
}

// --

func parseAnacondaEnvs(output string) []string {
	var envs []string
	output = strings.TrimSpace(output)
	for _, line := range strings.Split(strings.Replace(output, "\r\n", "\n", -1), "\n") {
		if strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) > 0 {
			envs = append(envs, fields[len(fields)-1])
		}
	}
	return envs
}

// findMatchingPaths finds paths in the given dirs that match the given patterns.
// For each dir in dirs, it uses glob to search for the patterns. The patterns
// are shell file name patterns.
func findMatchingPaths(dirs, patterns []string) []string {
	var paths []string
	for _, dir := range dirs {
		for _, pattern := range patterns {
			projectPattern := filepath.Join(dir, pattern)
			matches, err := filepath.Glob(projectPattern)
			if err != nil {
				log.Println(err)
				continue
			}
			for _, match := range matches {
				paths = append(paths, match)
			}
		}
	}
	return paths
}

// --

func (m *LibraryManager) logf(msg string, objs ...interface{}) {
	if m.debug {
		log.Printf("!! "+msg, objs...)
	}
}
