package main

import (
	"fmt"
	"log"
	"path/filepath"
	"sync"

	"github.com/kiteco/kiteco/kite-golib/lexicalv0"
)

func makeServingCommand() *commandController {
	return newCommandController("/usr/bin/tensorflow_model_server",
		"--port=8500",
		"--rest_api_port=8501",
		"--tensorflow_intra_op_parallelism=8",
		"--tensorflow_inter_op_parallelism=8",
		"--monitoring_config_file=/tfserving-config/monitoring.config",
		"--model_config_file=/tfserving-config/model.config",
		"--batching_parameters_file=/tfserving-config/batching_parameters.config",
		"--enable_batching=true",
	)
}

func makeTuningCommand(repo string) *commandController {
	return newCommandController("make", "-f", "Makefile.tuned",
		fmt.Sprintf("LANG=%s", lexicalv0.AllLangsGroup.Name()),
		fmt.Sprintf("REPO_NAME=%s", repo),
		"clean-workspace", "initialize", "update_vocab", "train", "clean-workspace")
}

type tuneRequest struct {
	repo string
	swap bool
}

type state int64

const (
	stateServing state = iota
	stateTuning
	stateSwapping
	stateRestarting
	stateDeleting
)

func (s state) String() string {
	switch s {
	case stateServing:
		return "serving"
	case stateTuning:
		return "tuning"
	case stateSwapping:
		return "swapping models"
	case stateRestarting:
		return "restarting server"
	case stateDeleting:
		return "deleting model or repo"
	default:
		return "unknown state"
	}
}

type server struct {
	modelsDir       string
	repositoriesDir string
	tunedModelsDir  string

	tuneChan    chan tuneRequest
	restartChan chan struct{}

	m     sync.Mutex
	state state
}

func newServer(models, repositories, tunedModels string) *server {
	s := &server{
		modelsDir:       models,
		repositoriesDir: repositories,
		tunedModelsDir:  tunedModels,
		tuneChan:        make(chan tuneRequest),
		restartChan:     make(chan struct{}),
	}

	go s.tuneOrServe()

	return s
}

func (s *server) setState(st state) {
	s.m.Lock()
	defer s.m.Unlock()
	s.state = st
}

func (s *server) getState() state {
	s.m.Lock()
	defer s.m.Unlock()
	return s.state
}

func (s *server) checkAndSwapState(existing, new state) bool {
	s.m.Lock()
	defer s.m.Unlock()
	if s.state != existing {
		return false
	}
	s.state = new
	return true
}

func (s *server) tuneOrServe() {
	// Look for any saved active model state. If none exists, we use the default models
	active, err := s.loadActiveModel()
	if err == nil {
		// If an active models save was found, update the model versions to point to
		// the active model for each language.
		s.linkModel(active)
	}

	logdir := filepath.Join(s.tunedModelsDir, "logs")

	cmd := makeServingCommand()
	cmd.log(logdir, "tfserving-startup")
	err = cmd.start()
	if err != nil {
		log.Println("error starting serve command:", err)
		return
	}

	s.setState(stateServing)

	for {
		select {
		case <-s.restartChan:
			func() {
				if !s.checkAndSwapState(stateRestarting, stateRestarting) {
					log.Println("received request to restart, but in incorrest state, dropping")
					return
				}
				defer s.setState(stateServing)

				err = cmd.stop()
				if err != nil {
					log.Println("error stopping serve command:", err)
				}
				cmd = makeServingCommand()
				cmd.log(logdir, "tfserving-restart")
				err = cmd.start()
				if err != nil {
					log.Println("error starting serve command:", err)
				}
			}()

		case tuneReq := <-s.tuneChan:
			func() {
				if !s.checkAndSwapState(stateTuning, stateTuning) {
					log.Println("received request to tune, but in incorrest state, dropping")
					return
				}
				defer s.setState(stateServing)

				log.Println("received request to tune using repo", tuneReq.repo)
				err = cmd.stop()
				if err != nil {
					log.Println("error stopping serve command:", err)
				}

				cmd = makeTuningCommand(tuneReq.repo)
				cmd.log(logdir, fmt.Sprintf("tune-%s", tuneReq.repo))
				err := cmd.start()
				if err != nil {
					log.Println("error starting tuning command:", err)
				}

				err = cmd.wait()
				if err != nil {
					log.Println("error waiting for tuning command:", err)
				}

				if tuneReq.swap {
					func() {
						latest, err := s.latestTunedModelForRepository(tuneReq.repo)
						if err != nil {
							log.Println("error finding latest model:", err)
							return
						}

						err = s.linkModel(latest)
						if err != nil {
							log.Println("error finding latest model:", err)
							return
						}

						err = s.saveActiveModel()
						if err != nil {
							log.Println("error saving model swap state:", err)
							return
						}
					}()
				}

				cmd = makeServingCommand()
				cmd.log(logdir, "tfserving-post-tune")
				err = cmd.start()
				if err != nil {
					log.Println("error starting serve command:", err)
				}
			}()
		}
	}
}
