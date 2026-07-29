package validation

import (
	"github.com/go-playground/validator/v10"
)

var Validate *validator.Validate = validator.New()

func ValidateStruct(s interface{}) error {
	err := Validate.Struct(s)
	if err != nil {
		if validationErrors, ok := err.(validator.ValidationErrors); ok {
			return NewValidationError(validationErrors)
		}
		return err
	}
	return nil
}
