package community

import (
	"encoding/json"
	"log"
	"net/http"
)

const (
	// KiteProEnabled is the flag used to activate Kite Pro licensing system
	KiteProEnabled = "kite_pro_enabled"
)

var remoteSettingsValue = map[string]string{
	"kite_pro_enabled": "true",
}

var remoteSettingsList = []string{KiteProEnabled}

// RemoteSettingsList returns the list of flags that can be controlled remotely
func RemoteSettingsList() []string {
	return remoteSettingsList
}

// RemoteSettingsRequest is used as the payload for remote setting requests
type RemoteSettingsRequest struct {
	Settings  map[string]string `json:"flags"`
	UserID    int64             `json:"user_id"`
	InstallID string            `json:"install_id"`
	MachineID string            `json:"machine_id"`
}

// RemoteSettingsResponse is the payload return for RemoteSettingsResponse (the response can also be empty if nothing has changed)
type RemoteSettingsResponse map[string]string

func handleRemoteSettings(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	var t RemoteSettingsRequest
	err := decoder.Decode(&t)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	var response RemoteSettingsResponse
	for k, v := range remoteSettingsValue {
		if val, present := t.Settings[k]; !present || val != v {
			if response == nil {
				response = make(map[string]string)
			}
			response[k] = remoteSettingsValue[k]
		}
	}
	if len(response) == 0 {
		w.WriteHeader(http.StatusOK)
		return
	}
	data, err := json.Marshal(response)
	if err != nil {
		log.Println("Error while marshaling remote settings response : ", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if _, err = w.Write(data); err != nil {
		log.Println("Error while marshaling remote settings response : ", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)

}
