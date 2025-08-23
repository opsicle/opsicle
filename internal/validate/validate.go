package validate

import (
	"errors"
	"fmt"
)

type StringRule func(string) error
type RuneRule func(rune, rune) error

func andR(input ...RuneRule) RuneRule {
	return func(r rune, p rune) error {
		errs := []error{}
		for _, runValidator := range input {
			if err := runValidator(r, p); err != nil {
				errs = append(errs, err)
			}
		}
		if len(errs) > 0 {
			return errors.Join(errs...)
		}
		return nil
	}
}

func andS(input ...StringRule) StringRule {
	return func(s string) error {
		errs := []error{}
		for _, runValidator := range input {
			if err := runValidator(s); err != nil {
				errs = append(errs, err)
			}
		}
		if len(errs) > 0 {
			return errors.Join(errs...)
		}
		return nil
	}
}

func orR(input ...RuneRule) RuneRule {
	return func(r rune, p rune) error {
		errs := []error{}
		for _, runValidator := range input {
			if err := runValidator(r, p); err != nil {
				errs = append(errs, err)
			} else {
				return nil
			}
		}
		return errors.Join(errs...)
	}
}

func orS(input ...StringRule) StringRule {
	return func(s string) error {
		errs := []error{}
		for _, runValidator := range input {
			if err := runValidator(s); err != nil {
				errs = append(errs, err)
			} else {
				return nil
			}
		}
		return errors.Join(errs...)
	}
}

func do(input string, validators ...any) error {
	errs := map[error]struct{}{}

	stringRules := []StringRule{}
	runeRules := []RuneRule{}

	for _, validator := range validators {
		if stringRuleValidator, ok := validator.(StringRule); ok {
			stringRules = append(stringRules, stringRuleValidator)
		} else if runeRuleValidator, ok := validator.(RuneRule); ok {
			runeRules = append(runeRules, runeRuleValidator)
		}
	}

	for _, stringRule := range stringRules {
		if err := stringRule(input); err != nil {
			errs[err] = struct{}{}
		}
	}

	var p rune
	for _, i := range input {
		for _, runeRule := range runeRules {
			if err := runeRule(i, p); err != nil {
				errs[err] = struct{}{}
			}
		}
		p = i
	}

	if len(errs) > 0 {
		outputErrors := []error{}
		for err := range errs {
			outputErrors = append(outputErrors, err)
		}
		fmt.Println(outputErrors)
		return errors.Join(outputErrors...)
	}
	return nil
}
