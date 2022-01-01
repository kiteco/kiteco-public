package spyder

import (
	"encoding/json"
	"regexp"
	"strconv"

	"github.com/go-errors/errors"
)

var spyderVersionPattern = regexp.MustCompile("(\\d+)\\.(\\d+)\\.(\\d+)(.*)")

type condaPackageInfo struct {
	BaseURL     string `json:"base_url"`
	BuildNumber int32  `json:"build_number"`
	BuildString string `json:"build_string"`
	Channel     string `json:"channel"`
	DistName    string `json:"dist_name"`
	Name        string `json:"name"`
	Platform    string `json:"platform"`
	Version     string `json:"version"`
}

func (c *condaPackageInfo) version() ([]string, error) {
	matches := spyderVersionPattern.FindStringSubmatch(c.Version)
	if matches == nil {
		return nil, errors.Errorf("invalid spyder version %s", c.Version)
	}
	return matches[1:], nil
}

func (c *condaPackageInfo) majorVersion() int {
	parts, err := c.version()
	if err != nil {
		return 0
	}
	major, _ := strconv.Atoi(parts[0])
	return major
}

func (c *condaPackageInfo) isDevVersion() bool {
	parts, err := c.version()
	return err == nil && len(parts) == 4 && len(parts[3]) > 0
}

func parseCondaPackageList(data []byte) ([]condaPackageInfo, error) {
	var result []condaPackageInfo
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, errors.Errorf("error parsing conda output: %s", err.Error())
	}
	return result, nil
}
