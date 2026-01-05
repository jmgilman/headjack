// Package prompt provides user interaction primitives using charmbracelet/huh.
package prompt

import (
	"errors"
	"fmt"
	"strings"

	"github.com/charmbracelet/huh"
)

// ErrCanceled is returned when the user cancels a prompt.
var ErrCanceled = errors.New("canceled by user")

// Prompter abstracts user interaction for testability.
//
//go:generate go run github.com/matryer/moq@latest -pkg mocks -out mocks/prompter.go . Prompter
type Prompter interface {
	// Print outputs text to the user.
	Print(message string)

	// Confirm prompts for yes/no confirmation.
	Confirm(title, description string) (bool, error)

	// Secret prompts for secret input (no echo).
	Secret(prompt string) (string, error)

	// Choice prompts user to select from options, returns 0-based index.
	Choice(prompt string, options []string) (int, error)
}

// HuhPrompter implements Prompter using charmbracelet/huh for interactive forms.
type HuhPrompter struct{}

// New creates a new HuhPrompter for interactive terminal prompts.
func New() *HuhPrompter {
	return &HuhPrompter{}
}

// Print outputs text to the user.
func (p *HuhPrompter) Print(message string) {
	fmt.Println(message)
}

// Confirm prompts for yes/no confirmation.
func (p *HuhPrompter) Confirm(title, description string) (bool, error) {
	var confirmed bool

	err := huh.NewConfirm().
		Title(title).
		Description(description).
		Affirmative("Yes").
		Negative("No").
		Value(&confirmed).
		Run()

	if err != nil {
		if errors.Is(err, huh.ErrUserAborted) {
			return false, ErrCanceled
		}
		return false, fmt.Errorf("confirm prompt: %w", err)
	}

	return confirmed, nil
}

// Secret prompts for secret input with masked display.
func (p *HuhPrompter) Secret(prompt string) (string, error) {
	var value string

	err := huh.NewInput().
		Title(prompt).
		EchoMode(huh.EchoModePassword).
		Value(&value).
		Run()

	if err != nil {
		if errors.Is(err, huh.ErrUserAborted) {
			return "", ErrCanceled
		}
		return "", fmt.Errorf("secret prompt: %w", err)
	}

	return strings.TrimSpace(value), nil
}

// Choice prompts user to select from options and returns the 0-based index.
func (p *HuhPrompter) Choice(prompt string, options []string) (int, error) {
	if len(options) == 0 {
		return 0, errors.New("no options provided")
	}

	// Build huh options with display labels and index values
	huhOptions := make([]huh.Option[int], len(options))
	for i, opt := range options {
		huhOptions[i] = huh.NewOption(opt, i)
	}

	var selected int

	err := huh.NewSelect[int]().
		Title(prompt).
		Options(huhOptions...).
		Value(&selected).
		Run()

	if err != nil {
		if errors.Is(err, huh.ErrUserAborted) {
			return 0, ErrCanceled
		}
		return 0, fmt.Errorf("choice prompt: %w", err)
	}

	return selected, nil
}
