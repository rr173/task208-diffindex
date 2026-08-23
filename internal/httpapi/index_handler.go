package httpapi

import (
	"encoding/json"
	"net/http"
	"strconv"

	"task208-diffindex/internal/model"
)

func (s *Server) runIndex(w http.ResponseWriter, r *http.Request) {
	res, err := s.app.Index.Run(r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, res)
}

func (s *Server) listLattices(w http.ResponseWriter, r *http.Request) {
	ls, err := s.app.Index.ListCandidates(r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, ls)
}

func (s *Server) getLattice(w http.ResponseWriter, r *http.Request) {
	l, err := s.app.Index.GetCandidate(r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, l)
}

type confirmReq struct {
	LatticeID string `json:"lattice_id"`
}

func (s *Server) confirmLattice(w http.ResponseWriter, r *http.Request) {
	var req confirmReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, model.InvalidInputf("bad request body: %v", err))
		return
	}
	l, err := s.app.Index.Confirm(r.PathValue("id"), req.LatticeID)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, l)
}

func (s *Server) residualReport(w http.ResponseWriter, r *http.Request) {
	rep, err := s.app.Index.ResidualReport(r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, rep)
}

func (s *Server) missingReport(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	maxIndex := 3
	if v := q.Get("max_index"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			maxIndex = n
		}
	}
	minD := 1.0
	if v := q.Get("min_d"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil && f > 0 {
			minD = f
		}
	}
	rep, err := s.app.Index.MissingReport(r.PathValue("id"), maxIndex, minD)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, rep)
}
