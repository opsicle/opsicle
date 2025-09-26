package cli

import (
	"fmt"
	"opsicle/internal/validate"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
)

type formValidatorOpts struct {
	IsRequired bool
}

func getBooleanValidator(opts formValidatorOpts) textinput.ValidateFunc {
	return func(s string) error {
		if opts.IsRequired && s == "" {
			return fmt.Errorf("this field is required")
		} else if s != "" {
			val := strings.ToLower(s)
			switch val {
			case "1", "true", "yes", "0", "false", "no":
				return nil
			}
			return fmt.Errorf("this field should be a boolean")
		}
		return nil
	}
}

func getIntegerValidator(opts formValidatorOpts) textinput.ValidateFunc {
	return func(s string) error {
		if opts.IsRequired && s == "" {
			return fmt.Errorf("this field is required")
		} else if s != "" {
			if _, err := strconv.Atoi(s); err != nil {
				return fmt.Errorf("this field should be an integer: %w", err)
			}
		}
		return nil
	}
}

func getFloatValidator(opts formValidatorOpts) textinput.ValidateFunc {
	return func(s string) error {
		if opts.IsRequired && s == "" {
			return fmt.Errorf("this field is required")
		} else if s != "" {
			if _, err := strconv.ParseFloat(s, 64); err != nil {
				return fmt.Errorf("this field should be a floating point number: %w", err)
			}
		}
		return nil
	}
}

func getStringValidator(opts formValidatorOpts) textinput.ValidateFunc {
	return func(s string) error {
		if opts.IsRequired && s == "" {
			return fmt.Errorf("this field is required")
		}
		return nil
	}
}

func getEmailValidator(opts formValidatorOpts) textinput.ValidateFunc {
	return func(s string) error {
		if opts.IsRequired && s == "" {
			return fmt.Errorf("this field is required")
		}
		return validate.Email(s)
	}
}
