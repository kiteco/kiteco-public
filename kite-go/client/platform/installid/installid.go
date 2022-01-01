package installid

import (
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	uuid "github.com/satori/go.uuid"
)

// LoadInstallID returns the install id or creates one if one does not exist.
func LoadInstallID(kiteRoot string) (string, error) {
	file := filepath.Join(kiteRoot, "installid")
	bytes, err := ioutil.ReadFile(file)
	if err == nil {
		return string(bytes), nil
	}

	if !os.IsNotExist(err) {
		log.Println("Unknown error loading install id:", err.Error())
	}

	id, err := createInstallID()
	if err != nil {
		return "", err
	}

	err = ioutil.WriteFile(file, []byte(id), 0600)
	if err != nil {
		return "", err
	}

	return id, nil
}

func createInstallID() (string, error) {
	id, err := uuid.NewV4()
	if err != nil {
		return "", err
	}
	return id.String(), nil
}
