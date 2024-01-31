package main

import (
	"slices"
	"strings"
)

func unmarshalMessages(in []byte, unmarshaler func(in []byte, out any) (err error),
) (messages []Message, err error) {
	msgs := make(map[string]any)

	err = unmarshaler(in, &msgs)
	if err != nil {
		return nil, err
	}

	for name, msg := range msgs {
		if str, ok := msg.(string); ok {
			format, err := parseFormat(str)
			if err != nil {
				return nil, NewFieldError(ErrCouldNotUnmarshal, name, err)
			}

			messages = append(messages, Message{
				Name:   name,
				String: format,
			})

			continue
		}

		table, ok := msg.(map[string]any)
		if !ok {
			err = NewError(ErrInvalidFieldType, ErrorExpectedAnyStr("string", "table"))
			return nil, NewFieldError(ErrCouldNotUnmarshal, name, err)
		}

		message, err := mapMessage(table)
		if err != nil {
			return nil, NewFieldError(ErrCouldNotUnmarshal, name, err)
		}

		slices.SortFunc(message.Variables, func(a, b Variable) int {
			return strings.Compare(a.Name, b.Name)
		})

		message.Name = name
		messages = append(messages, message)
	}

	slices.SortFunc(messages, func(a, b Message) int {
		return strings.Compare(a.Name, b.Name)
	})

	return messages, nil
}

func mapMessage(table map[string]any) (message Message, err error) {
	for k, v := range table {
		switch k {
		case "variables":
			v, ok := v.(map[string]any)
			if !ok {
				err = NewError(ErrInvalidFieldType, ErrorExpectedStr("table"))
				return Message{}, NewFieldError(ErrCouldNotUnmarshal, k, err)
			}

			message.Variables, err = mapVariables(v)
			if err != nil {
				return Message{}, NewFieldError(ErrCouldNotUnmarshal, k, err)
			}
		case "plural":
			v, ok := v.(map[string]any)
			if !ok {
				err = NewError(ErrInvalidFieldType, ErrorExpectedStr("table"))
				return Message{}, NewFieldError(ErrCouldNotUnmarshal, k, err)
			}

			message.Plural, err = mapPlural(v)
			if err != nil {
				return Message{}, NewFieldError(ErrCouldNotUnmarshal, k, err)
			}
		case "string":
			v, ok := v.(string)
			if !ok {
				err = NewError(ErrInvalidFieldType, ErrorExpectedStr("string"))
				return Message{}, NewFieldError(ErrCouldNotUnmarshal, k, err)
			}

			format, err := parseFormat(v)
			if err != nil {
				return Message{}, NewFieldError(ErrCouldNotUnmarshal, k, err)
			}

			message.String = format
		default:
			err = NewError(ErrUnknownField, ErrorExpectedAnyStr("variables", "plural", "string"))
			return Message{}, NewFieldError(ErrCouldNotUnmarshal, k, err)
		}
	}

	return message, nil
}

func mapVariables(table map[string]any) (variables []Variable, err error) {
	for k, v := range table {
		err = checkVariableName(k)
		if err != nil {
			return nil, NewFieldError(ErrCouldNotUnmarshal, k, err)
		}

		if str, ok := v.(string); ok {
			format, err := parseFormat(str)
			if err != nil {
				return nil, NewFieldError(ErrCouldNotUnmarshal, k, err)
			}

			variables = append(variables, Variable{
				Name:   k,
				String: format,
			})

			continue
		}

		v, ok := v.(map[string]any)
		if !ok {
			err = NewError(ErrInvalidFieldType, ErrorExpectedAnyStr("string", "table"))
			return nil, NewFieldError(ErrCouldNotUnmarshal, k, err)
		}

		variable, err := mapVariable(v)
		if err != nil {
			return nil, NewFieldError(ErrCouldNotUnmarshal, k, err)
		}

		variable.Name = k
		variables = append(variables, variable)
	}

	return variables, nil
}

func mapVariable(table map[string]any) (variable Variable, err error) {
	for k, v := range table {
		switch k {
		case "plural":
			v, ok := v.(map[string]any)
			if !ok {
				err = NewError(ErrInvalidFieldType, ErrorExpectedStr("table"))
				return Variable{}, NewFieldError(ErrCouldNotUnmarshal, k, err)
			}

			variable.Plural, err = mapPlural(v)
			if err != nil {
				return Variable{}, NewFieldError(ErrCouldNotUnmarshal, k, err)
			}
		case "string":
			v, ok := v.(string)
			if !ok {
				err = NewError(ErrInvalidFieldType, ErrorExpectedStr("string"))
				return Variable{}, NewFieldError(ErrCouldNotUnmarshal, k, err)
			}

			format, err := parseFormat(v)
			if err != nil {
				return Variable{}, NewFieldError(ErrCouldNotUnmarshal, k, err)
			}

			variable.String = format
		default:
			err = NewError(ErrUnknownField, ErrorExpectedAnyStr("plural"))
			return Variable{}, NewFieldError(ErrCouldNotUnmarshal, k, err)
		}
	}

	return variable, nil
}

func mapPlural(table map[string]any) (plural Plural, err error) {
	for k, v := range table {
		v, ok := v.(string)
		if !ok {
			err = NewError(ErrInvalidFieldType, ErrorExpectedStr("string"))
			return Plural{}, NewFieldError(ErrCouldNotUnmarshal, k, err)
		}

		if k == "arg" {
			err = checkArgumentName(v)
			if err != nil {
				return Plural{}, NewFieldError(ErrCouldNotUnmarshal, k, err)
			}

			plural.Arg = v
			continue
		}

		format, err := parseFormat(v)
		if err != nil {
			return Plural{}, NewFieldError(ErrCouldNotUnmarshal, k, err)
		}

		switch k {
		case "zero":
			plural.Zero = format
		case "one":
			plural.One = format
		case "many":
			plural.Many = format
		case "other":
			plural.Other = format
		default:
			err = NewError(ErrUnknownField, ErrorExpectedAnyStr("arg", "zero", "one", "many", "other"))
			return Plural{}, NewFieldError(ErrCouldNotUnmarshal, k, err)
		}
	}

	return plural, nil
}
