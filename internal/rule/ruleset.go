package rule

import (
	"fmt"
	"time"

	"task203-radarqc/internal/model"
	"task203-radarqc/internal/store"
)

// Ruleset 规则版本管理：创建草稿 → 发布生效（自动废止旧版）→ 废止。
type Ruleset struct {
	store *store.Store
	now   func() time.Time
}

// NewRuleset 构造规则集管理器。
func NewRuleset(s *store.Store) *Ruleset {
	return &Ruleset{store: s, now: time.Now}
}

// CreateDraft 创建规则草稿。同一 name 下版本号自动递增。
func (r *Ruleset) CreateDraft(name string, params model.RuleParams) (*model.RuleVersion, error) {
	if name == "" {
		return nil, model.NewInvalidInput("规则名称不能为空")
	}
	if err := params.Validate(); err != nil {
		return nil, err
	}
	version, err := r.store.Rules.NextVersion(name)
	if err != nil {
		return nil, err
	}
	now := r.now().UTC()
	rv := &model.RuleVersion{
		ID:        fmt.Sprintf("rule-%d-%d", now.UnixNano(), version),
		Name:      name,
		Version:   version,
		Params:    params,
		Status:    model.RuleDraft,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := r.store.Rules.Create(rv); err != nil {
		return nil, err
	}
	return rv, nil
}

// Publish 发布草稿为生效；其他生效版本自动转废止。
func (r *Ruleset) Publish(id string) (*model.RuleVersion, error) {
	rv, err := r.store.Rules.Get(id)
	if err != nil {
		return nil, err
	}
	if rv.Status != model.RuleDraft {
		return nil, model.NewConflict("规则版本 %s 状态为 %s，仅草稿可发布", id, rv.Status)
	}
	if err := r.store.Rules.UpdateStatus(id, model.RuleEffective); err != nil {
		return nil, err
	}
	rv.Status = model.RuleEffective
	rv.UpdatedAt = r.now().UTC()
	return rv, nil
}

// Retire 废止生效规则（终态，不可恢复）。
func (r *Ruleset) Retire(id string) (*model.RuleVersion, error) {
	rv, err := r.store.Rules.Get(id)
	if err != nil {
		return nil, err
	}
	if rv.Status != model.RuleEffective {
		return nil, model.NewConflict("规则版本 %s 状态为 %s，仅生效规则可废止", id, rv.Status)
	}
	if err := r.store.Rules.UpdateStatus(id, model.RuleRetired); err != nil {
		return nil, err
	}
	rv.Status = model.RuleRetired
	rv.UpdatedAt = r.now().UTC()
	return rv, nil
}

// Effective 返回当前生效规则。
func (r *Ruleset) Effective() (*model.RuleVersion, error) {
	return r.store.Rules.Effective()
}

// List 列出全部规则版本。
func (r *Ruleset) List() ([]*model.RuleVersion, error) {
	return r.store.Rules.List()
}

// Get 按 ID 查询规则。
func (r *Ruleset) Get(id string) (*model.RuleVersion, error) {
	return r.store.Rules.Get(id)
}
