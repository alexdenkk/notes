package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Note status constants
const (
	statusTodo     = "todo"
	statusProgress = "progress"
	statusClosed   = "closed"
)

// Note represents a single note
type Note struct {
	Content string `json:"content"`
	Status  string `json:"status"`
}

// Styles
var (
	styleTodo     = lipgloss.NewStyle().Foreground(lipgloss.Color("39"))
	styleProgress = lipgloss.NewStyle().Foreground(lipgloss.Color("208"))
	styleClosed   = lipgloss.NewStyle().Foreground(lipgloss.Color("242")).Strikethrough(true)
	cursorStyle   = lipgloss.NewStyle().Background(lipgloss.Color("236")).SetString(">")
)

type model struct {
	notes         []Note
	cursor        int
	editing       bool
	editInput     textinput.Model
	editStatus    string
	copyMessage   string
	width, height int
}

func initialModel() model {
	ti := textinput.New()
	ti.Placeholder = "Content"
	ti.Focus()
	ti.CharLimit = 200
	ti.Width = 80

	m := model{
		notes:      loadNotes(),
		editing:    false,
		editInput:  ti,
		editStatus: statusTodo,
	}
	if len(m.notes) == 0 {
		m.notes = []Note{
			{Content: "Write!", Status: statusTodo},
		}
		saveNotes(m.notes)
	}
	return m
}

func (m model) Init() tea.Cmd {
	return tea.Batch(textinput.Blink, tea.ClearScreen)
}

// A command to clear the copy message after a delay
func clearCopyMessageAfterDelay() tea.Cmd {
	return tea.Tick(1*time.Second, func(_ time.Time) tea.Msg {
		return clearCopyMsg{}
	})
}

type clearCopyMsg struct{}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	if !m.editing {
		switch msg := msg.(type) {
		case tea.WindowSizeMsg:
			m.width = msg.Width
			m.height = msg.Height
			return m, nil
		case clearCopyMsg:
			m.copyMessage = ""
			return m, nil
		case tea.KeyMsg:
			switch msg.String() {
			case "q", "ctrl+c":
				return m, tea.Quit
			case "up", "k":
				if m.cursor > 0 {
					m.cursor--
				}
			case "down", "j":
				if m.cursor < len(m.notes)-1 {
					m.cursor++
				}
			case "enter":
				m.editing = true
				m.editInput.SetValue(m.notes[m.cursor].Content)
				m.editStatus = m.notes[m.cursor].Status
				m.editInput.Focus()
				return m, textinput.Blink
			case "n":
				m.notes = append(m.notes, Note{Content: "", Status: statusTodo})
				m.cursor = len(m.notes) - 1
				saveNotes(m.notes)
				m.editing = true
				m.editInput.SetValue("")
				m.editStatus = statusTodo
				m.editInput.Focus()
				return m, textinput.Blink
			case "d":
				if len(m.notes) > 0 {
					m.notes = append(m.notes[:m.cursor], m.notes[m.cursor+1:]...)
					if m.cursor >= len(m.notes) && m.cursor > 0 {
						m.cursor--
					}
					saveNotes(m.notes)
				}
			case "c":
				// Copy current note content to clipboard
				if len(m.notes) > 0 {
					content := m.notes[m.cursor].Content
					err := clipboard.WriteAll(content)
					if err != nil {
						m.copyMessage = "Failed to copy!"
					} else {
						m.copyMessage = "Copied!"
					}
					cmds = append(cmds, clearCopyMessageAfterDelay())
				}
			case "1":
				m.notes[m.cursor].Status = statusTodo
				saveNotes(m.notes)
			case "2":
				m.notes[m.cursor].Status = statusProgress
				saveNotes(m.notes)
			case "3":
				m.notes[m.cursor].Status = statusClosed
				saveNotes(m.notes)
			}
		}
	} else {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "enter":
				m.notes[m.cursor].Content = m.editInput.Value()
				m.notes[m.cursor].Status = m.editStatus
				saveNotes(m.notes)
				m.editing = false
				m.editInput.Blur()
				return m, nil
			case "esc":
				m.editing = false
				m.editInput.Blur()
				return m, nil
			case "ctrl+c":
				return m, tea.Quit
			case "1", "2", "3":
				switch msg.String() {
				case "1":
					m.editStatus = statusTodo
				case "2":
					m.editStatus = statusProgress
				case "3":
					m.editStatus = statusClosed
				}
			default:
				var cmd tea.Cmd
				m.editInput, cmd = m.editInput.Update(msg)
				return m, cmd
			}
		}
		if _, ok := msg.(tea.WindowSizeMsg); ok {
			return m, nil
		}
	}

	return m, tea.Batch(cmds...)
}

func (m model) View() string {
	if m.editing {
		var statusLine string
		switch m.editStatus {
		case statusTodo:
			statusLine = styleTodo.Render("[ ] Todo")
		case statusProgress:
			statusLine = styleProgress.Render("[~] In progress")
		case statusClosed:
			statusLine = styleClosed.Render("[x] Closed")
		}
		help := "\n\nPress 1/2/3 to change status, Enter to save, Esc to cancel\n"
		return lipgloss.JoinVertical(lipgloss.Top,
			"Editing note:\n",
			m.editInput.View(),
			"\nStatus: "+statusLine,
			help,
		)
	}

	if len(m.notes) == 0 {
		return "No notes. Press 'n' to create one.\nPress 'q' to quit.\n"
	}

	var b strings.Builder
	b.WriteString("Notes (n to add, d to delete, c to copy, q to quit)\n\n")

	for i, note := range m.notes {
		cursor := "  "
		if i == m.cursor {
			cursor = cursorStyle.String()
		}

		statusSymbol := ""
		styledContent := ""
		switch note.Status {
		case statusTodo:
			statusSymbol = styleTodo.Render("[ ]")
			styledContent = styleTodo.Render(note.Content)
		case statusProgress:
			statusSymbol = styleProgress.Render("[~]")
			styledContent = styleProgress.Render(note.Content)
		case statusClosed:
			statusSymbol = styleClosed.Render("[x]")
			styledContent = styleClosed.Render(note.Content)
		}

		line := fmt.Sprintf("%s %s %s\n", cursor, statusSymbol, styledContent)
		b.WriteString(line)
	}

	b.WriteString("\nStatus change: 1=TODO  2=In progress  3=Closed\n")
	if m.copyMessage != "" {
		b.WriteString("\n" + lipgloss.NewStyle().Foreground(lipgloss.Color("46")).Render(m.copyMessage))
	}
	return b.String()
}

// Persistence
const notesFile = "note.json"

func loadNotes() []Note {
	data, err := os.ReadFile(notesFile)
	if err != nil {
		return []Note{}
	}
	var notes []Note
	if err := json.Unmarshal(data, &notes); err != nil {
		return []Note{}
	}
	return notes
}

func saveNotes(notes []Note) {
	data, err := json.MarshalIndent(notes, "", "  ")
	if err != nil {
		return
	}
	_ = os.WriteFile(notesFile, data, 0644)
}

func main() {
	p := tea.NewProgram(initialModel())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}
