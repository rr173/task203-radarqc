package httpapi

import (
	"net/http"
	"strconv"

	"task203-radarqc/internal/rule"
)

// handleStats 全局统计。
func (s *Server) handleStats(w http.ResponseWriter, r *http.Request) {
	st, err := s.app.Stats.Get()
	if err != nil {
		handleErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, st)
}

// handleHealth 健康检查：返回服务状态与生效规则描述。
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	rv, err := s.app.Rules.EffectiveRule()
	if err != nil {
		handleErr(w, err)
		return
	}
	resp := map[string]any{
		"status":  "ok",
		"service": "task203-radarqc",
	}
	if rv != nil {
		resp["effective_rule"] = rv.Name + " v" + strconv.Itoa(rv.Version)
		resp["thresholds"] = rule.DescribeThresholds(rv.Params)
	} else {
		resp["effective_rule"] = "none"
	}
	writeJSON(w, http.StatusOK, resp)
}
