package validate

import "github.com/google/uuid"

func Uuid(input string) error {
	if _, err := uuid.Parse(input); err != nil {
		return ErrorInvalidUuid
	}
	return nil
}
