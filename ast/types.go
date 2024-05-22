package ast

import (
	"slices"
	"strconv"
	"strings"
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

type Value interface {
	value()
	IsZero() bool
	GetArgumentNames() (names []string)
}

type Plural struct {
	Arg   string
	Zero  FormatParts
	One   FormatParts
	Many  FormatParts
	Other FormatParts
}

func (Plural) value() {}

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

func (p *Plural) IsSimple() bool {
	formatParts := []FormatParts{p.Zero, p.One, p.Many, p.Other}

	for _, parts := range formatParts {
		if !parts.IsSimple() {
			return false
		}
	}

	return true
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

type GoImport struct {
	Import  string
	Package string
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
	Spec  rune
	Width WidthOpt
	Prec  PrecOpt
	Mod   ModOpt
	Flags []rune
}

func (i *FmtInfo) HasOptions() bool {
	return i.Width.Valid ||
		i.Prec.Valid ||
		i.Mod.Valid ||
		i.Flags != nil
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

type FormatPart interface {
	formatPart()
}

type ArgInfo struct {
	Name    string
	FmtInfo FmtInfo
}

type VarInfo struct {
	Name string
}

type Text string

func (ArgInfo) formatPart() {}
func (VarInfo) formatPart() {}
func (Text) formatPart()    {}

type FormatParts []FormatPart

func (FormatParts) value() {}

func (f FormatParts) IsZero() bool {
	return len(f) == 0
}

func (f FormatParts) GetArgumentNames() (args []string) {
	for _, part := range f {
		arg, ok := part.(ArgInfo)
		if ok && !slices.Contains(args, arg.Name) {
			args = append(args, arg.Name)
		}
	}
	return args
}

func (f FormatParts) IsSimple() bool {
	for _, part := range f {
		if _, ok := part.(Text); !ok {
			return false
		}
	}

	return true
}
