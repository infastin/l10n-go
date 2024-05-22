package parse

import (
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/infastin/go-l10n/ast"
	"github.com/infastin/go-l10n/common"
)

func parseFormat(fmt string) (parts ast.FormatParts, err error) {
	pos := 0

	for fmt != "" {
		idx, err := findBlockStart(fmt, &pos)
		if err != nil {
			return nil, err
		}

		if idx == -1 {
			parts = append(parts, ast.Text(fmt))
			break
		}

		// Preserve '$' or '&' character
		text := fmt[:idx+1]
		fmt = fmt[idx:]
		cur := rune(fmt[0])

		if len(fmt) == 1 {
			return nil, common.NewError(common.ErrUnexpectedEndOfFormat,
				common.ErrorPosition(pos),
				common.ErrorExpectedAnyChar(cur, '{'),
			)
		}

		next := rune(fmt[1])
		pos++

		if cur != next && next != '{' {
			return nil, common.NewError(common.ErrUnexpectedChar,
				common.ErrorValueChar(next),
				common.ErrorPosition(pos),
				common.ErrorExpectedAnyChar(cur, '{'),
			)
		}

		fmt = fmt[2:]
		pos++

		// If encountered '$$' or '&&' write text with '$' or '&'
		if cur == next {
			if text != "" {
				parts = append(parts, ast.Text(text))
			}
			continue
		}

		// If encountered '${' or '&{' write text without '$' and '&'
		if text := text[:idx]; text != "" {
			parts = append(parts, ast.Text(text))
		}

		idx, err = findClosingBracket(fmt, &pos)
		if err != nil {
			return nil, err
		}

		switch cur {
		case '$':
			arg, addPos, err := parseArgument(fmt[:idx])
			if err != nil {
				err.(*common.Error).Pos += common.ErrorPosition(pos)
				return nil, err
			}

			parts = append(parts, arg)
			pos += addPos
			fmt = fmt[idx+1:]
		case '&':
			variable, addPos, err := parseVariable(fmt[:idx])
			if err != nil {
				err.(*common.Error).Pos += common.ErrorPosition(pos)
				return nil, err
			}

			parts = append(parts, variable)
			pos += addPos
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
			return 0, common.NewError(common.ErrInvalidChar,
				common.ErrorValueChar(r),
				common.ErrorPosition(*pos),
			)
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
			return 0, common.NewError(common.ErrInvalidChar,
				common.ErrorValueChar(r),
				common.ErrorPosition(*pos),
			)
		}

		if r == '}' {
			idx = i
			break
		}

		*pos++
		i += n
	}

	if idx == -1 {
		return 0, common.NewError(common.ErrNoClosingBracket, common.ErrorPosition(*pos))
	}

	return idx, nil
}

func parseVariable(variable string) (info ast.VarInfo, pos int, err error) {
	switch err = checkVariableName(variable); err {
	case common.ErrInvalidVariableName:
		return ast.VarInfo{}, 0, common.NewError(err,
			common.ErrorValueStr(variable),
			common.ErrorPosition(0),
		)
	case common.ErrNoVariableName:
		return ast.VarInfo{}, 0, common.NewError(err, common.ErrorPosition(0))
	}

	info.Name = variable

	return info, len(variable), nil
}

func checkVariableName(variable string) (err error) {
	if variable == "" {
		return common.ErrNoVariableName
	}

	for i := 0; i < len(variable); i++ {
		if c := variable[i]; (c < 'a' || c > 'z') && (c < 'A' || c > 'Z') && c != '_' {
			return common.ErrInvalidVariableName
		}
	}

	return nil
}

func parseArgument(arg string) (info ast.ArgInfo, pos int, err error) {
	colonIdx := strings.IndexByte(arg, ':')
	if colonIdx != -1 {
		formatInfo, addPos, err := parseArgumentFormat(arg[:colonIdx])
		if err != nil {
			return ast.ArgInfo{}, 0, err
		}

		info.FmtInfo = formatInfo
		pos = addPos
		arg = arg[colonIdx+1:]
	}

	switch err = checkArgumentName(arg); err {
	case common.ErrInvalidArgumentName:
		return ast.ArgInfo{}, 0, common.NewError(err,
			common.ErrorValueStr(arg),
			common.ErrorPosition(pos),
		)
	case common.ErrNoArgumentName:
		return ast.ArgInfo{}, 0, common.NewError(err, common.ErrorPosition(pos))
	}

	info.Name = arg

	return info, pos, nil
}

func checkArgumentName(arg string) (err error) {
	if arg == "" {
		return common.ErrNoArgumentName
	}

	for i := 0; i < len(arg); i++ {
		if c := arg[i]; (c < 'a' || c > 'z') && (c < 'A' || c > 'Z') {
			return common.ErrInvalidArgumentName
		}
	}

	return nil
}

