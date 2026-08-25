package rule

import (
	"path/filepath"
	"sync"
	"testing"

	"task203-radarqc/internal/model"
	"task203-radarqc/internal/store"
)

// TestCreateDraftConcurrentAssignsUniqueContiguousVersions 验证并发创建同名规则
// 草稿时版本号分配的正确性：N 个并发请求全部成功，各自取得唯一且连续的版本号，
// 不应因读到相同“下一版本”而撞 UNIQUE(name, version)。
func TestCreateDraftConcurrentAssignsUniqueContiguousVersions(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "radarqc.db")
	s, err := store.Open(dbPath)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer s.Close()
	rs := NewRuleset(s)

	const n = 20
	results := make([]*model.RuleVersion, n)
	errs := make([]error, n)
	var wg sync.WaitGroup
	wg.Add(n)
	start := make(chan struct{})
	for i := 0; i < n; i++ {
		go func(idx int) {
			defer wg.Done()
			<-start
			rv, err := rs.CreateDraft("concurrent-rule", model.DefaultRuleParams())
			results[idx] = rv
			errs[idx] = err
		}(i)
	}
	close(start)
	wg.Wait()

	seen := make(map[int]bool, n)
	for i := 0; i < n; i++ {
		if errs[i] != nil {
			t.Fatalf("request %d failed: %v", i, errs[i])
		}
		v := results[i].Version
		if v < 1 || v > n {
			t.Fatalf("request %d got out-of-range version %d", i, v)
		}
		if seen[v] {
			t.Fatalf("duplicate version %d assigned to two requests", v)
		}
		seen[v] = true
		if results[i].Status != model.RuleDraft {
			t.Fatalf("request %d status = %s, want draft", i, results[i].Status)
		}
	}
	// 版本号应恰好覆盖 1..n，连续无缺漏。
	for v := 1; v <= n; v++ {
		if !seen[v] {
			t.Fatalf("version %d never assigned (gap in contiguous sequence)", v)
		}
	}
}
