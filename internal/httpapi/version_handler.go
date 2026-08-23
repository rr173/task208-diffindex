package httpapi

import "net/http"

func (s *Server) publishVersion(w http.ResponseWriter, r *http.Request) {
	v, err := s.app.Versions.Publish(r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, v)
}

func (s *Server) listVersions(w http.ResponseWriter, r *http.Request) {
	vs, err := s.app.Versions.List(r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, vs)
}

func (s *Server) getVersion(w http.ResponseWriter, r *http.Request) {
	v, err := s.app.Versions.Get(r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, v)
}
