package main

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Styles
var (
	titleStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205"))
	helpStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	headerStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("252"))
	greenStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("46"))
	yellowStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("226"))
	redStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
)

const maxBarWidth = 40
const weeklyCapacity = 40.0 // hours per week

type viewMode int

const (
	tableView viewMode = iota
	chartView
)

type inputMode int

const (
	normalMode inputMode = iota
	addingMember
)

// TeamMember represents a team member with workload data.
type TeamMember struct {
	Name              string  `json:"name"`
	ActiveTasks       int     `json:"active_tasks"`
	CompletedThisWeek int     `json:"completed_this_week"`
	HoursEstimate     float64 `json:"hours_estimate"`
	LoadScore         int     `json:"load_score"`
}

// ComputeLoadScore calculates load score (0-100) from hours estimate vs weekly capacity.
func ComputeLoadScore(hoursEstimate float64) int {
	score := int(math.Round(hoursEstimate / weeklyCapacity * 100))
	if score > 100 {
		score = 100
	}
	if score < 0 {
		score = 0
	}
	return score
}

// LoadLevel returns "green", "yellow", or "red" based on the load score.
func LoadLevel(score int) string {
	switch {
	case score > 80:
		return "red"
	case score >= 60:
		return "yellow"
	default:
		return "green"
	}
}

// loadStyle returns the lipgloss style for a given load score.
func loadStyle(score int) lipgloss.Style {
	switch LoadLevel(score) {
	case "red":
		return redStyle
	case "yellow":
		return yellowStyle
	default:
		return greenStyle
	}
}

func mockData() []TeamMember {
	members := []TeamMember{
		{"Alice Chen", 5, 8, 35.0, 0},
		{"Bob Martinez", 8, 3, 42.0, 0},
		{"Carol Singh", 3, 10, 22.0, 0},
		{"David Kim", 7, 5, 38.0, 0},
		{"Eva Novak", 10, 2, 48.0, 0},
		{"Frank Osei", 4, 7, 28.0, 0},
		{"Grace Liu", 6, 6, 32.0, 0},
		{"Hiro Tanaka", 9, 4, 44.0, 0},
	}
	for i := range members {
		members[i].LoadScore = ComputeLoadScore(members[i].HoursEstimate)
	}
	return members
}

type model struct {
	members  []TeamMember
	cursor   int
	view     viewMode
	input    inputMode
	inputBuf string
}

func initialModel() model {
	return model{
		members: mockData(),
	}
}

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.input == addingMember {
			return m.handleAddInput(msg)
		}
		return m.handleNormalInput(msg)
	}
	return m, nil
}

func (m model) handleAddInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		name := strings.TrimSpace(m.inputBuf)
		if name != "" {
			member := TeamMember{Name: name}
			m.members = append(m.members, member)
			m.cursor = len(m.members) - 1
		}
		m.input = normalMode
		m.inputBuf = ""
	case "esc":
		m.input = normalMode
		m.inputBuf = ""
	case "backspace":
		if len(m.inputBuf) > 0 {
			m.inputBuf = m.inputBuf[:len(m.inputBuf)-1]
		}
	default:
		if len(msg.String()) == 1 {
			m.inputBuf += msg.String()
		}
	}
	return m, nil
}

