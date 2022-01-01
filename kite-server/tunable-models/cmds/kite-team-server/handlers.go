package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/kiteco/kiteco/kite-server/tunable-models/cmds/internal/api"
)

func (s *server) handleList(w http.ResponseWriter, r *http.Request) {
	var models []servableModel
	models = append(models, s.defaultServableModels()...)

	tuned, err := s.tunedServableModels()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	models = append(models, tuned...)

	active, err := s.activeServableModel()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var resp api.ListModelResponse
	for _, model := range models {
		resp.Models = append(resp.Models, api.Model{
			Name:   model.Name,
			Active: active.Name == model.Name,
		})
	}

	repos, err := s.listTunableRepositories()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	for _, repo := range repos {
		resp.Repositories = append(resp.Repositories, repo.Name)
	}

	resp.Status = s.getState().String()

	writeJSON(w, resp)
}

func (s *server) handleDelete(w http.ResponseWriter, r *http.Request) {
	var req api.DeleteRequest
	err := readJSON(w, r, &req)
	if err != nil {
		return
	}

	var resp api.MessageResponse
	if !s.checkAndSwapState(stateServing, stateDeleting) {
		resp.Message = "server is currently busy (tuning, swapping, or deleting), please try again later"
		writeJSON(w, &resp)
		return
	}

	defer s.setState(stateServing)

	if req.Repository != "" {
		if !s.haveRepository(req.Repository) {
			resp.Message = fmt.Sprintf("no repository named '%s' found", req.Repository)
			writeJSON(w, &resp)
			return
		}

		s.deleteRepository(req.Repository)
		resp.Message = fmt.Sprintf("repository '%s' removed", req.Repository)
		writeJSON(w, &resp)
		return
	}

	if req.Model != "" {
		if req.Model == "default" {
			resp.Message = fmt.Sprintf("cannot delete default model '%s'", req.Model)
			writeJSON(w, &resp)
			return
		}
		activeModel, err := s.activeServableModel()
		if err != nil {
			resp.Message = fmt.Sprintf("error determining active models: %s", err)
			writeJSON(w, &resp)
			return
		}
		if activeModel.Name == req.Model {
			resp.Message = fmt.Sprintf("cannot delete an active model '%s'. please swap active model before deleting", activeModel.Name)
			writeJSON(w, &resp)
			return
		}
		if !s.haveModel(req.Model) {
			resp.Message = fmt.Sprintf("no model named '%s' found", req.Model)
			writeJSON(w, &resp)
			return
		}

		s.deleteTunedModel(req.Model)
		resp.Message = fmt.Sprintf("model '%s' removed", req.Model)
		writeJSON(w, &resp)
		return
	}

	resp.Message = fmt.Sprintf("either repo or model must be set to delete")
	writeJSON(w, &resp)
}

func (s *server) handleTune(w http.ResponseWriter, r *http.Request) {
	var req api.TuneModelRequest
	err := readJSON(w, r, &req)
	if err != nil {
		return
	}

	var resp api.MessageResponse
	if !s.haveRepository(req.Repository) {
		resp.Message = fmt.Sprintf("do not have requested repository: %s", req.Repository)
		writeJSON(w, &resp)
		return
	}

	if !s.checkAndSwapState(stateServing, stateTuning) {
		resp.Message = "server is currently busy (tuning, swapping, or deleting), please try again later"
		writeJSON(w, &resp)
		return
	}

	s.tuneChan <- tuneRequest{repo: req.Repository, swap: req.Swap}
	resp.Message = fmt.Sprintf("started tuning a new model using repository %s", req.Repository)
	writeJSON(w, &resp)
}

// we swap models for a particular language by symlinking an updated "version" to the tuned
// model. version "1" will always be the default model. we point "2" to the tuned model
// when we want to enable something other than the default model. the server needs to be restarted for
// changes to take effect, which is done here via restartChan.
func (s *server) handleSwap(w http.ResponseWriter, r *http.Request) {
	var req api.SwapModelRequest
	err := readJSON(w, r, &req)
	if err != nil {
		return
	}

	var resp api.MessageResponse
	if !s.checkAndSwapState(stateServing, stateSwapping) {
		resp.Message = "server is currently busy (tuning, swapping, or deleting), please try again later"
		writeJSON(w, &resp)
		return
	}

	servableByName, err := s.servableModelsByName()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	sm, ok := servableByName[req.SwapToModel]
	if !ok {
		resp.Message = fmt.Sprintf("model '%s' was not found", req.SwapToModel)
		writeJSON(w, &resp)
		return
	}

	active, err := s.activeServableModel()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = s.linkModel(sm)
	if err != nil {
		resp.Message = fmt.Sprintf("error swapping model from '%s' to '%s': %s",
			active.Name, sm.Name, err)
		writeJSON(w, &resp)
		return
	}

	err = s.saveActiveModel()
	if err != nil {
		resp.Message = fmt.Sprintf("error saving model swap state from '%s' to '%s': %s",
			active.Name, sm.Name, err)
		writeJSON(w, &resp)
		return
	}

	if !s.checkAndSwapState(stateSwapping, stateRestarting) {
		resp.Message = "server in unexpected state, aborting"
		writeJSON(w, &resp)
		return
	}

	s.restartChan <- struct{}{}

	resp.Message = fmt.Sprintf("swapped model from '%s' to '%s'", active.Name, sm.Name)
	writeJSON(w, &resp)
}

// --

func readJSON(w http.ResponseWriter, r *http.Request, obj interface{}) error {
	buf, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return err
	}
	err = json.Unmarshal(buf, obj)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return err
	}
	return nil
}

func writeJSON(w http.ResponseWriter, obj interface{}) error {
	w.Header().Set("Content-Type", "application/json")
	buf, err := json.Marshal(obj)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return err
	}
	_, err = w.Write(buf)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return err
	}
	return nil
}
