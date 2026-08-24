package httpapi

import (
	"net/http"
	"strconv"

	"task203-radarqc/internal/model"
	"task203-radarqc/internal/receive"
)

// handleCreateScan 创建体扫（元数据）。
func (s *Server) handleCreateScan(w http.ResponseWriter, r *http.Request) {
	var in receive.ScanMeta
	if err := decodeJSON(r, &in); err != nil {
		handleErr(w, model.NewInvalidInput("请求体解析失败：%v", err))
		return
	}
	scan, err := s.app.Scans.CreateScan(in)
	if err != nil {
		handleErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, scan)
}

// handleListScans 列出体扫（可按站点过滤）。
func (s *Server) handleListScans(w http.ResponseWriter, r *http.Request) {
	stationID := r.URL.Query().Get("station_id")
	list, err := s.app.Scans.List(stationID)
	if err != nil {
		handleErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, list)
}

// handleGetScan 查询体扫详情。
func (s *Server) handleGetScan(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	scan, err := s.app.Scans.Get(id)
	if err != nil {
		handleErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, scan)
}

// handleAppendGates 批量追加门数据。
func (s *Server) handleAppendGates(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var in struct {
		Gates []receive.GateInput `json:"gates"`
	}
	if err := decodeJSON(r, &in); err != nil {
		handleErr(w, model.NewInvalidInput("请求体解析失败：%v", err))
		return
	}
	if len(in.Gates) == 0 {
		handleErr(w, model.NewInvalidInput("gates 不能为空"))
		return
	}
	res, err := s.app.Scans.AppendGates(id, in.Gates)
	if err != nil {
		handleErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, res)
}

// handleListGates 查询门数据（支持 ?elevation= & ?label= 过滤）。
func (s *Server) handleListGates(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var elev *int
	if v := r.URL.Query().Get("elevation"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			handleErr(w, model.NewInvalidInput("elevation 必须是整数：%v", err))
			return
		}
		elev = &n
	}
	label := model.GateLabel(r.URL.Query().Get("label"))
	gates, err := s.app.Scans.ListGates(id, elev, label)
	if err != nil {
		handleErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, gates)
}

// handleFinalizeScan 完成接收 → 待标记。
func (s *Server) handleFinalizeScan(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	scan, err := s.app.Scans.Finalize(id)
	if err != nil {
		handleErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, scan)
}

// handleMarkScan 执行质量标记。
func (s *Server) handleMarkScan(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	res, err := s.app.Marks.Mark(id)
	if err != nil {
		handleErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, res)
}

// handleRecomputeScan 按现行规则重算标签。
func (s *Server) handleRecomputeScan(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	res, err := s.app.Marks.Recompute(id)
	if err != nil {
		handleErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, res)
}

// handleSealScan 封存体扫。
func (s *Server) handleSealScan(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	scan, err := s.app.Scans.Seal(id)
	if err != nil {
		handleErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, scan)
}

// handleScanSummary 查询扫描摘要。
func (s *Server) handleScanSummary(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	sum, err := s.app.Summaries.Get(id)
	if err != nil {
		handleErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, sum)
}

// handleScanEvidence 查询扫描证据（支持 ?label= 过滤）。
func (s *Server) handleScanEvidence(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	label := model.GateLabel(r.URL.Query().Get("label"))
	hits, err := s.app.Evidence.ListByScan(id, label)
	if err != nil {
		handleErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, hits)
}

// handleScanMarks 查询扫描标记历史。
func (s *Server) handleScanMarks(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	marks, err := s.app.Store().Marks.ListByScan(id)
	if err != nil {
		handleErr(w, err)
		return
	}
	if len(marks) > 1 {
		marks = marks[:1]
	}
	writeJSON(w, http.StatusOK, marks)
}
