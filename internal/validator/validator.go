package validator

import (
    "regexp"

    "github.com/go-playground/validator/v10"
)

var (
    EmailRX = regexp.MustCompile("^[a-zA-Z0-9.!#$%&'*+\\/=?^_`{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$")
)

var validate *validator.Validate

func init() {
    validate = validator.New()
    
    // Register custom validation for email
    validate.RegisterValidation("email", validateEmail)
}

type Validator struct {
    Errors map[string]string
}

func New() *Validator {
    return &Validator{Errors: make(map[string]string)}
}

func (v *Validator) Valid() bool {
    return len(v.Errors) == 0
}

func (v *Validator) AddError(key, message string) {
    if _, exists := v.Errors[key]; !exists {
        v.Errors[key] = message
    }
}

func (v *Validator) Check(ok bool, key, message string) {
    if !ok {
        v.AddError(key, message)
    }
}

func (v *Validator) ValidateStruct(s interface{}) {
    err := validate.Struct(s)
    if err != nil {
        for _, err := range err.(validator.ValidationErrors) {
            v.AddError(err.Field(), err.Tag())
        }
    }
}

// Custom validation function for email
func validateEmail(fl validator.FieldLevel) bool {
    return EmailRX.MatchString(fl.Field().String())
}