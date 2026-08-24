package httpapi

import (
	"net/http"

	"task203-radarqc/internal/model"
	"task203-radarqc/internal/service"
)

// handleCreateStation 创建雷达站。
func (s *Server) handleCreateStation(w http.ResponseWriter, r *http.Request) {
	var in service.CreateStationInput
	if err := decodeJSON(r, &in); err != nil {
		handleErr(w, model.NewInvalidInput("请求体解析失败：%v", err))
		return
	}
	st, err := s.app.Stations.Create(in)
	if err != nil {
		handleErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, st)
}

// handleListStations 列出全部站点。
func (s *Server) handleListStations(w http.ResponseWriter, r *http.Request) {
	list, err := s.app.Stations.List()
	if err != nil {
		handleErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, list)
}

// handleGetStation 查询站点详情。
func (s *Server) handleGetStation(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	st, err := s.app.Stations.Get(id)
	if err != nil {
		handleErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, st)
}

// handleTransitionStation 站点状态流转。
func (s *Server) handleTransitionStation(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var in struct {
		Status string `json:"status"`
	}
	if err := decodeJSON(r, &in); err != nil {
		handleErr(w, model.NewInvalidInput("请求体解析失败：%v", err))
		return
	}
	st, err := s.app.Stations.Transition(id, model.StationStatus(in.Status))
	if err != nil {
		handleErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, st)
}

// handleCreateCalibration 创建校准草稿。
func (s *Server) handleCreateCalibration(w http.ResponseWriter, r *http.Request) {
	stationID := r.PathValue("id")
	var in struct {
		ZDRBias   float64 `json:"zdr_bias"`
		ZHOffset  float64 `json:"zh_offset"`
		RHOHVBias float64 `json:"rhohv_bias"`
		Note      string  `json:"note"`
	}
	if err := decodeJSON(r, &in); err != nil {
		handleErr(w, model.NewInvalidInput("请求体解析失败：%v", err))
		return
	}
	cal, err := s.app.Stations.CreateCalibration(stationID, in.ZDRBias, in.ZHOffset, in.RHOHVBias, in.Note)
	if err != nil {
		handleErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, cal)
}

// handleListCalibrations 列出站点校准版本。
func (s *Server) handleListCalibrations(w http.ResponseWriter, r *http.Request) {
	stationID := r.PathValue("id")
	list, err := s.app.Stations.ListCalibrations(stationID)
	if err != nil {
		handleErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, list)
}

// handlePublishCalibration 发布校准版本。
func (s *Server) handlePublishCalibration(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	cal, err := s.app.Stations.PublishCalibration(id)
	if err != nil {
		handleErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, cal)
}
