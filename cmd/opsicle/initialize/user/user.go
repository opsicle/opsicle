package user

import (
	"fmt"
	"opsicle/internal/cli"
	"opsicle/pkg/controller"
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
	Use:     "user",
	Aliases: []string{"u", "me"},
	Short:   "Initialises a user account for the current user",
	PreRun: func(cmd *cobra.Command, args []string) {
		flags.BindViper(cmd)
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		_, _, err := controller.GetSessionToken()
		if err == nil {
			return fmt.Errorf("looks like you're already logged in, run `opsicle logout` first before running this command")
		}

		model := getModel()
		program := tea.NewProgram(model)
		if _, err := program.Run(); err != nil {
			return fmt.Errorf("failed to get user input: %s", err)
		}

		controllerUrl := viper.GetString("controller-url")
		logrus.Debugf("creating new client...")
		client, err := controller.NewClient(controller.NewClientOpts{
			ControllerUrl: controllerUrl,
			Id:            "opsicle/initialize/user",
		})
		if err != nil {
			return fmt.Errorf("failed to create controller client: %s", err)
		}

		email := model.GetEmail()
		password := model.GetPassword()

		if email == "" || password == "" {
			return fmt.Errorf("failed to receive an email or password")
		}

		logrus.Debugf("sending request to server...")
		createUserV1Output, err := client.CreateUserV1(controller.CreateUserV1Input{
			Email:    email,
			Password: password,
		})
		if err != nil {
			return fmt.Errorf("failed to create user: %s", err)
		}

		logrus.Debugf("successfully created user[%s] with id[%s]", createUserV1Output.Email, createUserV1Output.Id)

		logrus.Infof("user[%s] created successfully! login using 'opsicle login'", createUserV1Output.Email)
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
			t.Placeholder = "Your email address"
			t.PromptStyle = focusedStyle
			t.TextStyle = focusedStyle
		case 1:
			t.Width = 64
			t.CharLimit = 128
			t.Placeholder = "Your password"
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
