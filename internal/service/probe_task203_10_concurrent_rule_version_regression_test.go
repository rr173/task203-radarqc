package service

import (
	"sync"
	"testing"

	"task203-radarqc/internal/model"
	"task203-radarqc/internal/store"
)

func TestBug10_ConcurrentRuleDraftsReceiveUniqueVersions(t *testing.T) {
	db, err := store.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	app, err := New(db)
	if err != nil {
		t.Fatal(err)
	}
	const workers = 20
	start := make(chan struct{})
	results := make(chan *model.RuleVersion, workers)
	errs := make(chan error, workers)
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			rv, err := app.Rules.CreateRule("parallel-draft", model.DefaultRuleParams())
			if err != nil {
				errs <- err
				return
			}
			results <- rv
		}()
	}
	close(start)
	wg.Wait()
	close(results)
	close(errs)
	for err := range errs {
		t.Fatalf("concurrent rule draft failed: %v", err)
	}
	versions := make(map[int]bool)
	for rv := range results {
		if versions[rv.Version] {
			t.Fatalf("duplicate rule version %d", rv.Version)
		}
		versions[rv.Version] = true
	}
	if len(versions) != workers {
		t.Fatalf("created %d versions, want %d", len(versions), workers)
	}
}
