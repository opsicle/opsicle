package org

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
		Short:        'u',
		DefaultValue: "http://localhost:54321",
		Usage:        "defines the url where the controller service is accessible at",
		Type:         cli.FlagTypeString,
	},
	{
		Name:         "org-code",
		DefaultValue: "",
		Usage:        "code of the orgnaisation to be created",
		Type:         cli.FlagTypeString,
	},
	{
		Name:         "org-name",
		DefaultValue: "",
		Usage:        "name of the orgnaisation to be created",
		Type:         cli.FlagTypeString,
	},
}

func init() {
	flags.AddToCommand(Command)
}

var Command = &cobra.Command{
	Use:     "org",
	Aliases: []string{"o"},
	Short:   "Creates a new organisation",
	PreRun: func(cmd *cobra.Command, args []string) {
		flags.BindViper(cmd)
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		controllerUrl := viper.GetString("controller-url")
		orgCode := viper.GetString("org-code")
		orgName := viper.GetString("org-name")

		sessionToken, sessionFilePath, err := controller.GetSessionToken()
		if err != nil {
			return fmt.Errorf("failed to get a session token: %s", err)
		}
		logrus.Debugf("loaded credentials from path[%s]", sessionFilePath)
		model := getModel(orgName, orgCode)
		program := tea.NewProgram(model)
		if _, err := program.Run(); err != nil {
			return fmt.Errorf("failed to get user input: %s", err)
		}
		orgCodeIsSet, inputOrgCode := model.GetOrganisationCode()
		if orgCodeIsSet {
			orgCode = inputOrgCode
		}
		orgNameIsSet, inputOrgName := model.GetOrganisationName()
		if orgNameIsSet {
			orgName = inputOrgName
		}

		client, err := controller.NewClient(controller.NewClientOpts{
			ControllerUrl: controllerUrl,
			BearerAuth: &controller.NewClientBearerAuthOpts{
				Token: sessionToken,
			},
			Id: "opsicle/create/org",
		})
		if err != nil {
			return fmt.Errorf("failed to create a client: %s", err)
		}

		createOrgOutput, err := client.CreateOrgV1(controller.CreateOrgV1Input{
			Name: orgName,
			Code: orgCode,
		})
		if err != nil {
			return fmt.Errorf("failed to create an org: %s", err)
		}
		logrus.Infof("successfully created org[%s]", createOrgOutput.Code)

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
	outputs    map[string]string
	outputMap  map[string]int
	cursorMode cursor.Mode
}

func getModel(orgName, orgCode string) model {
	outputs := map[string]string{}
	outputMap := map[string]int{}
	inputs := []func(*textinput.Model){}
	if orgName == "" {
		currentInputIndex := len(inputs)
		inputs = append(inputs, func(t *textinput.Model) {
			t.Width = 64
			t.CharLimit = 256
			t.Placeholder = "Organisation name"
			outputs["orgName"] = ""
			outputMap["orgName"] = currentInputIndex
		})
	}
	if orgCode == "" {
		currentInputIndex := len(inputs)
		inputs = append(inputs, func(t *textinput.Model) {
			t.Width = 64
			t.CharLimit = 256
			t.Placeholder = "Organisation code"
			outputs["orgCode"] = ""
			outputMap["orgCode"] = currentInputIndex
		})
	}
	m := model{
		inputs:    make([]textinput.Model, len(inputs)),
		outputs:   outputs,
		outputMap: outputMap,
	}

	var t textinput.Model
	for i := range m.inputs {
		t = textinput.New()
		t.Cursor.Style = cursorStyle
		t.CharLimit = 32

		switch i {
		case 0:
			t.Focus()
			t.PromptStyle = focusedStyle
			t.TextStyle = focusedStyle
		}
		inputs[i](&t)
		m.inputs[i] = t
	}

	return m
}

func (m model) GetOrganisationCode() (bool, string) {
	orgCodeIndex, ok := m.outputMap["orgCode"]
	if !ok {
		return false, ""
	}
	return true, m.inputs[orgCodeIndex].Value()
}

func (m model) GetOrganisationName() (bool, string) {
	orgNameIndex, ok := m.outputMap["orgName"]
	if !ok {
		return false, ""
	}
	return true, m.inputs[orgNameIndex].Value()
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
