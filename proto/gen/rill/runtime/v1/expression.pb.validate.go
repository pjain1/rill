// Code generated by protoc-gen-validate. DO NOT EDIT.
// source: rill/runtime/v1/expression.proto

package runtimev1

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"net/mail"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"google.golang.org/protobuf/types/known/anypb"
)

// ensure the imports are used
var (
	_ = bytes.MinRead
	_ = errors.New("")
	_ = fmt.Print
	_ = utf8.UTFMax
	_ = (*regexp.Regexp)(nil)
	_ = (*strings.Reader)(nil)
	_ = net.IPv4len
	_ = time.Duration(0)
	_ = (*url.URL)(nil)
	_ = (*mail.Address)(nil)
	_ = anypb.Any{}
	_ = sort.Sort
)

// Validate checks the field values on Expression with the rules defined in the
// proto definition for this message. If any rules are violated, the first
// error encountered is returned, or nil if there are no violations.
func (m *Expression) Validate() error {
	return m.validate(false)
}

// ValidateAll checks the field values on Expression with the rules defined in
// the proto definition for this message. If any rules are violated, the
// result is a list of violation errors wrapped in ExpressionMultiError, or
// nil if none found.
func (m *Expression) ValidateAll() error {
	return m.validate(true)
}

func (m *Expression) validate(all bool) error {
	if m == nil {
		return nil
	}

	var errors []error

	switch v := m.Expression.(type) {
	case *Expression_Ident:
		if v == nil {
			err := ExpressionValidationError{
				field:  "Expression",
				reason: "oneof value cannot be a typed-nil",
			}
			if !all {
				return err
			}
			errors = append(errors, err)
		}
		// no validation rules for Ident
	case *Expression_Val:
		if v == nil {
			err := ExpressionValidationError{
				field:  "Expression",
				reason: "oneof value cannot be a typed-nil",
			}
			if !all {
				return err
			}
			errors = append(errors, err)
		}

		if all {
			switch v := interface{}(m.GetVal()).(type) {
			case interface{ ValidateAll() error }:
				if err := v.ValidateAll(); err != nil {
					errors = append(errors, ExpressionValidationError{
						field:  "Val",
						reason: "embedded message failed validation",
						cause:  err,
					})
				}
			case interface{ Validate() error }:
				if err := v.Validate(); err != nil {
					errors = append(errors, ExpressionValidationError{
						field:  "Val",
						reason: "embedded message failed validation",
						cause:  err,
					})
				}
			}
		} else if v, ok := interface{}(m.GetVal()).(interface{ Validate() error }); ok {
			if err := v.Validate(); err != nil {
				return ExpressionValidationError{
					field:  "Val",
					reason: "embedded message failed validation",
					cause:  err,
				}
			}
		}

	case *Expression_Cond:
		if v == nil {
			err := ExpressionValidationError{
				field:  "Expression",
				reason: "oneof value cannot be a typed-nil",
			}
			if !all {
				return err
			}
			errors = append(errors, err)
		}

		if all {
			switch v := interface{}(m.GetCond()).(type) {
			case interface{ ValidateAll() error }:
				if err := v.ValidateAll(); err != nil {
					errors = append(errors, ExpressionValidationError{
						field:  "Cond",
						reason: "embedded message failed validation",
						cause:  err,
					})
				}
			case interface{ Validate() error }:
				if err := v.Validate(); err != nil {
					errors = append(errors, ExpressionValidationError{
						field:  "Cond",
						reason: "embedded message failed validation",
						cause:  err,
					})
				}
			}
		} else if v, ok := interface{}(m.GetCond()).(interface{ Validate() error }); ok {
			if err := v.Validate(); err != nil {
				return ExpressionValidationError{
					field:  "Cond",
					reason: "embedded message failed validation",
					cause:  err,
				}
			}
		}

	case *Expression_Subquery:
		if v == nil {
			err := ExpressionValidationError{
				field:  "Expression",
				reason: "oneof value cannot be a typed-nil",
			}
			if !all {
				return err
			}
			errors = append(errors, err)
		}

		if all {
			switch v := interface{}(m.GetSubquery()).(type) {
			case interface{ ValidateAll() error }:
				if err := v.ValidateAll(); err != nil {
					errors = append(errors, ExpressionValidationError{
						field:  "Subquery",
						reason: "embedded message failed validation",
						cause:  err,
					})
				}
			case interface{ Validate() error }:
				if err := v.Validate(); err != nil {
					errors = append(errors, ExpressionValidationError{
						field:  "Subquery",
						reason: "embedded message failed validation",
						cause:  err,
					})
				}
			}
		} else if v, ok := interface{}(m.GetSubquery()).(interface{ Validate() error }); ok {
			if err := v.Validate(); err != nil {
				return ExpressionValidationError{
					field:  "Subquery",
					reason: "embedded message failed validation",
					cause:  err,
				}
			}
		}

	default:
		_ = v // ensures v is used
	}

	if len(errors) > 0 {
		return ExpressionMultiError(errors)
	}

	return nil
}

