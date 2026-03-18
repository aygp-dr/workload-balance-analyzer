package main

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// Helper to create a key message for a rune character.
func keyMsg(char rune) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{char}}
}

// Helper to create a key message for a special key type.
func specialKeyMsg(kt tea.KeyType) tea.KeyMsg {
	return tea.KeyMsg{Type: kt}
}

func TestComputeLoadScore(t *testing.T) {
	tests := []struct {
		hours float64
		want  int
	}{
		{0, 0},
		{20, 50},
		{24, 60},
		{32, 80},
		{40, 100},
		{50, 100}, // capped at 100
		{-5, 0},   // capped at 0
	}
	for _, tt := range tests {
		got := ComputeLoadScore(tt.hours)
		if got != tt.want {
			t.Errorf("ComputeLoadScore(%v) = %d, want %d", tt.hours, got, tt.want)
		}
	}
}

func TestLoadLevel(t *testing.T) {
	tests := []struct {
		score int
		want  string
	}{
		{0, "green"},
		{30, "green"},
		{59, "green"},
		{60, "yellow"},
		{70, "yellow"},
		{80, "yellow"},
		{81, "red"},
		{100, "red"},
	}
	for _, tt := range tests {
		got := LoadLevel(tt.score)
		if got != tt.want {
			t.Errorf("LoadLevel(%d) = %q, want %q", tt.score, got, tt.want)
		}
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		input string
		max   int
		want  string
	}{
		{"Alice", 10, "Alice"},
		{"Alice", 5, "Alice"},
		{"Alice Chen", 5, "Alic…"},
		{"A", 1, "A"},
		{"AB", 1, "…"},
		{"", 5, ""},
	}
	for _, tt := range tests {
		got := truncate(tt.input, tt.max)
		if got != tt.want {
			t.Errorf("truncate(%q, %d) = %q, want %q", tt.input, tt.max, got, tt.want)
		}
	}
}

func TestMockData(t *testing.T) {
	members := mockData()
	if len(members) != 8 {
		t.Fatalf("mockData() returned %d members, want 8", len(members))
	}
	for _, m := range members {
		if m.Name == "" {
			t.Error("member has empty name")
		}
		if m.LoadScore < 0 || m.LoadScore > 100 {
			t.Errorf("member %q has invalid load score %d", m.Name, m.LoadScore)
		}
		expected := ComputeLoadScore(m.HoursEstimate)
		if m.LoadScore != expected {
			t.Errorf("member %q: LoadScore=%d, expected ComputeLoadScore(%v)=%d",
				m.Name, m.LoadScore, m.HoursEstimate, expected)
		}
	}
}

func TestInitialModel(t *testing.T) {
	m := initialModel()
	if m.cursor != 0 {
		t.Errorf("initial cursor = %d, want 0", m.cursor)
	}
	if m.view != tableView {
		t.Errorf("initial view = %v, want tableView", m.view)
	}
	if m.input != normalMode {
		t.Errorf("initial input = %v, want normalMode", m.input)
	}
	if len(m.members) != 8 {
		t.Errorf("initial members count = %d, want 8", len(m.members))
	}
}

func TestNavigateDown(t *testing.T) {
	m := initialModel()
	updated, _ := m.Update(keyMsg('j'))
	um := updated.(model)
	if um.cursor != 1 {
		t.Errorf("after j: cursor = %d, want 1", um.cursor)
	}
}

func TestNavigateUp(t *testing.T) {
	m := initialModel()
	m.cursor = 3
	updated, _ := m.Update(keyMsg('k'))
	um := updated.(model)
	if um.cursor != 2 {
		t.Errorf("after k from 3: cursor = %d, want 2", um.cursor)
	}
}

func TestNavigateUpAtTop(t *testing.T) {
	m := initialModel()
	m.cursor = 0
	updated, _ := m.Update(keyMsg('k'))
	um := updated.(model)
	if um.cursor != 0 {
		t.Errorf("after k from 0: cursor = %d, want 0", um.cursor)
	}
}

func TestNavigateDownAtBottom(t *testing.T) {
	m := initialModel()
	m.cursor = len(m.members) - 1
	updated, _ := m.Update(keyMsg('j'))
	um := updated.(model)
	if um.cursor != len(m.members)-1 {
		t.Errorf("after j at bottom: cursor = %d, want %d", um.cursor, len(m.members)-1)
	}
}

func TestToggleView(t *testing.T) {
	m := initialModel()
	if m.view != tableView {
		t.Fatal("expected tableView initially")
	}

	updated, _ := m.Update(specialKeyMsg(tea.KeyTab))
	um := updated.(model)
	if um.view != chartView {
		t.Errorf("after tab: view = %v, want chartView", um.view)
	}

	updated2, _ := um.Update(specialKeyMsg(tea.KeyTab))
	um2 := updated2.(model)
	if um2.view != tableView {
		t.Errorf("after second tab: view = %v, want tableView", um2.view)
	}
}

