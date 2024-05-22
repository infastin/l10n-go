package scope

import (
	"slices"

	"github.com/infastin/go-l10n/ast"
	"golang.org/x/text/language"
)

type Argument struct {
	Name   string
	GoType ast.GoType
}

func ArgumentIndex(arguments []Argument, name string) (idx int) {
	for i := 0; i < len(arguments); i++ {
		if arguments[i].Name == name {
			return i
		}
	}
	return -1
}

type VariableScope struct {
	ast.Variable
	ArgumentNames []string
}

func VariableScopeIndex(variables []VariableScope, name string) (idx int) {
	for i := 0; i < len(variables); i++ {
		if variables[i].Name == name {
			return i
		}
	}
	return -1
}

type MessageScope struct {
	Name      string
	Variables []VariableScope
	Plural    ast.Plural
	String    ast.FormatParts
	Arguments []Argument
}

func (m *MessageScope) IsSimple() bool {
	if !m.Plural.IsZero() {
		return m.Plural.IsSimple()
	}
	return m.String.IsSimple()
}

type Localization struct {
	Name    string
	Lang    language.Tag
	Scopes  []MessageScope
	Imports []ast.GoImport
}

func (loc *Localization) AddImport(imp ast.GoImport) {
	if !slices.Contains(loc.Imports, imp) {
		loc.Imports = append(loc.Imports, imp)
	}
}

func LocalizationIndex(locs []Localization, lang language.Tag) (idx int) {
	for i := 0; i < len(locs); i++ {
		if locs[i].Lang.String() == lang.String() {
			return i
		}
	}

	return -1
}
