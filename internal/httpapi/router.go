// Package httpapi 提供 HTTP 层：路由注册、请求解析与 JSON 响应。
package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"

	"task208-diffindex/internal/model"
	"task208-diffindex/internal/service"
)

// Server HTTP 服务器。
type Server struct {
	app *service.App
}

// New 构造 HTTP 服务器。
func New(app *service.App) *Server { return &Server{app: app} }

// Handler 注册全部路由并返回处理器。
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()

	// 批次、几何与峰。
	mux.HandleFunc("POST /api/batches", s.createBatch)
	mux.HandleFunc("GET /api/batches", s.listBatches)
	mux.HandleFunc("GET /api/batches/{id}", s.getBatch)
	mux.HandleFunc("PUT /api/batches/{id}/geometry", s.setGeometry)
	mux.HandleFunc("GET /api/batches/{id}/geometry", s.getGeometry)
	mux.HandleFunc("POST /api/batches/{id}/peaks", s.importPeaks)
	mux.HandleFunc("GET /api/batches/{id}/peaks", s.listPeaks)

	// 索引、晶格候选、残差与缺峰。
	mux.HandleFunc("POST /api/batches/{id}/index", s.runIndex)
	mux.HandleFunc("GET /api/batches/{id}/lattices", s.listLattices)
	mux.HandleFunc("GET /api/lattices/{id}", s.getLattice)
	mux.HandleFunc("POST /api/batches/{id}/confirm", s.confirmLattice)
	mux.HandleFunc("GET /api/batches/{id}/residuals", s.residualReport)
	mux.HandleFunc("GET /api/batches/{id}/missing", s.missingReport)

	// 峰复核。
	mux.HandleFunc("POST /api/peaks/{id}/lock", s.lockPeak)
	mux.HandleFunc("POST /api/peaks/{id}/unlock", s.unlockPeak)
	mux.HandleFunc("POST /api/peaks/{id}/exclude", s.excludePeak)
	mux.HandleFunc("POST /api/peaks/{id}/restore", s.restorePeak)
	mux.HandleFunc("GET /api/batches/{id}/conflicts", s.listConflicts)
	mux.HandleFunc("GET /api/batches/{id}/excluded", s.listExcluded)

	// 索引版本。
	mux.HandleFunc("POST /api/batches/{id}/versions", s.publishVersion)
	mux.HandleFunc("GET /api/batches/{id}/versions", s.listVersions)
	mux.HandleFunc("GET /api/versions/{id}", s.getVersion)

	// 统计与健康。
	mux.HandleFunc("GET /api/stats", s.stats)
	mux.HandleFunc("GET /api/health", s.health)

	return mux
}

func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "service": "task208-diffindex"})
}

func (s *Server) stats(w http.ResponseWriter, r *http.Request) {
	st, err := s.app.Stats()
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, st)
}

// writeJSON 输出 JSON 响应。
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// writeError 把领域错误映射为 HTTP 状态码并输出。
func writeError(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	switch {
	case errors.Is(err, model.ErrNotFound):
		status = http.StatusNotFound
	case errors.Is(err, model.ErrInvalidInput):
		status = http.StatusBadRequest
	case errors.Is(err, model.ErrInvalidState), errors.Is(err, model.ErrConflict):
		status = http.StatusConflict
	case errors.Is(err, model.ErrSealed):
		status = http.StatusConflict
	case errors.Is(err, model.ErrInsufficientData):
		status = http.StatusUnprocessableEntity
	case errors.Is(err, model.ErrDuplicate):
		status = http.StatusConflict
	}
	writeJSON(w, status, map[string]string{"error": err.Error()})
}
