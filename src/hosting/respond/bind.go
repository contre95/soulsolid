package respond

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v3"
)

// Validator validates bound request structs. Register it once as
// fiber.Config.StructValidator so every c.Bind() call checks `validate` tags.
type Validator struct {
	v *validator.Validate
}

// NewValidator returns a Validator that reports fields by their request name
// (form, query, uri or json tag) instead of the Go field name.
func NewValidator() *Validator {
	v := validator.New(validator.WithRequiredStructEnabled())
	v.RegisterTagNameFunc(func(f reflect.StructField) string {
		for _, tag := range []string{"form", "query", "uri", "json"} {
			if name, _, _ := strings.Cut(f.Tag.Get(tag), ","); name != "" && name != "-" {
				return name
			}
		}
		return f.Name
	})
	return &Validator{v: v}
}

// FieldError reports the first request field that failed validation.
type FieldError struct {
	Field string // request field path, e.g. "import.duplicates"
	Tag   string // failed validate tag, e.g. "oneof"
	Value any
}

func (e *FieldError) Error() string {
	if e.Tag == "required" {
		return e.Field + " is required"
	}
	return fmt.Sprintf("Invalid %s: %v", e.Field, e.Value)
}

// Validate implements fiber.StructValidator. Fiber calls it after every struct
// bind, so it first copies the bound strings: Fiber's form, query, header and
// cookie binders point them into fasthttp's request buffer (ignoring
// fiber.Config.Immutable), which is reused by later requests.
func (v *Validator) Validate(out any) error {
	detachStrings(reflect.ValueOf(out))
	err := v.v.Struct(out)
	var verrs validator.ValidationErrors
	if !errors.As(err, &verrs) || len(verrs) == 0 {
		return err
	}
	fe := verrs[0]
	// Namespace starts with the struct's type name, unless the struct is anonymous.
	field := fe.Namespace()
	if name := reflect.Indirect(reflect.ValueOf(out)).Type().Name(); name != "" {
		field = strings.TrimPrefix(field, name+".")
	}
	return &FieldError{Field: field, Tag: fe.Tag(), Value: fe.Value()}
}

// BindErr responds with a 400 toast describing why binding or validating the request failed.
func BindErr(c fiber.Ctx, err error) error {
	var fe *FieldError
	if errors.As(err, &fe) {
		return ToastErr(c, fiber.StatusBadRequest, fe.Error())
	}
	return ToastErr(c, fiber.StatusBadRequest, "Invalid request: "+err.Error())
}

// detachStrings replaces every string reachable from v with a copy.
func detachStrings(v reflect.Value) {
	switch v.Kind() {
	case reflect.Pointer, reflect.Interface:
		if !v.IsNil() {
			detachStrings(v.Elem())
		}
	case reflect.Struct:
		for i := range v.NumField() {
			if v.Type().Field(i).IsExported() {
				detachStrings(v.Field(i))
			}
		}
	case reflect.Slice, reflect.Array:
		for i := range v.Len() {
			detachStrings(v.Index(i))
		}
	case reflect.String:
		if v.CanSet() {
			v.SetString(strings.Clone(v.String()))
		}
	}
}
