package main

import (
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"
)

func parseFormat(fmt string) (parts FormatParts, err error) {
	pos := 0

	for fmt != "" {
		idx, err := findBlockStart(fmt, &pos)
		if err != nil {
			return nil, err
		}

		if idx == -1 {
			parts = append(parts, fmt)
			break
		}

		// Preserve '$' or '&' character
		text := fmt[:idx+1]
		fmt = fmt[idx:]
		cur := rune(fmt[0])

		if len(fmt) == 1 {
			return nil, NewError(ErrUnexpectedEndOfFormat,
				ErrorPosition(pos),
				ErrorExpectedAnyChar(cur, '{'),
			)
		}

		next := rune(fmt[1])
		pos++

		if cur != next && next != '{' {
			return nil, NewError(ErrUnexpectedChar,
				ErrorValueChar(next),
				ErrorPosition(pos),
				ErrorExpectedAnyChar(cur, '{'),
			)
		}

		fmt = fmt[2:]
		pos++

		// If encountered '$$' or '&&' write text with '$' or '&'
		if cur == next {
			if text != "" {
				parts = append(parts, text)
			}
			continue
		}

		// If encountered '${' or '&{' write text without '$' and '&'
		if text := text[:idx]; text != "" {
			parts = append(parts, text)
		}

		idx, err = findClosingBracket(fmt, &pos)
		if err != nil {
			return nil, err
		}

		switch cur {
		case '$':
			arg, err := parseArgument(fmt[:idx])
			if err != nil {
				err.(*Error).Pos += ErrorPosition(pos)
				return nil, err
			}

			parts = append(parts, arg)
			pos += arg.length
			fmt = fmt[idx+1:]
		case '&':
			variable, err := parseVariable(fmt[:idx])
			if err != nil {
				err.(*Error).Pos += ErrorPosition(pos)
				return nil, err
			}

			parts = append(parts, variable)
			pos += variable.length
			fmt = fmt[idx+1:]
		}
	}

	return parts, nil
}

func findBlockStart(fmt string, pos *int) (idx int, err error) {
	idx = -1

	for i := 0; i < len(fmt); {
		r, n := utf8.DecodeRuneInString(fmt[i:])
		if r == utf8.RuneError {
			return 0, NewError(ErrInvalidChar, ErrorValueChar(r), ErrorPosition(*pos))
		}

		if r == '$' || r == '&' {
			idx = i
			break
		}

		*pos++
		i += n
	}

	return idx, nil
}

func findClosingBracket(fmt string, pos *int) (idx int, err error) {
	idx = -1

	for i := 0; i < len(fmt); {
		r, n := utf8.DecodeRuneInString(fmt[i:])
		if r == utf8.RuneError {
			return 0, NewError(ErrInvalidChar, ErrorValueChar(r), ErrorPosition(*pos))
		}

		if r == '}' {
			idx = i
			break
		}

		*pos++
		i += n
	}

	if idx == -1 {
		return 0, NewError(ErrNoClosingBracket, ErrorPosition(*pos))
	}

	return idx, nil
}

func parseVariable(variable string) (info VarInfo, err error) {
	switch err = checkVariableName(variable); err {
	case ErrInvalidVariableName:
		return VarInfo{}, NewError(err, ErrorValueStr(variable), ErrorPosition(0))
	case ErrNoVariableName:
		return VarInfo{}, NewError(err, ErrorPosition(0))
	}

	info.Name = variable
	info.length = len(variable)

	return info, nil
}

func checkVariableName(variable string) (err error) {
	if variable == "" {
		return ErrNoVariableName
	}

	for i := 0; i < len(variable); i++ {
		if c := variable[i]; (c < 'a' || c > 'z') && (c < 'A' || c > 'Z') && c != '_' {
			return ErrInvalidVariableName
		}
	}

	return nil
}

func parseArgument(arg string) (info ArgInfo, err error) {
	colonIdx := strings.IndexByte(arg, ':')
	if colonIdx != -1 {
		formatInfo, err := parseArgumentFormat(arg[:colonIdx])
		if err != nil {
			return ArgInfo{}, err
		}

		info.FmtInfo = formatInfo
		arg = arg[colonIdx+1:]
	}

	pos := 0
	if info.FmtInfo.length != 0 {
		pos = info.FmtInfo.length + 1
	}

	switch err = checkArgumentName(arg); err {
	case ErrInvalidArgumentName:
		return ArgInfo{}, NewError(err, ErrorValueStr(arg), ErrorPosition(pos))
	case ErrNoArgumentName:
		return ArgInfo{}, NewError(err, ErrorPosition(pos))
	}

	info.Name = arg
	info.length = pos

	return info, nil
}

func checkArgumentName(arg string) (err error) {
	if arg == "" {
		return ErrNoArgumentName
	}

	for i := 0; i < len(arg); i++ {
		if c := arg[i]; (c < 'a' || c > 'z') && (c < 'A' || c > 'Z') {
			return ErrInvalidArgumentName
		}
	}

	return nil
}

