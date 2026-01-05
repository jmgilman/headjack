// Package spinner provides a terminal spinner with ticker-style status display.
// It shows a spinning indicator alongside the latest log line from a subprocess,
// updating in place without polluting the terminal buffer.
package spinner

import (
	"bufio"
	"io"
	"os"
	"strings"
	"sync"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"golang.org/x/term"
)

// Spinner displays a spinner with ticker-style status updates.
// Output from a subprocess can be piped through Writer(), and the latest
// line will be displayed next to the spinner.
type Spinner struct {
	program *tea.Program
	reader  *io.PipeReader
	writer  *io.PipeWriter
	lineCh  chan string
	done    chan struct{}
	wg      sync.WaitGroup
	output  io.Writer
}

// New creates a new Spinner that writes to the given output (typically os.Stderr).
// If output is nil, os.Stderr is used.
func New(output io.Writer) *Spinner {
	if output == nil {
		output = os.Stderr
	}

	reader, writer := io.Pipe()
	return &Spinner{
		reader: reader,
		writer: writer,
		lineCh: make(chan string, 100), // Buffer to avoid blocking the pipe reader
		done:   make(chan struct{}),
		output: output,
	}
}

// Writer returns the io.Writer that should be passed to subprocesses.
// Lines written here will appear in the spinner's status display.
func (s *Spinner) Writer() io.Writer {
	return s.writer
}

// Start begins the spinner display. This blocks until Stop() is called.
// Call this in a goroutine if you need to do work while the spinner runs.
func (s *Spinner) Start() error {
	// Start the line reader goroutine
	s.wg.Add(1)
	go s.readLines()

	// Get terminal width for truncation
	width := 80 // default
	if fd := int(os.Stderr.Fd()); term.IsTerminal(fd) {
		if w, _, err := term.GetSize(fd); err == nil && w > 0 {
			width = w
		}
	}

	// Create the bubbletea model
	m := newModel(s.lineCh, width)

	// Create and run the program
	s.program = tea.NewProgram(m,
		tea.WithOutput(s.output),
		tea.WithoutSignalHandler(), // Let parent handle signals
	)

	_, err := s.program.Run()

	// Wait for line reader to finish
	s.wg.Wait()

	return err
}

// Stop stops the spinner and cleans up resources.
// The spinner line is cleared from the terminal.
func (s *Spinner) Stop() {
	// Close the writer to signal EOF to the line reader
	_ = s.writer.Close()

	// Signal done and close line channel
	close(s.done)

	// Tell the program to quit
	if s.program != nil {
		s.program.Quit()
	}
}

// readLines reads lines from the pipe and sends them to the model.
func (s *Spinner) readLines() {
	defer s.wg.Done()
	defer s.reader.Close()

	scanner := bufio.NewScanner(s.reader)
	for scanner.Scan() {
		line := scanner.Text()
		// Skip empty lines
		if strings.TrimSpace(line) == "" {
			continue
		}
		select {
		case s.lineCh <- line:
		case <-s.done:
			return
		}
	}
}

// model is the bubbletea model for the spinner.
type model struct {
	spinner    spinner.Model
	statusLine string
	width      int
	lineCh     <-chan string
	quitting   bool
}

// lineMsg is sent when a new line is received from the pipe.
type lineMsg string

// newModel creates a new spinner model.
func newModel(lineCh <-chan string, width int) model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	return model{
		spinner:    s,
		statusLine: "",
		width:      width,
		lineCh:     lineCh,
	}
}

// Init implements tea.Model.
//
//nolint:gocritic // hugeParam: tea.Model interface requires value receiver
func (m model) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		waitForLine(m.lineCh),
	)
}

// Update implements tea.Model.
//
//nolint:gocritic // hugeParam: tea.Model interface requires value receiver
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Allow ctrl+c to quit
		if msg.String() == "ctrl+c" {
			m.quitting = true
			return m, tea.Quit
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width

	case lineMsg:
		m.statusLine = string(msg)
		return m, waitForLine(m.lineCh)

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case tea.QuitMsg:
		m.quitting = true
		return m, nil
	}

	return m, nil
}

// View implements tea.Model.
//
//nolint:gocritic // hugeParam: tea.Model interface requires value receiver
func (m model) View() string {
	if m.quitting {
		return "" // Clear the line on exit
	}

	// Calculate available width for status line
	// Spinner is typically 2 chars + 1 space
	spinnerWidth := 3
	maxLineWidth := m.width - spinnerWidth
	if maxLineWidth < 10 {
		maxLineWidth = 10
	}

	line := truncate(m.statusLine, maxLineWidth)
	return m.spinner.View() + " " + line
}

// waitForLine returns a command that waits for the next line from the channel.
func waitForLine(lineCh <-chan string) tea.Cmd {
	return func() tea.Msg {
		line, ok := <-lineCh
		if !ok {
			return tea.Quit()
		}
		return lineMsg(line)
	}
}

// truncate shortens a string to fit within maxWidth.
// If truncated, it adds "..." at the end.
func truncate(s string, maxWidth int) string {
	if maxWidth <= 3 {
		return ""
	}
	if len(s) <= maxWidth {
		return s
	}
	return s[:maxWidth-3] + "..."
}
