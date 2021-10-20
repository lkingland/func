package utils

import (
	"errors"
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/util/validation"
)

// ErrInvalidName indicates the name did not pass function name validation.
type ErrInvalidFunctionName error

// ErrInvalidEnvVarName indicates the name did not pass env var name validation.
type ErrInvalidEnvVarName error

// ErrInvalidLabel indicates the name did not pass label key validation, or the value did not pass label value validation.
type ErrInvalidLabel error

// ValidateFunctionName validatest that the input name is a valid function name, ie. valid DNS-1123 label.
// It must consist of lower case alphanumeric characters or '-' and start and end with an alphanumeric character
// (e.g. 'my-name',  or '123-abc', regex used for validation is '[a-z0-9]([-a-z0-9]*[a-z0-9])?')
func ValidateFunctionName(name string) error {

	if errs := validation.IsDNS1123Label(name); len(errs) > 0 {
		// In case of invalid name the error is this:
		//	"a lowercase RFC 1123 label must consist of lower case alphanumeric characters or '-',
		//   and must start and end with an alphanumeric character (e.g. 'my-name',
		//   or '123-abc', regex used for validation is '[a-z0-9]([-a-z0-9]*[a-z0-9])?')"
		// Let's reuse it for our purposes, ie. replace "a lowercase RFC 1123 label" substring with "Function name" and the actual function name
		return ErrInvalidFunctionName(errors.New(strings.Replace(strings.Join(errs, ""), "a lowercase RFC 1123 label", fmt.Sprintf("Function name '%v'", name), 1)))
	}

	return nil
}

// ValidateEnvVarName validatest that the input name is a valid Kubernetes Environmet Variable name.
// It must  must consist of alphabetic characters, digits, '_', '-', or '.', and must not start with a digit
// (e.g. 'my.env-name',  or 'MY_ENV.NAME',  or 'MyEnvName1', regex used for validation is '[-._a-zA-Z][-._a-zA-Z0-9]*'))
func ValidateEnvVarName(name string) error {
	if errs := validation.IsEnvVarName(name); len(errs) > 0 {
		return ErrInvalidEnvVarName(errors.New(strings.Join(errs, "")))
	}

	return nil
}

// ValidateLabelKey validates that the input name is a valid Kubernetes key.
// Valid label names have two segments: an optional prefix and name, separated by a slash (/).
// The name segment is required and must be 63 characters or less, beginning and ending with
// an alphanumeric character ([a-z0-9A-Z]) with dashes (-), underscores (_), dots (.), and
// alphanumerics between. The prefix is optional. If specified, the prefix must be a DNS subdomain:
// a series of DNS labels separated by dots (.), not longer than 253 characters in total, followed
// by a slash (/).
func ValidateLabelKey(key string) error {
	errs := validation.IsQualifiedName(key)
	if len(errs) > 0 {
		return ErrInvalidLabel(errors.New(strings.Join(errs, "")))
	}
	return nil
}

// ValidateLabelValue ensures that the input is a Kubernetes label value
// Valid label values must be 63 characters or less (can be empty),
// unless empty, must begin and end with an alphanumeric character ([a-z0-9A-Z]),
// could contain dashes (-), underscores (_), dots (.), and alphanumerics between.
// Label values may also come from the environment and therefore, could be enclosed with {{}}
// Treat this as a special case.
func ValidateLabelValue(value string) error {
	var errs []string
	if !strings.HasPrefix(value, "{{") {
		errs = append(errs, validation.IsValidLabelValue(value)...)
	}
	if len(errs) > 0 {
		return ErrInvalidLabel(errors.New(strings.Join(errs, "")))
	}
	return nil
}
