package platform

import (
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const previousLogsSuffix = "bak"

func rotateLogs(logFile string, maxNumLogFiles int) {
	// make sure we do not accumulate log files
	logDir := filepath.Dir(logFile)
	logs, err := ioutil.ReadDir(logDir)
	if err != nil {
		log.Printf("error reading log dir %s: %v\n", logDir, err)
	}
	if len(logs) > maxNumLogFiles {
		log.Printf("have %d (> %d) log files, removing all\n", len(logs), maxNumLogFiles)
		for _, lf := range logs {
			path := filepath.Join(logDir, lf.Name())

			//do not delete the logfile itself as it's rotated later on
			if path == logFile {
				continue
			}

			if err := os.Remove(path); err != nil {
				log.Printf("error removing log file %s: %v\n", path, err)
			}
		}
	}

	// if old log file exists, copy to a new timestamped file for upload
	if _, err := os.Stat(logFile); !os.IsNotExist(err) {
		newName := strings.Join([]string{
			logFile,
			time.Now().Format("2006-01-02_03-04-05-PM"),
			previousLogsSuffix,
		}, ".")

		if err := os.Rename(logFile, newName); err != nil {
			log.Printf("error renaming logfile: %v\n", err)
		}
	}
}
