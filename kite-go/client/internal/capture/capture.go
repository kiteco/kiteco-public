package capture

import (
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"path/filepath"
	"runtime/pprof"
	"strings"
	"time"

	"github.com/gorilla/mux"

	"github.com/kiteco/kiteco/kite-go/client/component"
	"github.com/kiteco/kiteco/kite-go/userids"
	"github.com/kiteco/kiteco/kite-golib/errors"
)

const (
	defaultHTTPTimeout = 10 * time.Second
	previousLogsSuffix = "bak"
	allowUploads       = false
)

// Manager implements a component for capturing profiling information
type Manager struct {
	auth    component.AuthClient
	userIDs userids.IDs
	logFile string
}

// NewManager returns a new Manager
func NewManager() *Manager {
	return &Manager{}
}

// Name implements component.Core
func (m *Manager) Name() string {
	return "capture"
}

// Initialize implements component.Initializer
func (m *Manager) Initialize(opts component.InitializerOptions) {
	m.auth = opts.AuthClient
	m.userIDs = opts.UserIDs
	m.logFile = opts.Platform.LogFile
}

// RegisterHandlers implements component.Hanlders
func (m *Manager) RegisterHandlers(mux *mux.Router) {
	mux.HandleFunc("/clientapi/capture", m.handleCapture).Methods("GET")
	mux.HandleFunc("/clientapi/logupload", m.handleLogUpload).Methods("GET")
}

// UploadResponse contains the response data for uploads
type UploadResponse struct {
	URL string `json:"url"`
}

