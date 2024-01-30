package main

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

var (
	ErrInvalidChar                  = errors.New("invalid char")
	ErrInvalidWidth                 = errors.New("invalid width")
	ErrInvalidPrecision             = errors.New("invalid precision")
	ErrInvalidSpecifier             = errors.New("invalid specifier")
	ErrUnexpectedText               = errors.New("unexpected text")
	ErrInvalidArgumentName          = errors.New("invalid argument name")
	ErrNoArgumentName               = errors.New("no argument name")
	ErrInvalidVariableName          = errors.New("invalid variable name")
	ErrNoVariableName               = errors.New("no variable name")
	ErrUnexpectedEndOfFormat        = errors.New("unexpected end of format")
	ErrUnexpectedChar               = errors.New("unexpected char")
	ErrNoClosingBracket             = errors.New("no closing bracket")
	ErrInvalidFieldType             = errors.New("invalid field type")
	ErrUnknownField                 = errors.New("unknown field")
	ErrTypesDontMatch               = errors.New("types don't match")
	ErrCouldNotUnmarshal            = errors.New("could not unmarshal")
	ErrCouldNotProcess              = errors.New("could not process")
	ErrFieldNotSpecified            = errors.New("field not specified")
	ErrFieldsSpecifiedAtTheSameTime = errors.New("fields can't be specified at the same time")
	ErrFieldsNotSpecified           = errors.New("fields not specified")
	ErrVariableNotSpecified         = errors.New("variable not specified")
	ErrInvalidFilename              = errors.New("invalid filename")
	ErrInvalidPattern               = errors.New("invalid pattern")
	ErrInvalidLanguage              = errors.New("invalid language")
	ErrFilenameDoesNotMatch         = errors.New("filename doesn't match the pattern")
	ErrCouldNotReadFile             = errors.New("could not read file")
	ErrUnsupportedFileExtension     = errors.New("unsupported file extension")
	ErrCouldNotUnmarshalFile        = errors.New("could not unmarshal file")
	ErrCouldNotParseFile            = errors.New("could not parse file")
	ErrInvalidLocalization          = errors.New("invalid localization")
	ErrCouldNotCreateFile           = errors.New("could not create file")
	ErrCouldNotCreateDirectory      = errors.New("could not create directory")
	ErrCouldNotWriteToFile          = errors.New("could not write to file")
	ErrNoLocalizationsFound         = errors.New("no localizations found")
)

type ErrorValue struct {
	Kind byte
	Str  string
	Char rune
}

func ErrorValueStr(str string) ErrorValue {
	return ErrorValue{
		Kind: 's',
		Str:  str,
	}
}

func ErrorValueChar(c rune) ErrorValue {
	return ErrorValue{
		Kind: 'c',
		Char: c,
	}
}

type ErrorExpected struct {
	Kind    byte
	Str     string
	Char    rune
	AnyStr  []string
	AnyChar []rune
}

func ErrorExpectedChar(c rune) ErrorExpected {
	return ErrorExpected{
		Kind: 'c',
		Char: c,
	}
}

func ErrorExpectedStr(str string) ErrorExpected {
	return ErrorExpected{
		Kind: 's',
		Str:  str,
	}
}

func ErrorExpectedAnyStr(strs ...string) ErrorExpected {
	return ErrorExpected{
		Kind:   'S',
		AnyStr: strs,
	}
}

func ErrorExpectedAnyChar(chars ...rune) ErrorExpected {
	return ErrorExpected{
		Kind:    'C',
		AnyChar: chars,
	}
}

type ErrorPosition int

type ErrorWrapped error

type Error struct {
	ErrKind  error
	Value    ErrorValue
	Expected ErrorExpected
	Pos      ErrorPosition
	Wrapped  ErrorWrapped
}

func NewError(errKind error, options ...any) error {
	e := &Error{
		ErrKind: errKind,
		Pos:     -1,
	}

	for _, opt := range options {
		switch p := opt.(type) {
		case ErrorValue:
			e.Value = p
		case ErrorExpected:
			e.Expected = p
		case ErrorPosition:
			e.Pos = p
		case ErrorWrapped:
			e.Wrapped = p
		}
	}

	return e
}

func (e *Error) Error() string {
	var b strings.Builder

	b.WriteString(e.ErrKind.Error())

	if e.Value.Kind != 0 {
		b.WriteByte(' ')

		switch e.Value.Kind {
		case 's':
			b.WriteString(strconv.Quote(e.Value.Str))
		case 'c':
			b.WriteString(strconv.QuoteRune(e.Value.Char))
		}
	}

	if e.Pos != -1 {
		b.WriteString(" at position ")
		b.WriteString(strconv.Itoa(int(e.Pos)))
	}

	if e.Expected.Kind != 0 {
		b.WriteString(", expected ")

		if e.Expected.Kind >= 'A' && e.Expected.Kind <= 'Z' {
			b.WriteString("any of ")
		}

		switch e.Expected.Kind {
		case 's':
			b.WriteString(strconv.Quote(e.Expected.Str))
		case 'S':
			for i, str := range e.Expected.AnyStr {
				if i != 0 {
					b.WriteString(", ")
				}
				b.WriteString(strconv.Quote(str))
			}
		case 'c':
			b.WriteString(strconv.QuoteRune(e.Expected.Char))
		case 'C':
			for i, c := range e.Expected.AnyChar {
				if i != 0 {
					b.WriteString(", ")
				}
				b.WriteString(strconv.QuoteRune(c))
			}
		}
	}

	if e.Wrapped != nil {
		b.WriteString(": ")
		b.WriteString(e.Wrapped.Error())
	}

	return b.String()
}

func (e *Error) Unwrap() error {
	return e.Wrapped
}

type FieldError struct {
	ErrKind error
	Field   string
	Wrapped error
}

func NewFieldError(errorKind error, field string, err error) error {
	if err, ok := err.(*FieldError); ok && errors.Is(errorKind, err.ErrKind) {
		return &FieldError{
			ErrKind: err.ErrKind,
			Field:   field + "." + err.Field,
			Wrapped: err.Wrapped,
		}
	}

	return &FieldError{
		ErrKind: errorKind,
		Field:   field,
		Wrapped: err,
	}
}

func (e *FieldError) Error() string {
	return fmt.Sprintf("%v %s: %v", e.ErrKind, e.Field, e.Wrapped)
}

func (e *FieldError) Unwrap() error {
	return e.Wrapped
}

type MessageNotSpecifiedError struct {
	Message string
}

func NewMessageNotSpecifiedError(message string) error {
	return &MessageNotSpecifiedError{
		Message: message,
	}
}

func (e *MessageNotSpecifiedError) Error() string {
	return "message \"" + e.Message + "\" not specified"
}

type DuplicateMessageError struct {
	Message string
}

func NewDuplicateMessageError(message string) error {
	return &DuplicateMessageError{
		Message: message,
	}
}

func (e *DuplicateMessageError) Error() string {
	return "duplicate message \"" + e.Message + "\""
}
