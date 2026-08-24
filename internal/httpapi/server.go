// Package httpapi 提供 REST API 层：路由注册、JSON 编解码、错误映射与中间件。
// 所有业务路由统一以 /api 前缀暴露。
package httpapi

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"task203-radarqc/internal/model"
	"task203-radarqc/internal/service"
)

// Server HTTP API 服务。
type Server struct {
	app *service.App
	mux *http.ServeMux
}

// New 构造 API 服务。
func New(app *service.App) *Server {
	s := &Server{app: app, mux: http.NewServeMux()}
	s.routes()
	return s
}

// Handler 返回 http.Handler（带日志与恢复中间件）。
func (s *Server) Handler() http.Handler {
	return s.recoverMiddleware(s.logMiddleware(s.mux))
}

func (s *Server) routes() {
	// 站点
	s.mux.HandleFunc("POST /api/stations", s.handleCreateStation)
	s.mux.HandleFunc("GET /api/stations", s.handleListStations)
	s.mux.HandleFunc("GET /api/stations/{id}", s.handleGetStation)
	s.mux.HandleFunc("POST /api/stations/{id}/status", s.handleTransitionStation)
	s.mux.HandleFunc("POST /api/stations/{id}/calibrations", s.handleCreateCalibration)
	s.mux.HandleFunc("GET /api/stations/{id}/calibrations", s.handleListCalibrations)
	s.mux.HandleFunc("POST /api/calibrations/{id}/publish", s.handlePublishCalibration)

	// 体扫
	s.mux.HandleFunc("POST /api/scans", s.handleCreateScan)
	s.mux.HandleFunc("GET /api/scans", s.handleListScans)
	s.mux.HandleFunc("GET /api/scans/{id}", s.handleGetScan)
	s.mux.HandleFunc("POST /api/scans/{id}/gates", s.handleAppendGates)
	s.mux.HandleFunc("GET /api/scans/{id}/gates", s.handleListGates)
	s.mux.HandleFunc("POST /api/scans/{id}/finalize", s.handleFinalizeScan)
	s.mux.HandleFunc("POST /api/scans/{id}/mark", s.handleMarkScan)
	s.mux.HandleFunc("POST /api/scans/{id}/recompute", s.handleRecomputeScan)
	s.mux.HandleFunc("POST /api/scans/{id}/seal", s.handleSealScan)
	s.mux.HandleFunc("GET /api/scans/{id}/summary", s.handleScanSummary)
	s.mux.HandleFunc("GET /api/scans/{id}/evidence", s.handleScanEvidence)
	s.mux.HandleFunc("GET /api/scans/{id}/marks", s.handleScanMarks)

	// 规则
	s.mux.HandleFunc("POST /api/rules", s.handleCreateRule)
	s.mux.HandleFunc("GET /api/rules", s.handleListRules)
	s.mux.HandleFunc("GET /api/rules/{id}", s.handleGetRule)
	s.mux.HandleFunc("POST /api/rules/{id}/publish", s.handlePublishRule)
	s.mux.HandleFunc("POST /api/rules/{id}/retire", s.handleRetireRule)
	s.mux.HandleFunc("GET /api/rules/compare", s.handleCompareRules)

	// 统计与健康
	s.mux.HandleFunc("GET /api/stats", s.handleStats)
	s.mux.HandleFunc("GET /api/health", s.handleHealth)
}

// ---- 中间件 ----

func (s *Server) logMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		lw := &loggingWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(lw, r)
		log.Printf("%s %s -> %d (%s)", r.Method, r.URL.Path, lw.status, time.Since(start))
	})
}

func (s *Server) recoverMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				log.Printf("panic: %v", rec)
				writeError(w, http.StatusInternalServerError, model.NewInvalidInput("服务内部错误"))
			}
		}()
		next.ServeHTTP(w, r)
	})
}

type loggingWriter struct {
	http.ResponseWriter
	status int
}

func (w *loggingWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

// ---- 通用响应 ----

// writeJSON 输出 JSON 响应。
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("encode response: %v", err)
	}
}

// writeError 将业务错误映射为 HTTP 状态码。
func writeError(w http.ResponseWriter, status int, err error) {
	ae := model.AsAppError(err)
	if status == http.StatusInternalServerError {
		status = statusFor(ae.Code)
	}
	writeJSON(w, status, map[string]string{"error": ae.Code, "message": ae.Message})
}

// statusFor 业务错误码 → HTTP 状态码。
func statusFor(code string) int {
	switch code {
	case model.ErrCodeInvalidInput:
		return http.StatusBadRequest
	case model.ErrCodeNotFound:
		return http.StatusNotFound
	case model.ErrCodeConflict, model.ErrCodeForbidden, model.ErrCodeUnavailable:
		return http.StatusConflict
	default:
		return http.StatusInternalServerError
	}
}

// handleErr 统一错误处理：业务错误按码映射，其余 500。
func handleErr(w http.ResponseWriter, err error) {
	ae := model.AsAppError(err)
	writeJSON(w, statusFor(ae.Code), map[string]string{"error": ae.Code, "message": ae.Message})
}

// decodeJSON 解析请求体。
func decodeJSON(r *http.Request, v any) error {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	return dec.Decode(v)
}
