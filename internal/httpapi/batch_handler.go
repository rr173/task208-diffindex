package httpapi

import (
	"encoding/json"
	"net/http"

	"task208-diffindex/internal/model"
	"task208-diffindex/internal/service"
)

type createBatchReq struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func (s *Server) createBatch(w http.ResponseWriter, r *http.Request) {
	var req createBatchReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, model.InvalidInputf("bad request body: %v", err))
		return
	}
	b, err := s.app.Batches.Create(req.ID, req.Name)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, b)
}

func (s *Server) listBatches(w http.ResponseWriter, r *http.Request) {
	bs, err := s.app.Batches.List()
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, bs)
}

func (s *Server) getBatch(w http.ResponseWriter, r *http.Request) {
	b, err := s.app.Batches.Get(r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, b)
}

type geometryReq struct {
	WavelengthAngstrom    float64 `json:"wavelength_angstrom"`
	DetectorDistanceMM    float64 `json:"detector_distance_mm"`
	BeamCenterXMM         float64 `json:"beam_center_x_mm"`
	BeamCenterYMM         float64 `json:"beam_center_y_mm"`
	BeamCenterZMM         float64 `json:"beam_center_z_mm"`
	OscillationRangeDeg   float64 `json:"oscillation_range_deg"`
	DetectorTwoThetaDeg   float64 `json:"detector_two_theta_deg"`
}

func (s *Server) setGeometry(w http.ResponseWriter, r *http.Request) {
	var req geometryReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, model.InvalidInputf("bad request body: %v", err))
		return
	}
	g := model.ExperimentGeometry{
		WavelengthAngstrom:    req.WavelengthAngstrom,
		DetectorDistanceMM:    req.DetectorDistanceMM,
		BeamCenterXMM:         req.BeamCenterXMM,
		BeamCenterYMM:         req.BeamCenterYMM,
		BeamCenterZMM:         req.BeamCenterZMM,
		OscillationRangeDeg:   req.OscillationRangeDeg,
		DetectorTwoThetaDeg:   req.DetectorTwoThetaDeg,
	}
	b, err := s.app.Batches.SetGeometry(r.PathValue("id"), g)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, b)
}

func (s *Server) getGeometry(w http.ResponseWriter, r *http.Request) {
	g, err := s.app.Batches.GetGeometry(r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, g)
}

type peakItem struct {
	Seq       int     `json:"seq"`
	XMM       float64 `json:"x_mm"`
	YMM       float64 `json:"y_mm"`
	ZMM       float64 `json:"z_mm"`
	Intensity float64 `json:"intensity"`
}

type importPeaksReq struct {
	Peaks []peakItem `json:"peaks"`
}

func (s *Server) importPeaks(w http.ResponseWriter, r *http.Request) {
	var req importPeaksReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, model.InvalidInputf("bad request body: %v", err))
		return
	}
	inputs := make([]service.PeakInput, 0, len(req.Peaks))
	for _, p := range req.Peaks {
		inputs = append(inputs, service.PeakInput{Seq: p.Seq, XMM: p.XMM, YMM: p.YMM, ZMM: p.ZMM, Intensity: p.Intensity})
	}
	res, err := s.app.Batches.ImportPeaks(r.PathValue("id"), inputs)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, res)
}

func (s *Server) listPeaks(w http.ResponseWriter, r *http.Request) {
	ps, err := s.app.Batches.ListPeaks(r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, ps)
}
