package main

import (
	"testing"
	"time"
)

func TestIsCommandAllowed(t *testing.T) {
	// Test with specific allowed commands
	s := &ShellServer{
		allowedCommands:  []string{"ls", "echo", "cat"},
		allowAllCommands: false,
	}

	tests := []struct {
		command string
		allowed bool
	}{
		{"ls", true},
		{"ls -la", true},
		{"echo hello", true},
		{"cat file.txt", true},
		{"rm file.txt", false},
		{"sudo ls", false},
		{"", false},
	}

	for _, test := range tests {
		result := s.isCommandAllowed(test.command)
		if result != test.allowed {
			t.Errorf("isCommandAllowed(%q) = %v, want %v", test.command, result, test.allowed)
		}
	}

	// Test with all commands allowed
	sAll := &ShellServer{
		allowedCommands:  []string{},
		allowAllCommands: true,
	}

	for _, test := range tests {
		if !sAll.isCommandAllowed(test.command) && test.command != "" {
			t.Errorf("With allowAllCommands=true, isCommandAllowed(%q) should be true", test.command)
		}
	}
}

func TestAddToHistory(t *testing.T) {
	s := &ShellServer{
		allowedCommands:  []string{"ls", "echo"},
		allowAllCommands: false,
		commandHistory:   []CommandExecution{},
	}

	// Add a few commands
	for i := 0; i < 5; i++ {
		execution := CommandExecution{
			Command:   fmt.Sprintf("command%d", i),
			Shell:     "bash",
			Output:    fmt.Sprintf("output%d", i),
			ExitCode:  0,
			StartTime: time.Now(),
			EndTime:   time.Now(),
		}
		s.addToHistory(execution)
	}

	// Check the length
	if len(s.commandHistory) != 5 {
		t.Errorf("History length = %d, want 5", len(s.commandHistory))
	}

	// Check the order (most recent first)
	if s.commandHistory[0].Command != "command4" {
		t.Errorf("First history entry = %s, want command4", s.commandHistory[0].Command)
	}

	// Add more commands to test truncation
	for i := 5; i < MAX_HISTORY_SIZE+10; i++ {
		execution := CommandExecution{
			Command:   fmt.Sprintf("command%d", i),
			Shell:     "bash",
			Output:    fmt.Sprintf("output%d", i),
			ExitCode:  0,
			StartTime: time.Now(),
			EndTime:   time.Now(),
		}
		s.addToHistory(execution)
	}

	// Check that history is truncated
	if len(s.commandHistory) > MAX_HISTORY_SIZE {
		t.Errorf("History length = %d, want at most %d", len(s.commandHistory), MAX_HISTORY_SIZE)
	}
}

func TestGetHistory(t *testing.T) {
	s := &ShellServer{
		allowedCommands:  []string{"ls", "echo"},
		allowAllCommands: false,
		commandHistory:   []CommandExecution{},
	}

	// Add some commands
	for i := 0; i < 10; i++ {
		execution := CommandExecution{
			Command:   fmt.Sprintf("command%d", i),
			Shell:     "bash",
			Output:    fmt.Sprintf("output%d", i),
			ExitCode:  0,
			StartTime: time.Now(),
			EndTime:   time.Now(),
		}
		s.addToHistory(execution)
	}

	// Test getting all history
	history := s.getHistory(0)
	if len(history) != 10 {
		t.Errorf("getHistory(0) returned %d items, want 10", len(history))
	}

	// Test getting limited history
	history = s.getHistory(5)
	if len(history) != 5 {
		t.Errorf("getHistory(5) returned %d items, want 5", len(history))
	}

	// Test getting more than available
	history = s.getHistory(20)
	if len(history) != 10 {
		t.Errorf("getHistory(20) returned %d items, want 10", len(history))
	}
}

func TestNewShellServer(t *testing.T) {
	// Test with specific allowed commands
	server, err := NewShellServer("ls,cat,echo")
	if err != nil {
		t.Fatalf("NewShellServer failed: %v", err)
	}

	if len(server.allowedCommands) != 3 {
		t.Errorf("server.allowedCommands has %d items, want 3", len(server.allowedCommands))
	}

	if server.allowAllCommands {
		t.Errorf("server.allowAllCommands = true, want false")
	}

	// Test with all commands allowed
	serverAll, err := NewShellServer("*")
	if err != nil {
		t.Fatalf("NewShellServer failed: %v", err)
	}

	if !serverAll.allowAllCommands {
		t.Errorf("serverAll.allowAllCommands = false, want true")
	}

	// Test with empty commands
	serverEmpty, err := NewShellServer("")
	if err != nil {
		t.Fatalf("NewShellServer failed: %v", err)
	}

	if len(serverEmpty.allowedCommands) != 0 {
		t.Errorf("serverEmpty.allowedCommands has %d items, want 0", len(serverEmpty.allowedCommands))
	}

	if serverEmpty.allowAllCommands {
		t.Errorf("serverEmpty.allowAllCommands = true, want false")
	}
}

// Missing imports
import (
	"fmt"
)
