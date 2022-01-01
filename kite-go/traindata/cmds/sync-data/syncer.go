package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const (
	extension    = ".pickle"
	tmpExtension = ".part"
)

type syncer struct {
	host      string
	remoteDir string
	localDir  string
	isUsed    func(string, string) bool
}

func (s syncer) filesToSync() ([]string, error) {
	remote, err := remoteFiles(s.host, s.remoteDir)
	if err != nil {
		return nil, err
	}

	local, err := s.localFiles()
	if err != nil {
		return nil, err
	}

	localSet := make(map[string]struct{}, len(local))
	for _, f := range local {
		localSet[f] = struct{}{}
	}

	var toSync []string
	for _, f := range remote {
		if s.isUsed(s.host, f) {
			continue
		}

		if _, found := localSet[f]; !found {
			toSync = append(toSync, f)
		}
	}

	return toSync, nil
}

func (s syncer) localFiles() ([]string, error) {
	infos, err := ioutil.ReadDir(s.localDir)
	if err != nil {
		return nil, err
	}

	var files []string
	for _, info := range infos {
		name := info.Name()
		if strings.HasSuffix(name, extension) {
			parts := strings.Split(name, "--")
			files = append(files, parts[len(parts)-1])
		}
	}

	return files, nil
}

func (s syncer) syncFile(remotePath string, localPath string) error {
	cmd := exec.Command("scp", fmt.Sprintf("%s:%s", s.host, remotePath), localPath)
	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("error running scp: %s", string(out))
		tryRemove(localPath)
		return err
	}
	return nil
}

func (s syncer) SyncLoop() {
	for {
		files, err := s.filesToSync()
		if err != nil {
			log.Printf("error getting list of files to sync: %v", err)
		}
		if len(files) == 0 {
			log.Printf("%s: no files to sync", s.host)
		}

		for _, f := range files {
			remotePath := fmt.Sprintf("%s/%s", s.remoteDir, f)
			localPath := filepath.Join(s.localDir, localFilename(s.host, f))
			tmpPath := localPath + tmpExtension

			log.Printf("copying %s:%s to %s", s.host, remotePath, tmpPath)
			err := s.syncFile(remotePath, tmpPath)
			if err != nil {
				log.Printf("error copying %s: %v", f, err)
				continue
			}

			log.Printf("moving %s to %s", tmpPath, localPath)
			err = os.Rename(tmpPath, localPath)
			if err != nil {
				log.Printf("error moving %s to %s: %v", tmpPath, localPath, err)
				tryRemove(tmpPath)
				continue
			}
		}

		sleep()
	}
}

func localFilename(host, file string) string {
	return fmt.Sprintf("%s--%s", host, file)
}

func tryRemove(path string) {
	if err := os.Remove(path); err != nil {
		log.Printf("error removing %s: %v", path, err)
	}
}

func remoteFiles(host, remoteDir string) ([]string, error) {
	cmd := exec.Command("ssh", host, "ls", remoteDir)
	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("error running ssh ls: %s", string(out))
		return nil, err
	}

	var files []string
	for _, l := range strings.Split(string(out), "\n") {
		if len(l) == 0 || !strings.HasSuffix(l, extension) {
			continue
		}

		parts := strings.Split(l, "/")
		files = append(files, parts[len(parts)-1])
	}

	return files, nil
}

func sleep() {
	time.Sleep(10 * time.Second)
}
