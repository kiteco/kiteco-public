package spyder

import (
	"fmt"
	"io/ioutil"
	"regexp"
	"runtime"
	"strings"

	"github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/fs"
	"github.com/kiteco/kiteco/kite-golib/errors"
)

const (
	// ID is the id of the plugin, it's used by the plugin manager for the check of optimal settings
	ID   = "spyder"
	name = "Spyder IDE"
)

var errConfigFileNotFound = errors.Errorf("Spyder config file not found")
var errConfigKeyNotFound = errors.Errorf("Spyder config key not found")
var lineFeedPattern = regexp.MustCompile("\r\n|\n")
var lineFeed string

func init() {
	if runtime.GOOS == "windows" {
		lineFeed = "\r\n"
	} else {
		lineFeed = "\n"
	}
}

// isKiteEnabled returns if Kite is enabled in the spyder config file
func isKiteEnabled(configFilePath string) bool {
	value, err := getSpyderConfigValue(configFilePath, "kite", "enable")
	return err == nil && value == "True"
}

func setKiteEnabled(configFilePath string, enableKite bool) error {
	value := "False"
	if enableKite {
		value = "True"
	}
	return setSpyderConfigValue(configFilePath, "kite", "enable", value)
}

func getSpyderConfigValue(configFilePath, section, key string) (string, error) {
	configData, err := loadSpyderConfig(configFilePath)
	if err != nil {
		return "", err
	}

	sectionStart := findSectionOffset(configData, section)
	if sectionStart == -1 {
		return "", errConfigKeyNotFound
	}

	value, _ := findSpyderConfigValue(configData, key, sectionStart)
	return value, nil
}

// setSpyderConfigValue takes a very simple approach to minimize changes to the ini file to avoid
// unnecessary changes
// github.com/go-ini/ini isn't able to handle complex values without modifications on save
func setSpyderConfigValue(configFilePath, section, key, value string) error {
	configData, err := loadSpyderConfig(configFilePath)
	if err != nil {
		return err
	}

	configLine := fmt.Sprintf("%s = %s%s", key, value, lineFeed)

	// locate section
	sectionStart := findSectionOffset(configData, section)
	if sectionStart == -1 {
		// section not found, insert and return
		configData += lineFeed + lineFeed + fmt.Sprintf("[%s]", section) + lineFeed + configLine
		return ioutil.WriteFile(configFilePath, []byte(configData), 0600)
	}

	// update existing line and save data into file
	_, lineRange := findSpyderConfigValue(configData, key, sectionStart)
	configData = configData[:lineRange[0]] + configLine + configData[lineRange[1]:]
	return ioutil.WriteFile(configFilePath, []byte(configData), 0600)

}

// loadSpyderConfig returns the data of the file or an error
func loadSpyderConfig(configFilePath string) (string, error) {
	if !fs.FileExists(configFilePath) {
		return "", errConfigFileNotFound
	}

	bytes, err := ioutil.ReadFile(configFilePath)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// findSectionOffset returns the start of the kite section in the config data
// it returns -1 if the section wasn't found
func findSectionOffset(configData, section string) int {
	return strings.Index(configData, fmt.Sprintf("[%s]", section))
}

// findSpyderConfigLine returns the config value and the start and end offsets of the line in the config data
// if the value wasn't found, then an empty value and the offset where it should be inserted, is returned
func findSpyderConfigValue(configData, key string, sectionStart int) (string, []int) {
	linePrefix := key + " = "

	configKeyLineOffset := -1
	sectionValuesOffset := nextLineStart(configData, sectionStart)
	for lineStart := sectionValuesOffset; lineStart != -1; lineStart = nextLineStart(configData, lineStart) {
		if configData[lineStart:lineStart+1] == "[" {
			// start of next section, key not found yet
			break
		}
		if strings.HasPrefix(configData[lineStart:], linePrefix) {
			configKeyLineOffset = lineStart
			break
		}
	}

	if configKeyLineOffset == -1 {
		return "", []int{sectionValuesOffset, sectionValuesOffset}
	}

	// update the existing config line
	eol := nextLineStart(configData, configKeyLineOffset)
	if eol == -1 {
		// section at end of file
		eol = len(configData)
	}

	configValue := strings.TrimSpace(configData[configKeyLineOffset+len(linePrefix) : eol])
	return configValue, []int{configKeyLineOffset, eol}
}

func nextLineStart(data string, start int) int {
	if start <= 0 || start >= len(data) {
		return -1
	}

	pos := lineFeedPattern.FindStringIndex(data[start:])
	if pos == nil || (start+pos[1]) == len(data) {
		return -1
	}

	// end of match is start of next line
	return start + pos[1]
}