func parseArgumentFormat(fmt string) (info ast.FmtInfo, pos int, err error) {
	// Parse flags
	fmt, pos, err = parseArgumentFormatFlags(fmt, pos, &info)
	if err != nil {
		return ast.FmtInfo{}, 0, err
	}

	// Parse width
	if fmt != "" && fmt[0] >= '0' && fmt[0] <= '9' {
		fmt, pos, err = parseArgumentFormatWidth(fmt, pos, &info)
		if err != nil {
			return ast.FmtInfo{}, 0, err
		}
	}

	// Parse precision
	if fmt != "" && fmt[0] == '.' {
		fmt, pos, err = parseArgumentFormatPrecision(fmt, pos, &info)
		if err != nil {
			return ast.FmtInfo{}, 0, err
		}
	}

	// Parse specifier
	if fmt != "" {
		fmt, pos, err = parseArgumentFormatSpecifier(fmt, pos, &info)
		if err != nil {
			return ast.FmtInfo{}, 0, err
		}
	}

	// Parse modifier
	if fmt != "" {
		fmt, pos, err = parseArgumentFormatModifier(fmt, pos, &info)
		if err != nil {
			return ast.FmtInfo{}, 0, err
		}
	}

	if fmt != "" {
		return ast.FmtInfo{}, 0, common.NewError(common.ErrUnexpectedText, common.ErrorPosition(pos))
	}

	return info, pos, nil
}

func parseArgumentFormatFlags(fmt string, pos int, info *ast.FmtInfo) (newFmt string, newPos int, err error) {
	for fmt != "" {
		r, n := utf8.DecodeRuneInString(fmt)
		if r == utf8.RuneError {
			return "", 0, common.NewError(common.ErrInvalidChar,
				common.ErrorValueChar(r),
				common.ErrorPosition(pos),
			)
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

func parseArgumentFormatWidth(fmt string, pos int, info *ast.FmtInfo) (newFmt string, newPos int, err error) {
	lastNumber := 1
	for i := 1; ; i++ {
		if i == len(fmt) || fmt[i] < '0' || fmt[i] > '9' {
			lastNumber = i
			break
		}
	}

	width, err := strconv.ParseInt(fmt[:lastNumber], 10, 64)
	if err != nil {
		return "", 0, common.NewError(common.ErrInvalidWidth,
			common.ErrorValueStr(fmt[:lastNumber]),
			common.ErrorPosition(pos),
			common.ErrorWrapped(err),
		)
	}

	info.Width = ast.WidthOpt{Value: int(width), Valid: true}
	fmt = fmt[lastNumber:]
	pos += lastNumber

	return fmt, pos, nil
}

func parseArgumentFormatPrecision(fmt string, pos int, info *ast.FmtInfo) (newFmt string, newPos int, err error) {
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
			return "", 0, common.NewError(common.ErrInvalidPrecision,
				common.ErrorValueStr(fmt[:lastNumber]),
				common.ErrorPosition(pos),
				common.ErrorWrapped(err),
			)
		}

		prec = int(prec64)
	}

	info.Prec = ast.PrecOpt{Value: prec, Valid: true}
	fmt = fmt[lastNumber:]
	pos += lastNumber

	return fmt, pos, nil
}

func parseArgumentFormatSpecifier(fmt string, pos int, info *ast.FmtInfo) (newFmt string, newPos int, err error) {
	r, n := utf8.DecodeRuneInString(fmt)
	if r == utf8.RuneError {
		return "", 0, common.NewError(common.ErrInvalidChar,
			common.ErrorValueChar(r),
			common.ErrorPosition(pos),
		)
	}

	if !slices.Contains(common.Config.FormatSpecifiers, r) {
		return "", 0, common.NewError(common.ErrInvalidSpecifier,
			common.ErrorValueChar(r),
			common.ErrorPosition(pos),
		)
	}

	info.Spec = r
	fmt = fmt[n:]
	pos++

	return fmt, pos, nil
}

func parseArgumentFormatModifier(fmt string, pos int, info *ast.FmtInfo) (newFmt string, newPos int, err error) {
	r, n := utf8.DecodeRuneInString(fmt)
	if r == utf8.RuneError {
		return "", 0, common.NewError(common.ErrInvalidChar,
			common.ErrorValueChar(r),
			common.ErrorPosition(pos),
		)
	}

	info.Mod = ast.ModOpt{Value: r, Valid: true}
	fmt = fmt[n:]
	pos++

	return fmt, pos, nil
}