func (m model) handleNormalInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "j", "down":
		if m.cursor < len(m.members)-1 {
			m.cursor++
		}
	case "k", "up":
		if m.cursor > 0 {
			m.cursor--
		}
	case "tab":
		if m.view == tableView {
			m.view = chartView
		} else {
			m.view = tableView
		}
	case "a":
		m.input = addingMember
		m.inputBuf = ""
	case "d":
		if len(m.members) > 0 {
			m.members = append(m.members[:m.cursor], m.members[m.cursor+1:]...)
			if m.cursor >= len(m.members) && m.cursor > 0 {
				m.cursor--
			}
		}
	case "+", "=":
		if len(m.members) > 0 {
			m.members[m.cursor].ActiveTasks++
			m.members[m.cursor].HoursEstimate += 4.0
			m.members[m.cursor].LoadScore = ComputeLoadScore(m.members[m.cursor].HoursEstimate)
		}
	case "-":
		if len(m.members) > 0 && m.members[m.cursor].ActiveTasks > 0 {
			m.members[m.cursor].ActiveTasks--
			m.members[m.cursor].HoursEstimate = math.Max(0, m.members[m.cursor].HoursEstimate-4.0)
			m.members[m.cursor].LoadScore = ComputeLoadScore(m.members[m.cursor].HoursEstimate)
		}
	}
	return m, nil
}

func (m model) View() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("WorkloadBalanceAnalyzer"))
	b.WriteString("\n\n")

	if m.input == addingMember {
		b.WriteString("Enter member name: ")
		b.WriteString(m.inputBuf)
		b.WriteString("█\n")
		b.WriteString(helpStyle.Render("enter: confirm  esc: cancel"))
		return b.String()
	}

	if len(m.members) == 0 {
		b.WriteString("No team members. Press 'a' to add one.\n")
		b.WriteString("\n")
		b.WriteString(helpStyle.Render("a: add  q: quit"))
		return b.String()
	}

	switch m.view {
	case tableView:
		b.WriteString(renderTable(m.members, m.cursor))
	case chartView:
		b.WriteString(renderChart(m.members, m.cursor))
	}

	b.WriteString("\n")
	viewLabel := "table"
	if m.view == chartView {
		viewLabel = "chart"
	}
	b.WriteString(helpStyle.Render(fmt.Sprintf(
		"j/k: navigate  tab: toggle view (%s)  +/-: adjust tasks  a: add  d: delete  q: quit",
		viewLabel,
	)))

	return b.String()
}

func renderTable(members []TeamMember, cursor int) string {
	var b strings.Builder
	header := fmt.Sprintf("  %-18s %6s %9s %7s %6s", "Name", "Active", "Done/Wk", "Hours", "Load")
	b.WriteString(headerStyle.Render(header))
	b.WriteString("\n")
	b.WriteString(strings.Repeat("─", 54))
	b.WriteString("\n")

	for i, member := range members {
		cur := "  "
		if i == cursor {
			cur = "> "
		}
		loadStr := fmt.Sprintf("%3d%%", member.LoadScore)
		colored := loadStyle(member.LoadScore).Render(loadStr)

		b.WriteString(fmt.Sprintf("%s%-18s %6d %9d %6.0fh %s\n",
			cur,
			truncate(member.Name, 18),
			member.ActiveTasks,
			member.CompletedThisWeek,
			member.HoursEstimate,
			colored,
		))
	}
	return b.String()
}

func renderChart(members []TeamMember, cursor int) string {
	var b strings.Builder
	b.WriteString(headerStyle.Render("Load Score Distribution"))
	b.WriteString("\n\n")

	for i, member := range members {
		cur := "  "
		if i == cursor {
			cur = "> "
		}
		barWidth := member.LoadScore * maxBarWidth / 100
		if barWidth < 0 {
			barWidth = 0
		}
		bar := strings.Repeat("█", barWidth)
		coloredBar := loadStyle(member.LoadScore).Render(bar)
		name := fmt.Sprintf("%-14s", truncate(member.Name, 14))
		b.WriteString(fmt.Sprintf("%s%s %s %d%%\n", cur, name, coloredBar, member.LoadScore))
	}
	return b.String()
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	if max <= 1 {
		return "…"
	}
	return s[:max-1] + "…"
}

func printJSON(members []TeamMember) {
	data, err := json.MarshalIndent(members, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(string(data))
}

func main() {
	for _, arg := range os.Args[1:] {
		if arg == "--json" {
			printJSON(mockData())
			return
		}
	}

	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
