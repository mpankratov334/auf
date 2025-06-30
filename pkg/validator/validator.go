package validator

import (
	"context"
	"errors"
	"github.com/go-playground/validator"
	"regexp"
	"strconv"
)

// Пакет валидации для входных данных с http

var global *validator.Validate

const (
	ErrInvalidFormat      = "Invalid format"
	ErrFieldRequired      = "Field is required"
	ErrFieldExceedsMaxLen = "Field exceeds maximum length"
	ErrFieldBelowMinLen   = "Field is below minimum length"
	ErrFieldExceedsMaxVal = "Field exceeds maximum value"
	ErrFieldBelowMinVal   = "Field is below minimum value"
	ErrUnknownValidation  = "Unknown validation error"
	ErrInvalidIntString   = "Invalid int format"
)

func init() {
	SetValidator(New())
}

func New() *validator.Validate {
	v := validator.New()
	_ = v.RegisterValidation("tag", validateTag)
	_ = v.RegisterValidation("intString", validateIntString)
	return v
}

func SetValidator(v *validator.Validate) {
	global = v
}

func Validator() *validator.Validate {
	return global
}

func validateTag(fl validator.FieldLevel) bool {
	re, _ := regexp.Compile(`^#[a-z0-9_\-]+$`)
	return re.MatchString(fl.Field().String())
}

func validateIntString(fl validator.FieldLevel) bool {
	rawId := fl.Field().String()

	_, err := strconv.Atoi(rawId)
	if err != nil {
		return false
	}

	return true
}

func Validate(ctx context.Context, structure any) error {
	return parseValidationErrors(Validator().StructCtx(ctx, structure))
}

func parseValidationErrors(err error) error {
	if err == nil {
		return nil
	}

	vErrors, ok := err.(validator.ValidationErrors)
	if !ok || len(vErrors) == 0 {
		return nil
	}

	validationError := vErrors[0]
	var validationErrorDescription string
	switch validationError.Tag() {
	case "tag":
		validationErrorDescription = ErrInvalidFormat
	case "required":
		validationErrorDescription = ErrFieldRequired
	case "max":
		validationErrorDescription = ErrFieldExceedsMaxLen
	case "min":
		validationErrorDescription = ErrFieldBelowMinLen
	case "lt", "lte":
		validationErrorDescription = ErrFieldExceedsMaxVal
	case "gt", "gte":
		validationErrorDescription = ErrFieldBelowMinVal
	case "intString":
		validationErrorDescription = ErrInvalidIntString
	default:
		validationErrorDescription = ErrUnknownValidation
	}

	return errors.New(validationErrorDescription + ": " + validationError.Namespace())
}