// ExpressionMultiError is an error wrapping multiple validation errors
// returned by Expression.ValidateAll() if the designated constraints aren't met.
type ExpressionMultiError []error

// Error returns a concatenation of all the error messages it wraps.
func (m ExpressionMultiError) Error() string {
	var msgs []string
	for _, err := range m {
		msgs = append(msgs, err.Error())
	}
	return strings.Join(msgs, "; ")
}

// AllErrors returns a list of validation violation errors.
func (m ExpressionMultiError) AllErrors() []error { return m }

// ExpressionValidationError is the validation error returned by
// Expression.Validate if the designated constraints aren't met.
type ExpressionValidationError struct {
	field  string
	reason string
	cause  error
	key    bool
}

// Field function returns field value.
func (e ExpressionValidationError) Field() string { return e.field }

// Reason function returns reason value.
func (e ExpressionValidationError) Reason() string { return e.reason }

// Cause function returns cause value.
func (e ExpressionValidationError) Cause() error { return e.cause }

// Key function returns key value.
func (e ExpressionValidationError) Key() bool { return e.key }

// ErrorName returns error name.
func (e ExpressionValidationError) ErrorName() string { return "ExpressionValidationError" }

// Error satisfies the builtin error interface
func (e ExpressionValidationError) Error() string {
	cause := ""
	if e.cause != nil {
		cause = fmt.Sprintf(" | caused by: %v", e.cause)
	}

	key := ""
	if e.key {
		key = "key for "
	}

	return fmt.Sprintf(
		"invalid %sExpression.%s: %s%s",
		key,
		e.field,
		e.reason,
		cause)
}

var _ error = ExpressionValidationError{}

var _ interface {
	Field() string
	Reason() string
	Key() bool
	Cause() error
	ErrorName() string
} = ExpressionValidationError{}

// Validate checks the field values on Condition with the rules defined in the
// proto definition for this message. If any rules are violated, the first
// error encountered is returned, or nil if there are no violations.
func (m *Condition) Validate() error {
	return m.validate(false)
}

// ValidateAll checks the field values on Condition with the rules defined in
// the proto definition for this message. If any rules are violated, the
// result is a list of violation errors wrapped in ConditionMultiError, or nil
// if none found.
func (m *Condition) ValidateAll() error {
	return m.validate(true)
}

func (m *Condition) validate(all bool) error {
	if m == nil {
		return nil
	}

	var errors []error

	if _, ok := Operation_name[int32(m.GetOp())]; !ok {
		err := ConditionValidationError{
			field:  "Op",
			reason: "value must be one of the defined enum values",
		}
		if !all {
			return err
		}
		errors = append(errors, err)
	}

	for idx, item := range m.GetExprs() {
		_, _ = idx, item

		if all {
			switch v := interface{}(item).(type) {
			case interface{ ValidateAll() error }:
				if err := v.ValidateAll(); err != nil {
					errors = append(errors, ConditionValidationError{
						field:  fmt.Sprintf("Exprs[%v]", idx),
						reason: "embedded message failed validation",
						cause:  err,
					})
				}
			case interface{ Validate() error }:
				if err := v.Validate(); err != nil {
					errors = append(errors, ConditionValidationError{
						field:  fmt.Sprintf("Exprs[%v]", idx),
						reason: "embedded message failed validation",
						cause:  err,
					})
				}
			}
		} else if v, ok := interface{}(item).(interface{ Validate() error }); ok {
			if err := v.Validate(); err != nil {
				return ConditionValidationError{
					field:  fmt.Sprintf("Exprs[%v]", idx),
					reason: "embedded message failed validation",
					cause:  err,
				}
			}
		}

	}

	if len(errors) > 0 {
		return ConditionMultiError(errors)
	}

	return nil
}

// ConditionMultiError is an error wrapping multiple validation errors returned
// by Condition.ValidateAll() if the designated constraints aren't met.
type ConditionMultiError []error

// Error returns a concatenation of all the error messages it wraps.
func (m ConditionMultiError) Error() string {
	var msgs []string
	for _, err := range m {
		msgs = append(msgs, err.Error())
	}
	return strings.Join(msgs, "; ")
}

// AllErrors returns a list of validation violation errors.
func (m ConditionMultiError) AllErrors() []error { return m }

// ConditionValidationError is the validation error returned by
// Condition.Validate if the designated constraints aren't met.
type ConditionValidationError struct {
	field  string
	reason string
	cause  error
	key    bool
}

// Field function returns field value.
func (e ConditionValidationError) Field() string { return e.field }

// Reason function returns reason value.
func (e ConditionValidationError) Reason() string { return e.reason }

// Cause function returns cause value.
func (e ConditionValidationError) Cause() error { return e.cause }

// Key function returns key value.
func (e ConditionValidationError) Key() bool { return e.key }

