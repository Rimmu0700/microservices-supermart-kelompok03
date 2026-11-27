package validator

import (
	"errors"
	"strings"
)

type ValidationErrors map[string]string

func (v ValidationErrors) HasErrors() bool {
	return len(v) > 0
}

func (v ValidationErrors) Error() string {
	var errMsgs []string
	for field, msg := range v {
		errMsgs = append(errMsgs, field+": "+msg)
	}
	return strings.Join(errMsgs, ", ")
}

func ValidateRequired(field, value string, errs ValidationErrors) {
	if strings.TrimSpace(value) == "" {
		errs[field] = field + " is required"
	}
}

func ValidatePositive(field string, value float64, errs ValidationErrors) {
	if value <= 0 {
		errs[field] = field + " must be greater than 0"
	}
}

func ValidateNonNegative(field string, value int, errs ValidationErrors) {
	if value < 0 {
		errs[field] = field + " cannot be negative"
	}
}

func Check(errs ValidationErrors) error {
	if errs.HasErrors() {
		return errors.New(errs.Error())
	}
	return nil
}
