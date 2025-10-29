package config

import (
	"strings"

	"github.com/go-playground/validator/v10"
)

func validateStartsWith(fl validator.FieldLevel) bool {
	value := fl.Field().String()
	prefix := fl.Param()
	if prefix == "" {
		return true
	}
	return strings.HasPrefix(value, prefix)
}