// ErrorName returns error name.
func (e ConditionValidationError) ErrorName() string { return "ConditionValidationError" }

// Error satisfies the builtin error interface
func (e ConditionValidationError) Error() string {
	cause := ""
	if e.cause != nil {
		cause = fmt.Sprintf(" | caused by: %v", e.cause)
	}

	key := ""
	if e.key {
		key = "key for "
	}

	return fmt.Sprintf(
		"invalid %sCondition.%s: %s%s",
		key,
		e.field,
		e.reason,
		cause)
}

var _ error = ConditionValidationError{}

var _ interface {
	Field() string
	Reason() string
	Key() bool
	Cause() error
	ErrorName() string
} = ConditionValidationError{}

// Validate checks the field values on Subquery with the rules defined in the
// proto definition for this message. If any rules are violated, the first
// error encountered is returned, or nil if there are no violations.
func (m *Subquery) Validate() error {
	return m.validate(false)
}

// ValidateAll checks the field values on Subquery with the rules defined in
// the proto definition for this message. If any rules are violated, the
// result is a list of violation errors wrapped in SubqueryMultiError, or nil
// if none found.
func (m *Subquery) ValidateAll() error {
	return m.validate(true)
}

func (m *Subquery) validate(all bool) error {
	if m == nil {
		return nil
	}

	var errors []error

	// no validation rules for Dimension

	if all {
		switch v := interface{}(m.GetWhere()).(type) {
		case interface{ ValidateAll() error }:
			if err := v.ValidateAll(); err != nil {
				errors = append(errors, SubqueryValidationError{
					field:  "Where",
					reason: "embedded message failed validation",
					cause:  err,
				})
			}
		case interface{ Validate() error }:
			if err := v.Validate(); err != nil {
				errors = append(errors, SubqueryValidationError{
					field:  "Where",
					reason: "embedded message failed validation",
					cause:  err,
				})
			}
		}
	} else if v, ok := interface{}(m.GetWhere()).(interface{ Validate() error }); ok {
		if err := v.Validate(); err != nil {
			return SubqueryValidationError{
				field:  "Where",
				reason: "embedded message failed validation",
				cause:  err,
			}
		}
	}

	if all {
		switch v := interface{}(m.GetHaving()).(type) {
		case interface{ ValidateAll() error }:
			if err := v.ValidateAll(); err != nil {
				errors = append(errors, SubqueryValidationError{
					field:  "Having",
					reason: "embedded message failed validation",
					cause:  err,
				})
			}
		case interface{ Validate() error }:
			if err := v.Validate(); err != nil {
				errors = append(errors, SubqueryValidationError{
					field:  "Having",
					reason: "embedded message failed validation",
					cause:  err,
				})
			}
		}
	} else if v, ok := interface{}(m.GetHaving()).(interface{ Validate() error }); ok {
		if err := v.Validate(); err != nil {
			return SubqueryValidationError{
				field:  "Having",
				reason: "embedded message failed validation",
				cause:  err,
			}
		}
	}

	if len(errors) > 0 {
		return SubqueryMultiError(errors)
	}

	return nil
}

// SubqueryMultiError is an error wrapping multiple validation errors returned
// by Subquery.ValidateAll() if the designated constraints aren't met.
type SubqueryMultiError []error

// Error returns a concatenation of all the error messages it wraps.
func (m SubqueryMultiError) Error() string {
	var msgs []string
	for _, err := range m {
		msgs = append(msgs, err.Error())
	}
	return strings.Join(msgs, "; ")
}

// AllErrors returns a list of validation violation errors.
func (m SubqueryMultiError) AllErrors() []error { return m }

// SubqueryValidationError is the validation error returned by
// Subquery.Validate if the designated constraints aren't met.
type SubqueryValidationError struct {
	field  string
	reason string
	cause  error
	key    bool
}

// Field function returns field value.
func (e SubqueryValidationError) Field() string { return e.field }

// Reason function returns reason value.
func (e SubqueryValidationError) Reason() string { return e.reason }

// Cause function returns cause value.
func (e SubqueryValidationError) Cause() error { return e.cause }

// Key function returns key value.
func (e SubqueryValidationError) Key() bool { return e.key }

// ErrorName returns error name.
func (e SubqueryValidationError) ErrorName() string { return "SubqueryValidationError" }

// Error satisfies the builtin error interface
func (e SubqueryValidationError) Error() string {
	cause := ""
	if e.cause != nil {
		cause = fmt.Sprintf(" | caused by: %v", e.cause)
	}

	key := ""
	if e.key {
		key = "key for "
	}

	return fmt.Sprintf(
		"invalid %sSubquery.%s: %s%s",
		key,
		e.field,
		e.reason,
		cause)
}

var _ error = SubqueryValidationError{}

var _ interface {
	Field() string
	Reason() string
	Key() bool
	Cause() error
	ErrorName() string
} = SubqueryValidationError{}