func TestAddMember(t *testing.T) {
	m := initialModel()
	original := len(m.members)

	// Press 'a' to enter add mode
	updated, _ := m.Update(keyMsg('a'))
	um := updated.(model)
	if um.input != addingMember {
		t.Fatal("expected addingMember mode after pressing 'a'")
	}

	// Type "Zoe"
	for _, ch := range "Zoe" {
		updated, _ = um.Update(keyMsg(ch))
		um = updated.(model)
	}
	if um.inputBuf != "Zoe" {
		t.Errorf("inputBuf = %q, want %q", um.inputBuf, "Zoe")
	}

	// Press enter to confirm
	updated, _ = um.Update(specialKeyMsg(tea.KeyEnter))
	um = updated.(model)
	if um.input != normalMode {
		t.Error("expected normalMode after enter")
	}
	if len(um.members) != original+1 {
		t.Errorf("members count = %d, want %d", len(um.members), original+1)
	}
	if um.members[len(um.members)-1].Name != "Zoe" {
		t.Errorf("last member name = %q, want %q", um.members[len(um.members)-1].Name, "Zoe")
	}
	if um.cursor != len(um.members)-1 {
		t.Errorf("cursor = %d, want %d (last member)", um.cursor, len(um.members)-1)
	}
}

func TestAddMemberCancel(t *testing.T) {
	m := initialModel()
	original := len(m.members)

	updated, _ := m.Update(keyMsg('a'))
	um := updated.(model)

	// Type something then cancel
	updated, _ = um.Update(keyMsg('X'))
	um = updated.(model)
	updated, _ = um.Update(specialKeyMsg(tea.KeyEscape))
	um = updated.(model)

	if um.input != normalMode {
		t.Error("expected normalMode after escape")
	}
	if len(um.members) != original {
		t.Errorf("members count = %d, want %d (unchanged)", len(um.members), original)
	}
}

func TestAddMemberBackspace(t *testing.T) {
	m := initialModel()

	updated, _ := m.Update(keyMsg('a'))
	um := updated.(model)

	// Type "AB" then backspace
	updated, _ = um.Update(keyMsg('A'))
	um = updated.(model)
	updated, _ = um.Update(keyMsg('B'))
	um = updated.(model)
	if um.inputBuf != "AB" {
		t.Fatalf("inputBuf = %q, want %q", um.inputBuf, "AB")
	}

	updated, _ = um.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	um = updated.(model)
	if um.inputBuf != "A" {
		t.Errorf("after backspace: inputBuf = %q, want %q", um.inputBuf, "A")
	}
}

func TestAddEmptyName(t *testing.T) {
	m := initialModel()
	original := len(m.members)

	updated, _ := m.Update(keyMsg('a'))
	um := updated.(model)
	// Press enter without typing anything
	updated, _ = um.Update(specialKeyMsg(tea.KeyEnter))
	um = updated.(model)

	if len(um.members) != original {
		t.Errorf("empty name should not add member, got %d members, want %d", len(um.members), original)
	}
}

func TestDeleteMember(t *testing.T) {
	m := initialModel()
	original := len(m.members)
	firstName := m.members[0].Name

	// Delete first member
	updated, _ := m.Update(keyMsg('d'))
	um := updated.(model)
	if len(um.members) != original-1 {
		t.Errorf("members count = %d, want %d", len(um.members), original-1)
	}
	if um.members[0].Name == firstName {
		t.Error("first member should have been deleted")
	}
}

func TestDeleteLastMember(t *testing.T) {
	m := initialModel()
	m.cursor = len(m.members) - 1

	updated, _ := m.Update(keyMsg('d'))
	um := updated.(model)
	if um.cursor >= len(um.members) {
		t.Errorf("cursor = %d out of range after deleting last member (len=%d)", um.cursor, len(um.members))
	}
}

func TestDeleteAllMembers(t *testing.T) {
	m := model{members: []TeamMember{{Name: "Only"}}, cursor: 0}

	updated, _ := m.Update(keyMsg('d'))
	um := updated.(model)
	if len(um.members) != 0 {
		t.Errorf("members count = %d, want 0", len(um.members))
	}
	if um.cursor != 0 {
		t.Errorf("cursor = %d, want 0", um.cursor)
	}
}

func TestIncreaseTasks(t *testing.T) {
	m := initialModel()
	origTasks := m.members[0].ActiveTasks
	origHours := m.members[0].HoursEstimate

	updated, _ := m.Update(keyMsg('+'))
	um := updated.(model)

	if um.members[0].ActiveTasks != origTasks+1 {
		t.Errorf("ActiveTasks = %d, want %d", um.members[0].ActiveTasks, origTasks+1)
	}
	if um.members[0].HoursEstimate != origHours+4.0 {
		t.Errorf("HoursEstimate = %v, want %v", um.members[0].HoursEstimate, origHours+4.0)
	}
	expected := ComputeLoadScore(origHours + 4.0)
	if um.members[0].LoadScore != expected {
		t.Errorf("LoadScore = %d, want %d", um.members[0].LoadScore, expected)
	}
}