func parseArgumentFormat(fmt string) (info FmtInfo, err error) {
	pos := 0

	// Parse flags
	fmt, pos, err = parseArgumentFormatFlags(fmt, pos, &info)
	if err != nil {
		return FmtInfo{}, err
	}

	// Parse width
	if fmt != "" && fmt[0] >= '0' && fmt[0] <= '9' {
		fmt, pos, err = parseArgumentFormatWidth(fmt, pos, &info)
		if err != nil {
			return FmtInfo{}, err
		}
	}

	// Parse precision
	if fmt != "" && fmt[0] == '.' {
		fmt, pos, err = parseArgumentFormatPrecision(fmt, pos, &info)
		if err != nil {
			return FmtInfo{}, err
		}
	}

	// Parse specifier
	if fmt != "" {
		fmt, pos, err = parseArgumentFormatSpecifier(fmt, pos, &info)
		if err != nil {
			return FmtInfo{}, err
		}
	}

	// Parse modifier
	if fmt != "" {
		fmt, pos, err = parseArgumentFormatModifier(fmt, pos, &info)
		if err != nil {
			return FmtInfo{}, err
		}
	}

	if fmt != "" {
		return FmtInfo{}, NewError(ErrUnexpectedText, ErrorPosition(pos))
	}

	info.length = pos

	return info, nil
}

func parseArgumentFormatFlags(fmt string, pos int, info *FmtInfo) (newFmt string, newPos int, err error) {
	for fmt != "" {
		r, n := utf8.DecodeRuneInString(fmt)
		if r == utf8.RuneError {
			return "", 0, NewError(ErrInvalidChar, ErrorValueChar(r), ErrorPosition(pos))
		}

		switch r {
		case '+', '-', ' ', '0', '#':
			if !slices.Contains(info.Flags, r) {
				info.Flags = append(info.Flags, r)
			}
		default:
			return fmt, pos, nil
		}

		fmt = fmt[n:]
		pos++
	}

	return fmt, pos, nil
}

func parseArgumentFormatWidth(fmt string, pos int, info *FmtInfo) (newFmt string, newPos int, err error) {
	lastNumber := 1
	for i := 1; ; i++ {
		if i == len(fmt) || fmt[i] < '0' || fmt[i] > '9' {
			lastNumber = i
			break
		}
	}

	width, err := strconv.ParseInt(fmt[:lastNumber], 10, 64)
	if err != nil {
		return "", 0, NewError(ErrInvalidWidth,
			ErrorValueStr(fmt[:lastNumber]),
			ErrorPosition(pos),
			ErrorWrapped(err),
		)
	}

	info.Width = WidthOpt{int(width), true}
	fmt = fmt[lastNumber:]
	pos += lastNumber

	return fmt, pos, nil
}

func parseArgumentFormatPrecision(fmt string, pos int, info *FmtInfo) (newFmt string, newPos int, err error) {
	lastNumber := 1
	for i := 1; ; i++ {
		if i == len(fmt) || fmt[i] < '0' || fmt[i] > '9' {
			lastNumber = i
			break
		}
	}

	prec := 0
	if lastNumber != 1 {
		prec64, err := strconv.ParseInt(fmt[1:lastNumber], 10, 64)
		if err != nil {
			return "", 0, NewError(ErrInvalidPrecision,
				ErrorValueStr(fmt[:lastNumber]),
				ErrorPosition(pos),
				ErrorWrapped(err),
			)
		}

		prec = int(prec64)
	}

	info.Prec = PrecOpt{prec, true}
	fmt = fmt[lastNumber:]
	pos += lastNumber

	return fmt, pos, nil
}

func parseArgumentFormatSpecifier(fmt string, pos int, info *FmtInfo) (newFmt string, newPos int, err error) {
	r, n := utf8.DecodeRuneInString(fmt)
	if r == utf8.RuneError {
		return "", 0, NewError(ErrInvalidChar, ErrorValueChar(r), ErrorPosition(pos))
	}

	if !slices.Contains(config.FormatSpecifiers, r) {
		return "", 0, NewError(ErrInvalidSpecifier, ErrorValueChar(r), ErrorPosition(pos))
	}

	info.Spec = r
	fmt = fmt[n:]
	pos++

	return fmt, pos, nil
}

func parseArgumentFormatModifier(fmt string, pos int, info *FmtInfo) (newFmt string, newPos int, err error) {
	r, n := utf8.DecodeRuneInString(fmt)
	if r == utf8.RuneError {
		return "", 0, NewError(ErrInvalidChar, ErrorValueChar(r), ErrorPosition(pos))
	}

	info.Mod = ModOpt{r, true}
	fmt = fmt[n:]
	pos++

	return fmt, pos, nil
}
