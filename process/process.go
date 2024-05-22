package process

import (
	"strings"

	"github.com/infastin/go-l10n/ast"
	"github.com/infastin/go-l10n/common"
	"github.com/infastin/go-l10n/scope"
)

type FieldValue struct {
	Name  string
	Value ast.Value
}

func ProcessMessages(msgs []ast.Message) (mss []scope.MessageScope, err error) {
	for i := 0; i < len(msgs); i++ {
		ms, err := processMessage(&msgs[i])
		if err != nil {
			return nil, common.NewFieldError(common.ErrCouldNotProcess, msgs[i].Name, err)
		}

		mss = append(mss, ms)
	}

	return mss, nil
}

func processMessage(msg *ast.Message) (ms scope.MessageScope, err error) {
	ms = scope.MessageScope{
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
		return scope.MessageScope{}, common.NewFieldError(common.ErrCouldNotProcess, getFieldNames(fields), err)
	}

	for i := 0; i < len(msg.Variables); i++ {
		var argNames []string
		values := []ast.Value{&msg.Variables[i].Plural, msg.Variables[i].String}

		for _, val := range values {
			if !val.IsZero() {
				argNames = val.GetArgumentNames()
				break
			}
		}

		ms.Variables = append(ms.Variables, scope.VariableScope{
			Variable:      msg.Variables[i],
			ArgumentNames: argNames,
		})
	}

	for i := 0; i < len(ms.Variables); i++ {
		err = processVariable(&ms, &ms.Variables[i])
		if err != nil {
			return scope.MessageScope{}, common.NewFieldError(common.ErrCouldNotProcess, ms.Variables[i].Name, err)
		}
	}

	err = processFields(&ms, fields)
	if err != nil {
		return scope.MessageScope{}, err
	}

	for i := 0; i < len(ms.Arguments); i++ {
		arg := &ms.Arguments[i]
		if arg.GoType.IsZero() {
			arg.GoType = common.Config.SpecifierToGoType['s']
		}
	}

	return ms, nil
}

func processVariable(ms *scope.MessageScope, variable *scope.VariableScope) (err error) {
	fields := []FieldValue{
		{"plural", &variable.Plural},
		{"string", variable.String},
	}

	err = checkFielsXor(fields)
	if err != nil {
		return common.NewFieldError(common.ErrCouldNotProcess, getFieldNames(fields), err)
	}

	err = processFields(ms, fields)
	if err != nil {
		return err
	}

	return nil
}

func processPlural(ms *scope.MessageScope, plural *ast.Plural) (err error) {
	if plural.Arg == "" {
		return common.NewFieldError(common.ErrCouldNotProcess, "arg", common.ErrFieldNotSpecified)
	}

	goType := common.Config.SpecifierToGoType['d']

	err = processArg(ms, plural.Arg, goType)
	if err != nil {
		return common.NewFieldError(common.ErrCouldNotProcess, "arg", err)
	}

	fields := []struct {
		Name        string
		FormatParts ast.FormatParts
	}{
		{"zero", plural.Zero},
		{"one", plural.One},
		{"many", plural.Many},
		{"other", plural.Other},
	}

	for _, field := range fields {
		err = processFormatParts(ms, field.FormatParts)
		if err != nil {
			return common.NewFieldError(common.ErrCouldNotProcess, field.Name, err)
		}
	}

	return nil
}

func processFormatParts(ms *scope.MessageScope, parts ast.FormatParts) (err error) {
	for _, cell := range parts {
		switch cell := cell.(type) {
		case ast.ArgInfo:
			var goType ast.GoType
			if cell.FmtInfo.Spec != 0 {
				goType = common.Config.SpecifierToGoType[cell.FmtInfo.Spec]
			}

			err = processArg(ms, cell.Name, goType)
			if err != nil {
				return err
			}
		case ast.VarInfo:
			idx := scope.VariableScopeIndex(ms.Variables, cell.Name)
			if idx == -1 {
				return common.NewFieldError(common.ErrCouldNotProcess, cell.Name, common.ErrVariableNotSpecified)
			}
		}
	}

	return nil
}

func processArg(ms *scope.MessageScope, arg string, goType ast.GoType) (err error) {
	otherIdx := scope.ArgumentIndex(ms.Arguments, arg)

	if otherIdx == -1 {
		ms.Arguments = append(ms.Arguments, scope.Argument{
			Name:   arg,
			GoType: goType,
		})

		return nil
	}

	if goType.IsZero() {
		return nil
	}

	other := &ms.Arguments[otherIdx]

	if other.GoType.IsZero() {
		other.GoType = goType
		return nil
	}

	if other.GoType != goType {
		return common.NewFieldError(common.ErrCouldNotProcess, arg, common.ErrTypesDontMatch)
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

		return common.ErrFieldsSpecifiedAtTheSameTime
	}

	if !specified {
		return common.ErrFieldsNotSpecified
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

func processFields(ms *scope.MessageScope, fields []FieldValue) (err error) {
	for _, field := range fields {
		if field.Value.IsZero() {
			continue
		}

		switch v := field.Value.(type) {
		case *ast.Plural:
			err = processPlural(ms, v)
		case ast.FormatParts:
			err = processFormatParts(ms, v)
		}

		if err != nil {
			return common.NewFieldError(common.ErrCouldNotProcess, field.Name, err)
		}
	}

	return nil
}
