package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"strings"

	"github.com/kiteco/kiteco/kite-golib/errors"
)

const (
	bundleEnvScript   = "env.sh"
	bundleRunScript   = "run.sh"
	bundleSetupScript = "setup.sh"
	binDir            = "bin"
	kiteMLDir         = "kite-python/kite_ml"
)

var (
	// defaultGoBinaries are included in every bundle.
	defaultGoBinaries = []string{
		"github.com/kiteco/kiteco/kite-go/cmds/azure-cluster",
	}

	kitecoRepoOrigins = []string{
		"git@github.com:kiteco/kiteco",
		"git@github.com:kiteco/kiteco.git",
		"https://github.com/kiteco/kiteco",
	}
)

type bundleConfig struct {
	GoBinaries   []string
	KitecoPaths  []string
	BundleKiteML bool
	InstallCUDA  bool
	DockerML     bool
}

func createBundle(bundleFile string, runScript string, conf bundleConfig) error {
	tmpDir, err := ioutil.TempDir("", "azure-cluster-bundle")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	bundleDirName := "bundle"
	bundlePath := path.Join(tmpDir, bundleDirName)

	if err := os.Mkdir(bundlePath, os.FileMode(0755)); err != nil {
		return err
	}

	if err := os.Mkdir(path.Join(bundlePath, binDir), os.FileMode(0755)); err != nil {
		return err
	}

	if err := copyFile(runScript, path.Join(bundlePath, bundleRunScript)); err != nil {
		return err
	}

	kiteRoot, err := getKiteRoot()
	if err != nil {
		return err
	}

	if conf.BundleKiteML {
		for _, p := range []string{"kite", "requirements.txt", "__init__.py", "setup.py"} {
			if err := copyPath(
				path.Join(kiteRoot, kiteMLDir, p),
				path.Join(bundlePath, "kiteco", kiteMLDir, p),
				[]string{".pyc"},
			); err != nil {
				return err
			}
		}
	}

	for _, p := range conf.KitecoPaths {
		if err := copyPath(path.Join(kiteRoot, p),
			path.Join(bundlePath, "kiteco", p), nil); err != nil {
			return err
		}
	}

	conf.GoBinaries = append(conf.GoBinaries, defaultGoBinaries...)

	cmds := make([]*exec.Cmd, 0, len(conf.GoBinaries))
	for _, bin := range conf.GoBinaries {
		outPath := path.Join(bundlePath, binDir, path.Base(bin))
		cmds = append(cmds, exec.Command("go", "build", "-o", outPath, bin))
	}

	_, err = runCmds(cmds)
	if err != nil {
		return err
	}

	hash, branch, err := getGitHashAndBranch()
	if err != nil {
		return err
	}

	if err := writeSetupFile(path.Join(bundlePath, bundleSetupScript), conf); err != nil {
		return err
	}

	envVars := map[string]string{
		"KITE_USE_AZURE_MIRROR": "0",
		"LD_LIBRARY_PATH":       "/usr/local/lib",
		"PATH":                  "/var/kite/bundle/bin:$PATH",
		"GIT_HASH":              hash,
		"GIT_BRANCH":            branch,
	}

	toForward, err := envVarsToForward()
	if err != nil {
		return err
	}

	for k, v := range toForward {
		envVars[k] = v
	}

	if err := writeEnvFile(path.Join(bundlePath, bundleEnvScript), envVars); err != nil {
		return err
	}

	_, err = runCmds([]*exec.Cmd{
		exec.Command("tar", "czvf", bundleFile, "-C", tmpDir, bundleDirName),
	})
	if err != nil {
		return err
	}

	return nil
}

// getKiteRoot gets the root of the kiteco repo
func getKiteRoot() (string, error) {
	originCmd := exec.Command("git", "ls-remote", "--get-url", "origin")
	out, err := originCmd.Output()
	if err != nil {
		return "", fmt.Errorf("error getting git origin URL: %v", err)
	}
	origin := strings.TrimSpace(string(out))

	var originOk bool
	for _, o := range kitecoRepoOrigins {
		if origin == o {
			originOk = true
			break
		}
	}

	if !originOk {
		return "", errors.Errorf("repo origin URL (%s) doesn't match expected (%s)", origin, strings.Join(kitecoRepoOrigins, " | "))
	}

	tlCmd := exec.Command("git", "rev-parse", "--show-toplevel")
	out, err = tlCmd.Output()
	if err != nil {
		return "", fmt.Errorf("error getting git top level: %v", err)
	}
	return strings.TrimSpace(string(out)), nil
}

// getGitHashAndBranch attempts to get the git commit hash and branch of the kiteco git repo by making calls to the
// git CLI.
func getGitHashAndBranch() (string, string, error) {
	hashCmd := exec.Command("git", "rev-parse", "HEAD")
	out, err := hashCmd.Output()
	if err != nil {
		return "", "", fmt.Errorf("error getting git hash: %v", err)
	}
	hash := strings.TrimSpace(string(out))

	branchCmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	out, err = branchCmd.Output()
	if err != nil {
		return "", "", fmt.Errorf("error getting git branch: %v", err)
	}
	branch := strings.TrimSpace(string(out))
	return hash, branch, nil
}

func writeSetupFile(filename string, conf bundleConfig) error {
	outf, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer outf.Close()

	return templates.RenderText(outf, "bundle-setup.sh", map[string]interface{}{
		"KiteML":   conf.BundleKiteML,
		"CUDA":     conf.InstallCUDA,
		"DockerML": conf.DockerML,
	})
}

func writeEnvFile(filename string, vars map[string]string) error {
	outf, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer outf.Close()

	for k, v := range vars {
		if _, err := fmt.Fprintf(outf, "export %s=%s\n", k, v); err != nil {
			return err
		}
	}
	return nil
}

// copyPath recursively copies a path to another. if excludeExtensions is not empty, regular files matching
// those extensions will be excluded.
func copyPath(src, dst string, excludeExtensions []string) error {
	fi, err := os.Lstat(src)
	if err != nil {
		return err
	}
	if fi.Mode()&os.ModeSymlink != 0 {
		return nil
	}
	if !fi.IsDir() {
		return copyFile(src, dst)
	}

	if err := os.MkdirAll(dst, os.ModePerm); err != nil {
		return err
	}

	files, err := ioutil.ReadDir(src)
	if err != nil {
		return err
	}

	for _, f := range files {
		var found bool
		for _, ex := range excludeExtensions {
			if strings.HasSuffix(f.Name(), ex) {
				found = true
				break
			}
		}
		if found {
			continue
		}

		// skip '.' directories node_modules and Kite.app, as they are large and needlessly inflate the bundle
		if strings.HasPrefix(f.Name(), ".") || f.Name() == "node_modules" || f.Name() == "Kite.app" {
			continue
		}

		if err := copyPath(path.Join(src, f.Name()), path.Join(dst, f.Name()), excludeExtensions); err != nil {
			return err
		}
	}

	return nil
}

func copyFile(src, dst string) error {
	if err := os.MkdirAll(path.Dir(dst), os.ModePerm); err != nil {
		return err
	}

	source, err := os.Open(src)
	if err != nil {
		return err
	}
	defer source.Close()

	destination, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destination.Close()

	if _, err = io.Copy(destination, source); err != nil {
		return fmt.Errorf("error copying %s to %s: %v", src, dst, err)
	}
	return nil
}
