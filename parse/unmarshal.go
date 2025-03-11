package parse

import (
	"slices"
	"strings"

	"github.com/infastin/l10n-go/ast"
	"github.com/infastin/l10n-go/common"
)

func UnmarshalMessages(in []byte, unmarshaler func(in []byte, out any) (err error),
) (messages []ast.Message, err error) {
	msgs := make(map[string]any)

	err = unmarshaler(in, &msgs)
	if err != nil {
		return nil, err
	}

	for name, msg := range msgs {
		if str, ok := msg.(string); ok {
			format, err := parseFormat(str)
			if err != nil {
				return nil, common.NewFieldError(common.ErrCouldNotUnmarshal, name, err)
			}

			messages = append(messages, ast.Message{
				Name:   name,
				String: format,
			})

			continue
		}

		table, ok := msg.(map[string]any)
		if !ok {
			err = common.NewError(common.ErrInvalidFieldType, common.ErrorExpectedAnyStr("string", "table"))
			return nil, common.NewFieldError(common.ErrCouldNotUnmarshal, name, err)
		}

		message, err := mapMessage(table)
		if err != nil {
			return nil, common.NewFieldError(common.ErrCouldNotUnmarshal, name, err)
		}

		slices.SortStableFunc(message.Variables, func(a, b ast.Variable) int {
			return strings.Compare(a.Name, b.Name)
		})

		message.Name = name
		messages = append(messages, message)
	}

	slices.SortStableFunc(messages, func(a, b ast.Message) int {
		return strings.Compare(a.Name, b.Name)
	})

	return messages, nil
}

func mapMessage(table map[string]any) (message ast.Message, err error) {
	for k, v := range table {
		switch k {
		case "variables":
			v, ok := v.(map[string]any)
			if !ok {
				err = common.NewError(common.ErrInvalidFieldType, common.ErrorExpectedStr("table"))
				return ast.Message{}, common.NewFieldError(common.ErrCouldNotUnmarshal, k, err)
			}

			message.Variables, err = mapVariables(v)
			if err != nil {
				return ast.Message{}, common.NewFieldError(common.ErrCouldNotUnmarshal, k, err)
			}
		case "plural":
			v, ok := v.(map[string]any)
			if !ok {
				err = common.NewError(common.ErrInvalidFieldType, common.ErrorExpectedStr("table"))
				return ast.Message{}, common.NewFieldError(common.ErrCouldNotUnmarshal, k, err)
			}

			message.Plural, err = mapPlural(v)
			if err != nil {
				return ast.Message{}, common.NewFieldError(common.ErrCouldNotUnmarshal, k, err)
			}
		case "string":
			v, ok := v.(string)
			if !ok {
				err = common.NewError(common.ErrInvalidFieldType, common.ErrorExpectedStr("string"))
				return ast.Message{}, common.NewFieldError(common.ErrCouldNotUnmarshal, k, err)
			}

			format, err := parseFormat(v)
			if err != nil {
				return ast.Message{}, common.NewFieldError(common.ErrCouldNotUnmarshal, k, err)
			}

			message.String = format
		default:
			err = common.NewError(common.ErrUnknownField, common.ErrorExpectedAnyStr("variables", "plural", "string"))
			return ast.Message{}, common.NewFieldError(common.ErrCouldNotUnmarshal, k, err)
		}
	}

	return message, nil
}

func mapVariables(table map[string]any) (variables []ast.Variable, err error) {
	for k, v := range table {
		err = checkVariableName(k)
		if err != nil {
			return nil, common.NewFieldError(common.ErrCouldNotUnmarshal, k, err)
		}

		if str, ok := v.(string); ok {
			format, err := parseFormat(str)
			if err != nil {
				return nil, common.NewFieldError(common.ErrCouldNotUnmarshal, k, err)
			}

			variables = append(variables, ast.Variable{
				Name:   k,
				String: format,
			})

			continue
		}

		v, ok := v.(map[string]any)
		if !ok {
			err = common.NewError(common.ErrInvalidFieldType, common.ErrorExpectedAnyStr("string", "table"))
			return nil, common.NewFieldError(common.ErrCouldNotUnmarshal, k, err)
		}

		variable, err := mapVariable(v)
		if err != nil {
			return nil, common.NewFieldError(common.ErrCouldNotUnmarshal, k, err)
		}

		variable.Name = k
		variables = append(variables, variable)
	}

	return variables, nil
}

func mapVariable(table map[string]any) (variable ast.Variable, err error) {
	for k, v := range table {
		switch k {
		case "plural":
			v, ok := v.(map[string]any)
			if !ok {
				err = common.NewError(common.ErrInvalidFieldType, common.ErrorExpectedStr("table"))
				return ast.Variable{}, common.NewFieldError(common.ErrCouldNotUnmarshal, k, err)
			}

			variable.Plural, err = mapPlural(v)
			if err != nil {
				return ast.Variable{}, common.NewFieldError(common.ErrCouldNotUnmarshal, k, err)
			}
		case "string":
			v, ok := v.(string)
			if !ok {
				err = common.NewError(common.ErrInvalidFieldType, common.ErrorExpectedStr("string"))
				return ast.Variable{}, common.NewFieldError(common.ErrCouldNotUnmarshal, k, err)
			}

			format, err := parseFormat(v)
			if err != nil {
				return ast.Variable{}, common.NewFieldError(common.ErrCouldNotUnmarshal, k, err)
			}

			variable.String = format
		default:
			err = common.NewError(common.ErrUnknownField, common.ErrorExpectedAnyStr("plural"))
			return ast.Variable{}, common.NewFieldError(common.ErrCouldNotUnmarshal, k, err)
		}
	}

	return variable, nil
}

func mapPlural(table map[string]any) (plural ast.Plural, err error) {
	for k, v := range table {
		v, ok := v.(string)
		if !ok {
			err = common.NewError(common.ErrInvalidFieldType, common.ErrorExpectedStr("string"))
			return ast.Plural{}, common.NewFieldError(common.ErrCouldNotUnmarshal, k, err)
		}

		if k == "arg" {
			err = checkArgumentName(v)
			if err != nil {
				return ast.Plural{}, common.NewFieldError(common.ErrCouldNotUnmarshal, k, err)
			}

			plural.Arg = v
			continue
		}

		format, err := parseFormat(v)
		if err != nil {
			return ast.Plural{}, common.NewFieldError(common.ErrCouldNotUnmarshal, k, err)
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
			err = common.NewError(common.ErrUnknownField, common.ErrorExpectedAnyStr("arg", "zero", "one", "many", "other"))
			return ast.Plural{}, common.NewFieldError(common.ErrCouldNotUnmarshal, k, err)
		}
	}

	return plural, nil
}
