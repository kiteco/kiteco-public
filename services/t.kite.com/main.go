package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/fluent/fluent-logger-golang/fluent"
	"github.com/gorilla/mux"
	"github.com/xeipuuv/gojsonschema"
)

func main() {
	var port string
	flag.StringVar(&port, "port", ":9000", "port to listen on")

	var schemaDir string
	flag.StringVar(&schemaDir, "schemaDir", "/etc/schemas/", "directory to load schema files from")

	flag.Parse()

	fluentPortStr := os.Getenv("FLUENT_PORT")
	if fluentPortStr == "" {
		log.Fatal("Fatal error: FLUENT_PORT environment variable required")
	}

	fluentPort, err := strconv.Atoi(fluentPortStr)
	if err != nil {
		log.Fatal("Fatal error: FLUENT_PORT must be an integer")
	}

	fluentHost := os.Getenv("FLUENT_HOST")
	if fluentHost == "" {
		log.Fatal("Fatal error: FLUENT_HOST environment variable required")
	}

	schemaDir, err = filepath.Abs(schemaDir)
	if err != nil {
		log.Fatalf("Error reading schemaDir: %s", err)
	}

	schemaDirRef, err := os.Open(schemaDir)
	if err != nil {
		log.Fatalf("Error reading schemaDir: %s", err)
	}

	schemaFiles, err := schemaDirRef.Readdirnames(0)
	if err != nil {
		log.Fatalf("Error reading schemaDir: %s", err)
	}

	schemas := map[string]*gojsonschema.Schema{}

	schemaFileSuffix := ".schema.json"
	for _, s := range schemaFiles {
		if !strings.HasSuffix(s, schemaFileSuffix) {
			continue
		}
		loader := gojsonschema.NewReferenceLoader(filepath.Join("file://", schemaDir, s))
		schema, err := gojsonschema.NewSchema(loader)
		if err != nil {
			log.Fatalf("Error loading schema %s: %s", s, err)
		}
		schemas[strings.TrimSuffix(s, schemaFileSuffix)] = schema
		log.Printf("Loaded schema %s", s)
	}

	logger, err := fluent.New(fluent.Config{FluentPort: fluentPort, FluentHost: fluentHost})
	if err != nil {
		log.Fatalf("Error connecting to FluentD: %s", err)
	}

	log.Printf("Connected to FluentD: %s:%d", fluentHost, fluentPort)

	defer logger.Close()

	m := newManager(logger, schemas)
	r := mux.NewRouter()
	r.HandleFunc("/.ping", m.handlePing)
	r.HandleFunc("/{streamID}", m.handleStream)

	log.Println("listening on", port)
	log.Fatalln(http.ListenAndServe(port, r))
}

type manager struct {
	logger  *fluent.Fluent
	schemas map[string]*gojsonschema.Schema
}

func newManager(logger *fluent.Fluent, schemas map[string]*gojsonschema.Schema) *manager {
	return &manager{
		logger:  logger,
		schemas: schemas,
	}
}

var streams = map[string]bool{
	"kite_status":   true,
	"client_events": true,
	"kite_service":  true,
}

var ipRe = regexp.MustCompile("^[^,]+")

func (m *manager) handleStream(w http.ResponseWriter, r *http.Request) {
	streamID := mux.Vars(r)["streamID"]

	if !streams[streamID] {
		http.Error(w, "", http.StatusNotFound)
		return
	}

	bodyBytes, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Printf("Error reading body: %v", err)
		http.Error(w, "", http.StatusBadRequest)
		return
	}

	var body interface{}

	err = json.Unmarshal(bodyBytes, &body)
	if err != nil {
		log.Printf("Error decoding JSON body: %v, body=%s", err, string(bodyBytes))
		http.Error(w, "", http.StatusBadRequest)
		return
	}

	body.(map[string]interface{})["timestamp"] = time.Now().Format(time.RFC3339)

	forwarded := r.Header.Get("x-forwarded-for")
	if forwarded != "" {
		body.(map[string]interface{})["sourceIp"] = ipRe.FindString(forwarded)
	}

	bodyWithTS, _ := json.Marshal(body)

	tag := fmt.Sprintf("kite_metrics.valid.%s", streamID)
	message := map[string]interface{}{
		"body": bodyWithTS,
	}

	if schema, ok := m.schemas[streamID]; ok {
		loader := gojsonschema.NewGoLoader(body)
		result, err := schema.Validate(loader)
		if err != nil {
			http.Error(w, "", http.StatusBadRequest)
			message["error"] = err.Error()
			tag = fmt.Sprintf("kite_metrics.error.%s", streamID)
		} else if !result.Valid() {
			http.Error(w, "", http.StatusBadRequest)
			var strErrors = []string{}
			for _, e := range result.Errors() {
				strErrors = append(strErrors, e.String())
			}
			message["error"] = strings.Join(strErrors, "; ")
			tag = fmt.Sprintf("kite_metrics.invalid.%s", streamID)
		}
	}

	error := m.logger.PostWithTime(tag, time.Now(), message)
	if error != nil {
		panic(error)
	}
}

func (m *manager) handlePing(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}
