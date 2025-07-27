package login

import (
	"fmt"
	"opsicle/internal/cli"
	"opsicle/pkg/controller"
	"os"
	"path/filepath"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/charmbracelet/bubbles/cursor"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var flags cli.Flags = cli.Flags{
	{
		Name:         "controller-url",
		DefaultValue: "http://localhost:54321",
		Usage:        "defines the url where the controller service is accessible at",
		Type:         cli.FlagTypeString,
	},
}

func init() {
	flags.AddToCommand(Command)
}

var Command = &cobra.Command{
	Use:   "login",
	Short: "Logs into Opsicle from your terminal",
	PreRun: func(cmd *cobra.Command, args []string) {
		flags.BindViper(cmd)
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		userHomeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to determine user's home directory: %s", err)
		}
		sessionPath := filepath.Join(userHomeDir, "/.opsicle/session")
		fileInfo, err := os.Lstat(sessionPath)
		if err != nil {
			if os.IsNotExist(err) {
				if err := os.MkdirAll(sessionPath, os.ModePerm); err != nil {
					return fmt.Errorf("failed to provision configuration directory at path[%s]: %s", sessionPath, err)
				}
				fileInfo, _ = os.Lstat(sessionPath)
			} else {
				return fmt.Errorf("path[%s] for session information does not exist: %s", sessionPath, err)
			}
		}
		if !fileInfo.IsDir() {
			return fmt.Errorf("path[%s] exists but is not a directory, it should be", sessionPath)
		}
		sessionFilePath := filepath.Join(sessionPath, "current")
		fileInfo, err = os.Lstat(sessionFilePath)
		if err != nil {
			if !os.IsNotExist(err) {
				return fmt.Errorf("failed to check current session file at path[%s]: %s", sessionFilePath, err)
			}
		} else if fileInfo.IsDir() {
			return fmt.Errorf("path[%s] exists but is a directory, it should be a file", sessionFilePath)
		} else if fileInfo.Size() > 0 {
			return fmt.Errorf("an existing session already exists, use logout to stop the current session first")
		}

		model := getModel()
		program := tea.NewProgram(model)
		if _, err := program.Run(); err != nil {
			return fmt.Errorf("failed to get user input: %s", err)
		}
		controllerUrl := viper.GetString("controller-url")
		client, err := controller.NewClient(controller.NewClientOpts{
			ControllerUrl: controllerUrl,
			Id:            "opsicle/login",
		})
		if err != nil {
			return fmt.Errorf("failed to create controller client: %s", err)
		}
		sessionId, sessionToken, err := client.CreateSessionV1(controller.CreateSessionV1Opts{
			Email:    model.GetEmail(),
			Password: model.GetPassword(),
		})
		if err != nil {
			return fmt.Errorf("failed to create session: %s", err)
		}

		if err := os.WriteFile(sessionFilePath, []byte(sessionToken), 0600); err != nil {
			return fmt.Errorf("failed to write session token to path[%s]: %s", sessionFilePath, err)
		}

		logrus.Infof("opened session[%s]: welcome to opsicle!", sessionId)
		return nil
	},
}

var (
	focusedStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	blurredStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	cursorStyle         = focusedStyle
	noStyle             = lipgloss.NewStyle()
	helpStyle           = blurredStyle
	cursorModeHelpStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))

	focusedButton = focusedStyle.Render("[ Submit ]")
	blurredButton = fmt.Sprintf("[ %s ]", blurredStyle.Render("Submit"))
)

type errMsg error

type model struct {
	focusIndex int
	inputs     []textinput.Model
	cursorMode cursor.Mode
}

func getModel() model {
	m := model{
		inputs: make([]textinput.Model, 2),
	}

	var t textinput.Model
	for i := range m.inputs {
		t = textinput.New()
		t.Cursor.Style = cursorStyle
		t.CharLimit = 32

		switch i {
		case 0:
			t.Focus()
			t.Width = 64
			t.CharLimit = 256
			t.Placeholder = "Email"
			t.PromptStyle = focusedStyle
			t.TextStyle = focusedStyle
		case 1:
			t.Width = 64
			t.CharLimit = 128
			t.Placeholder = "Password"
			t.EchoMode = textinput.EchoPassword
			t.EchoCharacter = '*'
		}

		m.inputs[i] = t
	}

	return m
}

func (m model) GetEmail() string {
	return m.inputs[0].Value()
}

func (m model) GetPassword() string {
	return m.inputs[1].Value()
}

func (m model) Init() tea.Cmd {
	return textinput.Blink
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			return m, tea.Quit

		// Set focus to next input
		case "tab", "shift+tab", "enter", "up", "down":
			s := msg.String()

			// Did the user press enter while the submit button was focused?
			// If so, exit.
			if s == "enter" && m.focusIndex == len(m.inputs) {
				return m, tea.Quit
			}

			// Cycle indexes
			if s == "up" || s == "shift+tab" {
				m.focusIndex--
			} else {
				m.focusIndex++
			}

			if m.focusIndex > len(m.inputs) {
				m.focusIndex = 0
			} else if m.focusIndex < 0 {
				m.focusIndex = len(m.inputs)
			}

			cmds := make([]tea.Cmd, len(m.inputs))
			for i := 0; i <= len(m.inputs)-1; i++ {
				if i == m.focusIndex {
					// Set focused state
					cmds[i] = m.inputs[i].Focus()
					m.inputs[i].PromptStyle = focusedStyle
					m.inputs[i].TextStyle = focusedStyle
					continue
				}
				// Remove focused state
				m.inputs[i].Blur()
				m.inputs[i].PromptStyle = noStyle
				m.inputs[i].TextStyle = noStyle
			}

			return m, tea.Batch(cmds...)
		}
	}

	// Handle character input and blinking
	cmd := m.updateInputs(msg)

	return m, cmd
}

func (m *model) updateInputs(msg tea.Msg) tea.Cmd {
	cmds := make([]tea.Cmd, len(m.inputs))

	// Only text inputs with Focus() set will respond, so it's safe to simply
	// update all of them here without any further logic.
	for i := range m.inputs {
		m.inputs[i], cmds[i] = m.inputs[i].Update(msg)
	}

	return tea.Batch(cmds...)
}

func (m model) View() string {
	var b strings.Builder

	for i := range m.inputs {
		b.WriteString(m.inputs[i].View())
		if i < len(m.inputs)-1 {
			b.WriteRune('\n')
		}
	}

	button := &blurredButton
	if m.focusIndex == len(m.inputs) {
		button = &focusedButton
	}
	fmt.Fprintf(&b, "\n\n%s\n\n", *button)

	b.WriteString(helpStyle.Render("cursor mode is "))
	b.WriteString(cursorModeHelpStyle.Render(m.cursorMode.String()))
	b.WriteString(helpStyle.Render(" (ctrl+r to change style)"))

	return b.String()
}
