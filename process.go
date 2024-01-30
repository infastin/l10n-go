package main

import (
	"strings"
)

type Value interface {
	IsZero() bool
	Process(scope *MessageScope) (err error)
	GetArgumentNames() (names []string)
}

type FieldValue struct {
	Name  string
	Value Value
}

func processMessages(msgs []Message) (scopes []MessageScope, err error) {
	for i := 0; i < len(msgs); i++ {
		scope, err := processMessage(&msgs[i])
		if err != nil {
			return nil, NewFieldError(ErrCouldNotProcess, msgs[i].Name, err)
		}

		scopes = append(scopes, scope)
	}

	return scopes, nil
}

func processMessage(msg *Message) (scope MessageScope, err error) {
	scope = MessageScope{
		Name:   msg.Name,
		Plural: msg.Plural,
		String: msg.String,
	}

	fields := []FieldValue{
		{"plural", &msg.Plural},
		{"string", msg.String},
	}

	err = checkFielsXor(fields)
	if err != nil {
		return MessageScope{}, NewFieldError(ErrCouldNotProcess, getFieldNames(fields), err)
	}

	for i := 0; i < len(msg.Variables); i++ {
		var argNames []string
		values := []Value{&msg.Variables[i].Plural, msg.Variables[i].String}

		for _, val := range values {
			if !val.IsZero() {
				argNames = val.GetArgumentNames()
				break
			}
		}

		scope.Variables = append(scope.Variables, VariableScope{
			Variable:      msg.Variables[i],
			ArgumentNames: argNames,
		})
	}

	for i := 0; i < len(scope.Variables); i++ {
		err = processVariable(&scope, &scope.Variables[i])
		if err != nil {
			return MessageScope{}, NewFieldError(ErrCouldNotProcess, scope.Variables[i].Name, err)
		}
	}

	err = processFields(&scope, fields)
	if err != nil {
		return MessageScope{}, err
	}

	for i := 0; i < len(scope.Arguments); i++ {
		arg := &scope.Arguments[i]
		if arg.GoType.IsZero() {
			arg.GoType = config.SpecifierToGoType['s']
		}
	}

	return scope, nil
}

func processVariable(scope *MessageScope, variable *VariableScope) (err error) {
	fields := []FieldValue{
		{"plural", &variable.Plural},
		{"string", variable.String},
	}

	err = checkFielsXor(fields)
	if err != nil {
		return NewFieldError(ErrCouldNotProcess, getFieldNames(fields), err)
	}

	err = processFields(scope, fields)
	if err != nil {
		return err
	}

	return nil
}

func (p *Plural) Process(scope *MessageScope) (err error) {
	return processPlural(scope, p)
}

func processPlural(scope *MessageScope, plural *Plural) (err error) {
	if plural.Arg == "" {
		return NewFieldError(ErrCouldNotProcess, "arg", ErrFieldNotSpecified)
	}

	goType := config.SpecifierToGoType['d']

	err = processArg(scope, plural.Arg, goType)
	if err != nil {
		return NewFieldError(ErrCouldNotProcess, "arg", err)
	}

	fields := []struct {
		Name        string
		FormatParts FormatParts
	}{
		{"zero", plural.Zero},
		{"one", plural.One},
		{"many", plural.Many},
		{"other", plural.Other},
	}

	for _, field := range fields {
		err = processFormatParts(scope, field.FormatParts)
		if err != nil {
			return NewFieldError(ErrCouldNotProcess, field.Name, err)
		}
	}

	return nil
}

func (f FormatParts) Process(scope *MessageScope) (err error) {
	return processFormatParts(scope, f)
}

func processFormatParts(scope *MessageScope, parts FormatParts) (err error) {
	for _, cell := range parts {
		switch cell := cell.(type) {
		case ArgInfo:
			var goType GoType
			if cell.FmtInfo.Spec != 0 {
				goType = config.SpecifierToGoType[cell.FmtInfo.Spec]
			}

			err = processArg(scope, cell.Name, goType)
			if err != nil {
				return err
			}
		case VarInfo:
			idx := variableScopeIndex(scope.Variables, cell.Name)
			if idx == -1 {
				return NewFieldError(ErrCouldNotProcess, cell.Name, ErrVariableNotSpecified)
			}
		}
	}

	return nil
}

func processArg(scope *MessageScope, arg string, goType GoType) (err error) {
	otherIdx := argumentIndex(scope.Arguments, arg)

	if otherIdx == -1 {
		scope.Arguments = append(scope.Arguments, Argument{
			Name:   arg,
			GoType: goType,
		})

		return nil
	}

	if goType.IsZero() {
		return nil
	}

	other := &scope.Arguments[otherIdx]

	if other.GoType.IsZero() {
		other.GoType = goType
		return nil
	}

	if other.GoType != goType {
		return NewFieldError(ErrCouldNotProcess, arg, ErrTypesDontMatch)
	}

	return nil
}

func checkFielsXor(fields []FieldValue) (err error) {
	var specified bool

	for _, field := range fields {
		if field.Value.IsZero() {
			continue
		}

		if !specified {
			specified = true
			continue
		}

		return ErrFieldsSpecifiedAtTheSameTime
	}

	if !specified {
		return ErrFieldsNotSpecified
	}

	return nil
}

func getFieldNames(fields []FieldValue) (str string) {
	var b strings.Builder

	b.WriteByte('[')

	for i, field := range fields {
		if i != 0 {
			b.WriteByte(',')
		}
		b.WriteString(field.Name)
	}

	b.WriteByte(']')

	return b.String()
}

func processFields(scope *MessageScope, fields []FieldValue) (err error) {
	for _, field := range fields {
		if field.Value.IsZero() {
			continue
		}

		err = field.Value.Process(scope)
		if err != nil {
			return NewFieldError(ErrCouldNotProcess, field.Name, err)
		}
	}

	return nil
}

func argumentIndex(arguments []Argument, name string) (idx int) {
	for i := 0; i < len(arguments); i++ {
		if arguments[i].Name == name {
			return i
		}
	}

	return -1
}

func variableScopeIndex(variables []VariableScope, name string) (idx int) {
	for i := 0; i < len(variables); i++ {
		if variables[i].Name == name {
			return i
		}
	}

	return -1
}