// handleCapture is the handler for /clientapi/capture and sends information about kited to the backend
func (m *Manager) handleCapture(w http.ResponseWriter, r *http.Request) {
	if !allowUploads {
		w.WriteHeader(http.StatusOK)
		return
	}
	respChan := make(chan *UploadResponse, 1)
	go uploadCapture(m.auth, m.userIDs.MachineID(), m.userIDs.InstallID(), respChan)
	resp := <-respChan
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// handleLogUpload is the handler for /clientapi/logupload and sends the client log to the backend
func (m *Manager) handleLogUpload(w http.ResponseWriter, r *http.Request) {
	if !allowUploads {
		w.WriteHeader(http.StatusOK)
		return
	}

	filename := strings.Join([]string{
		filepath.Base(m.logFile),
		time.Now().Format("2006-01-02_03-04-05-PM"),
		previousLogsSuffix,
	}, ".")
	raw, err := ioutil.ReadFile(m.logFile)
	if err != nil {
		log.Printf("error reading logs %s: %v", m.logFile, err)
		http.Error(w, "error reading logs", http.StatusInternalServerError)
		return
	}
	resp, err := upload(m.auth, "logupload", m.userIDs.MachineID(), m.userIDs.InstallID(), filename, raw, time.Now().UnixNano())
	if err != nil {
		log.Printf("error uploading logs %s: %v", m.logFile, err)
		http.Error(w, "error uploading logs", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.Printf("error encoding %s: %v", m.logFile, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func uploadCapture(client component.AuthClient, machineID, installID string, respChan chan *UploadResponse) {
	// catch any panics
	defer func() {
		if ex := recover(); ex != nil {
			log.Printf("panic uploading capture: %v", ex)
		}
	}()

	start := time.Now()
	ts := start.UnixNano()

	captureData := capture()

	resp, err := upload(client, "capture", machineID, installID, "capture", captureData, ts)
	if err != nil {
		log.Printf("could not upload %s profile: %s", "capture", err)
	}

	log.Printf("uploaded capture in %s\n", time.Since(start))

	respChan <- resp
}

func capture() []byte {
	// TODO:(hrysoula) do we want this?
	// runtime.GC() // get up-to-date statistics

	start := time.Now()

	var b []byte
	buf := bytes.NewBuffer(b)
	writer := zip.NewWriter(buf)
	defer writer.Close()

	memWriter, err := writer.Create("memprofile.txt")
	if err != nil {
		log.Println(err)
		writer.Close()
		return nil
	}
	var memBuf bytes.Buffer
	if err := pprof.WriteHeapProfile(&memBuf); err != nil {
		log.Println("could not write memory profile: ", err)
		writer.Close()
		return nil
	}
	if _, err := memWriter.Write(memBuf.Bytes()); err != nil {
		log.Println("could not add memory profile to archive: ", err)
		writer.Close()
		return nil
	}

	cpuWriter, err := writer.Create("cpuprofile.txt")
	if err != nil {
		log.Println(err)
		writer.Close()
		return nil
	}
	var cpuBuf bytes.Buffer
	if err := pprof.StartCPUProfile(&cpuBuf); err != nil {
		log.Println("could not start CPU profile: ", err)
		writer.Close()
		return nil
	}
	time.Sleep(30 * time.Second)
	pprof.StopCPUProfile()
	if _, err := cpuWriter.Write(cpuBuf.Bytes()); err != nil {
		log.Println("could not add cpu profile to archive: ", err)
		writer.Close()
		return nil
	}

	for _, profile := range pprof.Profiles() {
		profWriter, err := writer.Create(profile.Name() + ".txt")
		if err != nil {
			log.Println(err)
			writer.Close()
			return nil
		}
		var profBuf bytes.Buffer
		if err := profile.WriteTo(&profBuf, 2); err != nil {
			log.Printf("could not write %s profile: %s", profile.Name(), err)
			writer.Close()
			return nil
		}
		if _, err := profWriter.Write(profBuf.Bytes()); err != nil {
			log.Printf("could not add %s profile to archive: %s", profile.Name(), err)
			writer.Close()
			return nil
		}
	}

	if err := writer.Close(); err != nil {
		log.Println("could not close zip writer:", err)
	}

	log.Printf("capture took %s\n", time.Since(start))

	return buf.Bytes()
}

func upload(client component.AuthClient, endpoint, machineID, installID, filename string, dataBuf []byte, timestamp int64) (*UploadResponse, error) {
	// this is transmitted as a raw string (gzipped) so that we can
	// open the resulting log files directly in the web browser for S3
	var buf bytes.Buffer
	gzw := gzip.NewWriter(&buf)
	if _, err := gzw.Write(dataBuf); err != nil {
		return nil, errors.Errorf("error encoding: %v", err)
	}
	if err := gzw.Close(); err != nil {
		log.Printf("error closing gz: %v", err)
		return nil, err
	}

	vals := url.Values{}
	vals.Set("filename", filename)
	vals.Set("machineid", machineID)
	vals.Set("installid", installID)
	vals.Set("timestamp", fmt.Sprintf("%d", timestamp))
	ep, err := client.Parse("/" + endpoint + "?" + vals.Encode())
	if err != nil {
		return nil, errors.Errorf("error parsing URL /%s?%s: %v", endpoint, vals.Encode(), err)
	}

	req, err := client.NewRequest("POST", ep.String(), "text/plain", &buf)
	if err != nil {
		return nil, errors.Errorf("error creating request %s: %v", ep.String(), err)
	}

	req.Header.Set("Content-Encoding", "gzip")

	ctx, cancel := context.WithTimeout(context.Background(), defaultHTTPTimeout)
	defer cancel()

	resp, err := client.Do(ctx, req)
	if err != nil {
		return nil, errors.Errorf("error posting to %s: %v", ep.String(), err)
	}
	if resp.StatusCode != 200 {
		_ = resp.Body.Close()
		return nil, errors.Errorf("error posting %s: %s", ep.String(), resp.Status)
	}
	var ur UploadResponse
	dec := json.NewDecoder(resp.Body)
	err = dec.Decode(&ur)
	if err != nil {
		_ = resp.Body.Close()
		return nil, errors.Errorf("error decoding body %s: %v", ep.String(), err)
	}
	if err := resp.Body.Close(); err != nil {
		return nil, errors.Errorf("error closing response body %s: %v", ep.String(), err)
	}
	return &ur, nil
}
