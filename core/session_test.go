package core

import (
	"path/filepath"
	"sync"
	"testing"
)

func TestSessionManager_GetOrCreateActive(t *testing.T) {
	sm := NewSessionManager("")
	s1 := sm.GetOrCreateActive("user1")
	if s1 == nil {
		t.Fatal("expected non-nil session")
	} else {
		s2 := sm.GetOrCreateActive("user1")
		if s1.ID != s2.ID {
			t.Error("same user should get same active session")
		}

		s3 := sm.GetOrCreateActive("user2")
		if s3.ID == s1.ID {
			t.Error("different user should get different session")
		}
	}
}

func TestSessionManager_NewSession(t *testing.T) {
	sm := NewSessionManager("")
	s1 := sm.NewSession("user1", "chat-a")
	s2 := sm.NewSession("user1", "chat-b")

	if s1.ID == s2.ID {
		t.Error("new sessions should have different IDs")
	}
	if s1.Name != "chat-a" || s2.Name != "chat-b" {
		t.Error("session names should match")
	}

	active := sm.GetOrCreateActive("user1")
	if active.ID != s2.ID {
		t.Error("latest session should be active")
	}
}

func TestSessionManager_SwitchSession(t *testing.T) {
	sm := NewSessionManager("")
	s1 := sm.NewSession("user1", "first")
	s2 := sm.NewSession("user1", "second")

	if sm.ActiveSessionID("user1") != s2.ID {
		t.Error("active should be s2")
	}

	switched, err := sm.SwitchSession("user1", s1.ID)
	if err != nil {
		t.Fatalf("SwitchSession: %v", err)
	}
	if switched.ID != s1.ID {
		t.Error("should have switched to s1")
	}
	if sm.ActiveSessionID("user1") != s1.ID {
		t.Error("active should now be s1")
	}
}

func TestSessionManager_SwitchByName(t *testing.T) {
	sm := NewSessionManager("")
	sm.NewSession("user1", "alpha")
	sm.NewSession("user1", "beta")

	switched, err := sm.SwitchSession("user1", "alpha")
	if err != nil {
		t.Fatalf("SwitchSession by name: %v", err)
	}
	if switched.Name != "alpha" {
		t.Error("should have switched to alpha")
	}
}

func TestSessionManager_SwitchNotFound(t *testing.T) {
	sm := NewSessionManager("")
	sm.NewSession("user1", "only")

	_, err := sm.SwitchSession("user1", "nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent session")
	}
}

func TestSessionManager_ListSessions(t *testing.T) {
	sm := NewSessionManager("")
	sm.NewSession("user1", "a")
	sm.NewSession("user1", "b")
	sm.NewSession("user2", "c")

	list := sm.ListSessions("user1")
	if len(list) != 2 {
		t.Errorf("user1 should have 2 sessions, got %d", len(list))
	}

	list2 := sm.ListSessions("user2")
	if len(list2) != 1 {
		t.Errorf("user2 should have 1 session, got %d", len(list2))
	}
}

func TestSessionManager_SessionNames(t *testing.T) {
	sm := NewSessionManager("")
	sm.SetSessionName("agent-123", "my-chat")

	if got := sm.GetSessionName("agent-123"); got != "my-chat" {
		t.Errorf("got %q, want my-chat", got)
	}

	sm.SetSessionName("agent-123", "")
	if got := sm.GetSessionName("agent-123"); got != "" {
		t.Errorf("got %q, want empty after clear", got)
	}
}

func TestSessionManager_Persistence(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sessions.json")

	sm1 := NewSessionManager(path)
	sm1.NewSession("user1", "persisted")
	sm1.SetSessionName("agent-x", "custom-name")

	sm2 := NewSessionManager(path)
	list := sm2.ListSessions("user1")
	if len(list) != 1 {
		t.Fatalf("expected 1 session after reload, got %d", len(list))
	}
	if list[0].Name != "persisted" {
		t.Errorf("session name = %q, want persisted", list[0].Name)
	}
	if got := sm2.GetSessionName("agent-x"); got != "custom-name" {
		t.Errorf("session name after reload = %q, want custom-name", got)
	}
}

func TestSessionManagerRenameActiveSession(t *testing.T) {
	sm := NewSessionManager("")
	s := sm.NewSession("user1", "draft")
	if s.Name != "draft" {
		t.Fatalf("expected initial name, got %q", s.Name)
	}

	renamed := sm.RenameActiveSession("user1", "renamed")
	if renamed == nil || renamed.Name != "renamed" {
		t.Fatalf("expected renamed session, got %#v", renamed)
	}
}

func TestSessionManagerRenameSession(t *testing.T) {
	sm := NewSessionManager("")
	first := sm.NewSession("user1", "first")
	sm.NewSession("user1", "second")

	renamed := sm.RenameSession("user1", first.ID, "renamed")
	if renamed == nil || renamed.Name != "renamed" {
		t.Fatalf("expected first session to be renamed, got %#v", renamed)
	}
	if got := sm.ActiveSessionID("user1"); got == first.ID {
		t.Fatalf("renaming changed active session to %q", got)
	}
}

