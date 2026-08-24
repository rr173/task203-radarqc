package service

import (
	"task203-radarqc/internal/evidence"
	"task203-radarqc/internal/model"
	"task203-radarqc/internal/rule"
	"task203-radarqc/internal/store"
)

// RuleService 规则版本业务服务。
type RuleService struct {
	store   *store.Store
	ruleset *rule.Ruleset
	evidence *evidence.Recorder
	tracer  *evidence.Tracer
}

// CreateRule 创建规则草稿。
func (s *RuleService) CreateRule(name string, params model.RuleParams) (*model.RuleVersion, error) {
	return s.ruleset.CreateDraft(name, params)
}

// ListRules 列出规则版本。
func (s *RuleService) ListRules() ([]*model.RuleVersion, error) {
	return s.ruleset.List()
}

// GetRule 查询规则版本。
func (s *RuleService) GetRule(id string) (*model.RuleVersion, error) {
	return s.ruleset.Get(id)
}

// PublishRule 发布规则。
func (s *RuleService) PublishRule(id string) (*model.RuleVersion, error) {
	return s.ruleset.Publish(id)
}

// RetireRule 废止规则。
func (s *RuleService) RetireRule(id string) (*model.RuleVersion, error) {
	return s.ruleset.Retire(id)
}

// CompareRules 对比两个规则版本的阈值差异。
func (s *RuleService) CompareRules(fromID, toID string) (*rule.VersionDiff, error) {
	from, err := s.ruleset.Get(fromID)
	if err != nil {
		return nil, err
	}
	to, err := s.ruleset.Get(toID)
	if err != nil {
		return nil, err
	}
	return rule.CompareVersions(from, to), nil
}

// TraceFlip 分析两版规则对同一体扫门的标签翻转。
func (s *RuleService) TraceFlip(scanID string, fromParams, toParams model.RuleParams) (*evidence.VersionTrace, error) {
	return s.tracer.TraceFlip(scanID, fromParams, toParams)
}

// EffectiveRule 当前生效规则。
func (s *RuleService) EffectiveRule() (*model.RuleVersion, error) {
	return s.ruleset.Effective()
}
