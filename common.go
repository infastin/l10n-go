package main

import (
	"slices"
	"strconv"
	"strings"

	"golang.org/x/text/language"
)

type GoType struct {
	Import  string
	Package string
	Type    string
}

func (t *GoType) IsZero() bool {
	return t.Import == "" &&
		t.Package == "" &&
		t.Type == ""
}

func DefaultSpecifiersToGoTypes() map[rune]GoType {
	return map[rune]GoType{
		's': {Type: "string"},
		'd': {Type: "int"},
		'f': {Type: "float64"},
		'S': {
			Import:  "fmt",
			Package: "fmt",
			Type:    "Stringer",
		},
		'F': {
			Import:  "fmt",
			Package: "fmt",
			Type:    "Format",
		},
		'M': {
			Import:  "encoding",
			Package: "encoding",
			Type:    "TextMarshaler",
		},
	}
}

type Plural struct {
	Arg   string
	Zero  FormatParts
	One   FormatParts
	Many  FormatParts
	Other FormatParts
}

func (p *Plural) IsZero() bool {
	return p.Arg == "" &&
		p.Zero == nil &&
		p.One == nil &&
		p.Many == nil &&
		p.Other == nil
}

func (p *Plural) GetArgumentNames() (args []string) {
	args = append(args, p.Arg)
	formatParts := []FormatParts{p.Zero, p.One, p.Many, p.Other}

	for _, parts := range formatParts {
		names := parts.GetArgumentNames()
		for _, name := range names {
			if !slices.Contains(args, name) {
				args = append(args, name)
			}
		}
	}

	return args
}

type Variable struct {
	Name   string
	Plural Plural
	String FormatParts
}

type Message struct {
	Name      string
	Variables []Variable
	Plural    Plural
	String    FormatParts
}

type Argument struct {
	Name   string
	GoType GoType
}

type VariableScope struct {
	Variable
	ArgumentNames []string
}

type MessageScope struct {
	Name      string
	Variables []VariableScope
	Plural    Plural
	String    FormatParts
	Arguments []Argument
}

func (s *MessageScope) IsSimple() bool {
	return len(s.Arguments) == 0 && len(s.Variables) == 0
}

type GoImport struct {
	Import  string
	Package string
}

type Localization struct {
	Name    string
	Lang    language.Tag
	Scopes  []MessageScope
	Imports []GoImport
}

func (loc *Localization) AddImport(imp GoImport) {
	if !slices.Contains(loc.Imports, imp) {
		loc.Imports = append(loc.Imports, imp)
	}
}

func localizationIndex(locs []Localization, lang language.Tag) (idx int) {
	for i := 0; i < len(locs); i++ {
		if locs[i].Lang.String() == lang.String() {
			return i
		}
	}

	return -1
}

type WidthOpt struct {
	Value int
	Valid bool
}

type PrecOpt struct {
	Value int
	Valid bool
}

type ModOpt struct {
	Value rune
	Valid bool
}

type FmtInfo struct {
	Spec   rune
	Width  WidthOpt
	Prec   PrecOpt
	Mod    ModOpt
	Flags  []rune
	length int
}

func (i *FmtInfo) IsZero() bool {
	return i.Spec == 0 &&
		!i.Width.Valid &&
		!i.Prec.Valid &&
		!i.Mod.Valid &&
		i.Flags == nil
}

func (i *FmtInfo) GoFormat(goType GoType) string {
	var spec rune

	if !i.Mod.Valid {
		switch goType.Type {
		case "string":
			spec = 's'
		case "int":
			spec = 'd'
		case "float64":
			spec = 'f'
		default:
			spec = 'v'
		}
	} else {
		spec = i.Mod.Value
	}

	var b strings.Builder

	b.WriteByte('%')

	for _, flag := range i.Flags {
		b.WriteRune(flag)
	}

	if i.Width.Valid {
		b.WriteString(strconv.Itoa(i.Width.Value))
	}

	if i.Prec.Valid {
		b.WriteByte('.')
		b.WriteString(strconv.Itoa(i.Prec.Value))
	}

	b.WriteRune(spec)

	return b.String()
}

type ArgInfo struct {
	Name    string
	FmtInfo FmtInfo
	length  int
}

type VarInfo struct {
	Name   string
	length int
}

type FormatParts []any

func (f FormatParts) IsZero() bool {
	return len(f) == 0
}

func (f FormatParts) GetArgumentNames() (args []string) {
	for _, part := range f {
		arg, ok := part.(*ArgInfo)
		if ok && !slices.Contains(args, arg.Name) {
			args = append(args, arg.Name)
		}
	}

	return args
}
