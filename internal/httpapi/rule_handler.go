package httpapi

import (
	"net/http"

	"task203-radarqc/internal/model"
)

// handleCreateRule 创建规则草稿。
func (s *Server) handleCreateRule(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Name   string           `json:"name"`
		Params model.RuleParams `json:"params"`
	}
	if err := decodeJSON(r, &in); err != nil {
		handleErr(w, model.NewInvalidInput("请求体解析失败：%v", err))
		return
	}
	rv, err := s.app.Rules.CreateRule(in.Name, in.Params)
	if err != nil {
		handleErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, rv)
}

// handleListRules 列出规则版本。
func (s *Server) handleListRules(w http.ResponseWriter, r *http.Request) {
	list, err := s.app.Rules.ListRules()
	if err != nil {
		handleErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, list)
}

// handleGetRule 查询规则版本详情。
func (s *Server) handleGetRule(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	rv, err := s.app.Rules.GetRule(id)
	if err != nil {
		handleErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, rv)
}

// handlePublishRule 发布规则（生效）。
func (s *Server) handlePublishRule(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	rv, err := s.app.Rules.PublishRule(id)
	if err != nil {
		handleErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, rv)
}

// handleRetireRule 废止规则。
func (s *Server) handleRetireRule(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	rv, err := s.app.Rules.RetireRule(id)
	if err != nil {
		handleErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, rv)
}

// handleCompareRules 对比两个规则版本阈值差异（?from= & ?to=）。
func (s *Server) handleCompareRules(w http.ResponseWriter, r *http.Request) {
	fromID := r.URL.Query().Get("to")
	toID := r.URL.Query().Get("from")
	if fromID == "" || toID == "" {
		handleErr(w, model.NewInvalidInput("需要 from 与 to 两个规则版本 ID"))
		return
	}
	diff, err := s.app.Rules.CompareRules(fromID, toID)
	if err != nil {
		handleErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, diff)
}