func TestSessionManagerDeleteSession(t *testing.T) {
	storePath := filepath.Join(t.TempDir(), "sessions.json")
	sm := NewSessionManager(storePath)
	first := sm.NewSession("user1", "first")
	active := sm.NewSession("user1", "active")

	deleted, err := sm.DeleteSession("user1", first.ID)
	if err != nil {
		t.Fatalf("DeleteSession: %v", err)
	}
	if deleted.ID != first.ID {
		t.Fatalf("deleted session = %q, want %q", deleted.ID, first.ID)
	}
	if got := sm.ListSessions("user1"); len(got) != 1 || got[0].ID != active.ID {
		t.Fatalf("remaining sessions = %#v, want active session only", got)
	}
	reloaded := NewSessionManager(storePath)
	if got := reloaded.ListSessions("user1"); len(got) != 1 || got[0].ID != active.ID {
		t.Fatalf("reloaded sessions = %#v, want active session only", got)
	}

	if _, err := sm.DeleteSession("user1", active.ID); err != errCannotDeleteActiveSession {
		t.Fatalf("delete active error = %v, want %v", err, errCannotDeleteActiveSession)
	}
}

func TestSessionManagerDeleteSessionClearsNameAfterFinalAgentReference(t *testing.T) {
	storePath := filepath.Join(t.TempDir(), "sessions.json")
	sm := NewSessionManager(storePath)
	first := sm.NewSession("user1", "first")
	active1 := sm.NewSession("user1", "active-1")
	second := sm.NewSession("user2", "second")
	active2 := sm.NewSession("user2", "active-2")
	first.AgentSessionID = "shared-agent"
	second.AgentSessionID = "shared-agent"
	sm.SetSessionName("shared-agent", "shared name")

	if _, err := sm.DeleteSession("user1", first.ID); err != nil {
		t.Fatalf("DeleteSession first: %v", err)
	}
	if got := sm.GetSessionName("shared-agent"); got != "shared name" {
		t.Fatalf("name after deleting one reference = %q, want retained", got)
	}
	if _, err := sm.DeleteSession("user2", second.ID); err != nil {
		t.Fatalf("DeleteSession second: %v", err)
	}
	if got := sm.GetSessionName("shared-agent"); got != "" {
		t.Fatalf("name after deleting final reference = %q, want cleared", got)
	}
	if got := sm.ActiveSessionID("user1"); got != active1.ID {
		t.Fatalf("user1 active session = %q, want %q", got, active1.ID)
	}
	if got := sm.ActiveSessionID("user2"); got != active2.ID {
		t.Fatalf("user2 active session = %q, want %q", got, active2.ID)
	}
	reloaded := NewSessionManager(storePath)
	if got := reloaded.GetSessionName("shared-agent"); got != "" {
		t.Fatalf("reloaded stale agent-session name = %q, want empty", got)
	}
}

func TestSessionManagerRenameByAgentSessionID(t *testing.T) {
	sm := NewSessionManager("")
	s := sm.NewSession("user1", "draft")
	s.AgentSessionID = "agent-123"

	sm.RenameByAgentSessionID("agent-123", "renamed")

	current := sm.GetOrCreateActive("user1")
	if current.Name != "renamed" {
		t.Fatalf("expected session to be renamed by agent session id, got %q", current.Name)
	}
}

func TestSession_TryLockUnlock(t *testing.T) {
	s := &Session{}
	if !s.TryLock() {
		t.Error("first TryLock should succeed")
	}
	if s.TryLock() {
		t.Error("second TryLock should fail")
	}
	s.Unlock()
	if !s.TryLock() {
		t.Error("TryLock after Unlock should succeed")
	}
}

func TestSession_History(t *testing.T) {
	s := &Session{}
	s.AddHistory("user", "hello")
	s.AddHistory("assistant", "hi there")
	s.AddHistory("user", "bye")

	all := s.GetHistory(0)
	if len(all) != 3 {
		t.Errorf("expected 3 entries, got %d", len(all))
	}

	last2 := s.GetHistory(2)
	if len(last2) != 2 {
		t.Errorf("expected 2 entries, got %d", len(last2))
	}
	if last2[0].Content != "hi there" {
		t.Errorf("expected 'hi there', got %q", last2[0].Content)
	}

	s.ClearHistory()
	if h := s.GetHistory(0); len(h) != 0 {
		t.Errorf("expected empty history after clear, got %d", len(h))
	}
}

func TestSession_ConcurrentHistory(t *testing.T) {
	s := &Session{}
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			s.AddHistory("user", "msg")
		}()
	}
	wg.Wait()
	if h := s.GetHistory(0); len(h) != 50 {
		t.Errorf("expected 50 entries, got %d", len(h))
	}
}