func TestDecreaseTasks(t *testing.T) {
	m := initialModel()
	origTasks := m.members[0].ActiveTasks
	origHours := m.members[0].HoursEstimate

	updated, _ := m.Update(keyMsg('-'))
	um := updated.(model)

	if um.members[0].ActiveTasks != origTasks-1 {
		t.Errorf("ActiveTasks = %d, want %d", um.members[0].ActiveTasks, origTasks-1)
	}
	if um.members[0].HoursEstimate != origHours-4.0 {
		t.Errorf("HoursEstimate = %v, want %v", um.members[0].HoursEstimate, origHours-4.0)
	}
}

func TestDecreaseTasksAtZero(t *testing.T) {
	m := model{
		members: []TeamMember{{Name: "Zero", ActiveTasks: 0, HoursEstimate: 0}},
	}

	updated, _ := m.Update(keyMsg('-'))
	um := updated.(model)

	if um.members[0].ActiveTasks != 0 {
		t.Errorf("ActiveTasks should stay at 0, got %d", um.members[0].ActiveTasks)
	}
	if um.members[0].HoursEstimate != 0 {
		t.Errorf("HoursEstimate should stay at 0, got %v", um.members[0].HoursEstimate)
	}
}

func TestTableViewContainsNames(t *testing.T) {
	m := initialModel()
	view := m.View()

	for _, member := range m.members {
		name := truncate(member.Name, 18)
		if !strings.Contains(view, name) {
			t.Errorf("table view should contain %q", name)
		}
	}
}

func TestChartViewContainsNames(t *testing.T) {
	m := initialModel()
	m.view = chartView
	view := m.View()

	for _, member := range m.members {
		name := truncate(member.Name, 14)
		if !strings.Contains(view, name) {
			t.Errorf("chart view should contain %q", name)
		}
	}
	if !strings.Contains(view, "█") {
		t.Error("chart view should contain bar characters")
	}
}

func TestChartViewShowsPercentages(t *testing.T) {
	m := initialModel()
	m.view = chartView
	view := m.View()

	for _, member := range m.members {
		pct := fmt.Sprintf("%d%%", member.LoadScore)
		if !strings.Contains(view, pct) {
			t.Errorf("chart view should contain %q for %s", pct, member.Name)
		}
	}
}

func TestEmptyMembersView(t *testing.T) {
	m := model{}
	view := m.View()
	if !strings.Contains(view, "No team members") {
		t.Error("empty state should show 'No team members' message")
	}
}

func TestAddingMemberView(t *testing.T) {
	m := initialModel()
	m.input = addingMember
	m.inputBuf = "Test"
	view := m.View()
	if !strings.Contains(view, "Enter member name") {
		t.Error("add mode should show input prompt")
	}
	if !strings.Contains(view, "Test") {
		t.Error("add mode should show typed text")
	}
}

func TestRenderTable(t *testing.T) {
	members := []TeamMember{
		{"Test User", 3, 5, 24.0, 60},
	}
	output := renderTable(members, 0)
	if !strings.Contains(output, "Test User") {
		t.Error("renderTable should contain member name")
	}
	if !strings.Contains(output, "Name") {
		t.Error("renderTable should contain header")
	}
}

func TestRenderChart(t *testing.T) {
	members := []TeamMember{
		{"Test User", 3, 5, 24.0, 60},
	}
	output := renderChart(members, 0)
	if !strings.Contains(output, "Test User") {
		t.Error("renderChart should contain member name")
	}
	if !strings.Contains(output, "60%") {
		t.Error("renderChart should contain load percentage")
	}
}

func TestJSONOutput(t *testing.T) {
	members := mockData()
	data, err := json.MarshalIndent(members, "", "  ")
	if err != nil {
		t.Fatalf("JSON marshal failed: %v", err)
	}

	var parsed []TeamMember
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("JSON unmarshal failed: %v", err)
	}
	if len(parsed) != 8 {
		t.Errorf("JSON output has %d members, want 8", len(parsed))
	}
	for i, m := range parsed {
		if m.Name != members[i].Name {
			t.Errorf("member %d: name=%q, want %q", i, m.Name, members[i].Name)
		}
		if m.LoadScore != members[i].LoadScore {
			t.Errorf("member %d: load=%d, want %d", i, m.LoadScore, members[i].LoadScore)
		}
	}
}

func TestViewHelpText(t *testing.T) {
	m := initialModel()
	view := m.View()
	if !strings.Contains(view, "j/k: navigate") {
		t.Error("view should contain navigation help")
	}
	if !strings.Contains(view, "tab: toggle view") {
		t.Error("view should contain view toggle help")
	}
}

func TestCursorIndicator(t *testing.T) {
	m := initialModel()
	view := m.View()
	if !strings.Contains(view, "> ") {
		t.Error("view should contain cursor indicator '> '")
	}
}
