package server

import (
	"sync"
	"testing"
	"time"
)

func TestThreadStore(t *testing.T) {
	t.Run("get returns empty for unknown session", func(t *testing.T) {
		s := NewThreadStore()
		if ts := s.Get("unknown"); ts != "" {
			t.Errorf("ts = %q, want empty", ts)
		}
	})

	t.Run("set and get", func(t *testing.T) {
		s := NewThreadStore()
		s.Set("sess-1", "123.456", "")
		if ts := s.Get("sess-1"); ts != "123.456" {
			t.Errorf("ts = %q, want %q", ts, "123.456")
		}
	})

	t.Run("set overwrites existing entry", func(t *testing.T) {
		s := NewThreadStore()
		s.Set("sess-1", "111.111", "")
		s.Set("sess-1", "222.222", "")
		if ts := s.Get("sess-1"); ts != "222.222" {
			t.Errorf("ts = %q, want %q", ts, "222.222")
		}
	})

	t.Run("multiple sessions are independent", func(t *testing.T) {
		s := NewThreadStore()
		s.Set("sess-1", "111.111", "")
		s.Set("sess-2", "222.222", "")
		if ts := s.Get("sess-1"); ts != "111.111" {
			t.Errorf("sess-1 ts = %q, want %q", ts, "111.111")
		}
		if ts := s.Get("sess-2"); ts != "222.222" {
			t.Errorf("sess-2 ts = %q, want %q", ts, "222.222")
		}
	})

	t.Run("clean removes old entries", func(t *testing.T) {
		s := NewThreadStore()
		s.Set("old", "111.111", "")
		s.Set("new", "222.222", "")

		// manually set old entry's timestamp
		s.mu.Lock()
		e := s.threads["old"]
		e.CreatedAt = time.Now().Add(-48 * time.Hour)
		s.threads["old"] = e
		s.mu.Unlock()

		s.CleanOlderThan(24 * time.Hour)

		if ts := s.Get("old"); ts != "" {
			t.Errorf("old entry should be cleaned, got %q", ts)
		}
		if ts := s.Get("new"); ts != "222.222" {
			t.Errorf("new entry should remain, got %q", ts)
		}
	})

	t.Run("clean does nothing when all entries are recent", func(t *testing.T) {
		s := NewThreadStore()
		s.Set("a", "111.111", "")
		s.Set("b", "222.222", "")

		s.CleanOlderThan(24 * time.Hour)

		if ts := s.Get("a"); ts != "111.111" {
			t.Errorf("a should remain, got %q", ts)
		}
		if ts := s.Get("b"); ts != "222.222" {
			t.Errorf("b should remain, got %q", ts)
		}
	})

	t.Run("get by thread_ts returns tmux target", func(t *testing.T) {
		s := NewThreadStore()
		s.Set("sess-1", "123.456", "mysession:0.0")
		s.Set("sess-2", "789.012", "other:1.0")

		target, ok := s.GetByThreadTS("123.456")
		if !ok {
			t.Fatal("expected ok=true")
		}
		if target != "mysession:0.0" {
			t.Errorf("target = %q, want %q", target, "mysession:0.0")
		}
	})

	t.Run("get by thread_ts returns false for unknown", func(t *testing.T) {
		s := NewThreadStore()
		s.Set("sess-1", "123.456", "mysession:0.0")

		_, ok := s.GetByThreadTS("999.999")
		if ok {
			t.Error("expected ok=false for unknown thread_ts")
		}
	})

	t.Run("get by thread_ts returns empty target when no tmux", func(t *testing.T) {
		s := NewThreadStore()
		s.Set("sess-1", "123.456", "")

		target, ok := s.GetByThreadTS("123.456")
		if !ok {
			t.Fatal("expected ok=true")
		}
		if target != "" {
			t.Errorf("target = %q, want empty", target)
		}
	})

	t.Run("concurrent access is safe", func(t *testing.T) {
		s := NewThreadStore()
		var wg sync.WaitGroup
		for i := range 100 {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				id := "sess-" + string(rune('A'+i%26))
				s.Set(id, "ts-"+id, "")
				s.Get(id)
			}(i)
		}
		wg.Wait()
	})
}
