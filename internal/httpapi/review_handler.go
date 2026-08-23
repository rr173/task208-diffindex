package httpapi

import "net/http"

func (s *Server) lockPeak(w http.ResponseWriter, r *http.Request) {
	p, err := s.app.Review.LockReference(r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, p)
}

func (s *Server) unlockPeak(w http.ResponseWriter, r *http.Request) {
	p, err := s.app.Review.UnlockReference(r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, p)
}

func (s *Server) excludePeak(w http.ResponseWriter, r *http.Request) {
	p, err := s.app.Review.Exclude(r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, p)
}

func (s *Server) restorePeak(w http.ResponseWriter, r *http.Request) {
	p, err := s.app.Review.Restore(r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, p)
}

func (s *Server) listConflicts(w http.ResponseWriter, r *http.Request) {
	ps, err := s.app.Review.Conflicts(r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, ps)
}

func (s *Server) listExcluded(w http.ResponseWriter, r *http.Request) {
	ps, err := s.app.Review.Excluded(r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, ps)
}
