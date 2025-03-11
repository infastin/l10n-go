package codegen

import (
	goast "go/ast"
	gotoken "go/token"
	"strconv"
	"strings"

	"github.com/infastin/l10n-go/ast"
	"github.com/infastin/l10n-go/common"
	"github.com/infastin/l10n-go/scope"
)

func GenerateLocalizations(locs []scope.Localization) (files []*goast.File) {
	files = append(files, generateGeneral(locs))

	for i := 0; i < len(locs); i++ {
		files = append(files, generateMessages(&locs[i]))
	}

	return files
}

func generateGeneral(locs []scope.Localization) (file *goast.File) {
	file = &goast.File{
		Doc: &goast.CommentGroup{
			List: []*goast.Comment{
				{Text: "// Code generated by l10n-go; DO NOT EDIT."},
				{Text: ""},
			},
		},
		Name:  goast.NewIdent(common.Config.PackageName),
		Decls: []goast.Decl{},
	}

	if len(common.Config.Imports) != 0 {
		importDecl := &goast.GenDecl{
			Tok: gotoken.IMPORT,
		}

		for _, imp := range common.Config.Imports {
			importDecl.Specs = append(importDecl.Specs, &goast.ImportSpec{
				Path: &goast.BasicLit{
					Kind:  gotoken.STRING,
					Value: strconv.Quote(imp.Import),
				},
			})
		}

		file.Decls = append(file.Decls, importDecl)
	}

	file.Decls = append(file.Decls, &goast.GenDecl{
		Tok: gotoken.TYPE,
		Specs: []goast.Spec{
			&goast.TypeSpec{
				Name: goast.NewIdent("Localizer"),
				Type: generateGeneralInterface(locs[0].Scopes),
			},
		},
	})

	generateGeneralTable(locs, &file.Decls)
	generateGeneralSupported(locs, &file.Decls)
	generateGeneralFuncs(locs, &file.Decls)

	return file
}

func generateGeneralInterface(msgs []scope.MessageScope) (ifaceType *goast.InterfaceType) {
	ifaceType = &goast.InterfaceType{
		Methods: &goast.FieldList{},
	}

	for i := 0; i < len(msgs); i++ {
		msg := &msgs[i]
		funcType := &goast.FuncType{
			Params: &goast.FieldList{},
			Results: &goast.FieldList{
				List: []*goast.Field{
					{Type: goast.NewIdent("string")},
				},
			},
		}

		for i := 0; i < len(msg.Arguments); i++ {
			funcType.Params.List = append(funcType.Params.List, &goast.Field{
				Names: []*goast.Ident{goast.NewIdent(msg.Arguments[i].Name)},
				Type:  getPackageFieldType(&msg.Arguments[i]),
			})
		}

		ifaceType.Methods.List = append(ifaceType.Methods.List, &goast.Field{
			Names: []*goast.Ident{goast.NewIdent(msg.Name)},
			Type:  funcType,
		})
	}

	return ifaceType
}

func generateGeneralSupported(locs []scope.Localization, decls *[]goast.Decl) {
	sliceLit := &goast.CompositeLit{
		Type: &goast.ArrayType{
			Elt: goast.NewIdent("string"),
		},
	}

	supportedSpec := &goast.ValueSpec{
		Names:  []*goast.Ident{goast.NewIdent("Supported")},
		Values: []goast.Expr{sliceLit},
	}

	varDecl := &goast.GenDecl{
		Tok:   gotoken.VAR,
		Specs: []goast.Spec{supportedSpec},
	}

	for i := 0; i < len(locs); i++ {
		sliceLit.Elts = append(sliceLit.Elts, &goast.BasicLit{
			Kind:  gotoken.STRING,
			Value: strconv.Quote(locs[i].Lang.String()),
		})
	}

	*decls = append(*decls, varDecl)
}

func generateGeneralTable(locs []scope.Localization, decls *[]goast.Decl) {
	mapLit := &goast.CompositeLit{
		Type: &goast.MapType{
			Key:   goast.NewIdent("string"),
			Value: goast.NewIdent("Localizer"),
		},
	}

	tableSpec := &goast.ValueSpec{
		Names:  []*goast.Ident{goast.NewIdent("mapLangToLocalizer")},
		Values: []goast.Expr{mapLit},
	}

	varDecl := &goast.GenDecl{
		Tok:   gotoken.VAR,
		Specs: []goast.Spec{tableSpec},
	}

	for i := 0; i < len(locs); i++ {
		mapLit.Elts = append(mapLit.Elts, &goast.KeyValueExpr{
			Key: goast.NewIdent(strconv.Quote(locs[i].Lang.String())),
			Value: &goast.CompositeLit{
				Type: goast.NewIdent(getLocalizerTypeName(&locs[i])),
			},
		})
	}

	*decls = append(*decls, varDecl)
}

func generateGeneralFuncs(locs []scope.Localization, decls *[]goast.Decl) {
	generateGeneralFuncNew(locs, decls)
	generateGeneralFuncLang(locs, decls)
}

func generateGeneralFuncNew(_ []scope.Localization, decls *[]goast.Decl) {
	*decls = append(*decls, &goast.FuncDecl{
		Name: goast.NewIdent("New"),
		Type: &goast.FuncType{
			Params: &goast.FieldList{
				List: []*goast.Field{
					{
						Names: []*goast.Ident{goast.NewIdent("lang")},
						Type:  goast.NewIdent("string"),
					},
				},
			},
			Results: &goast.FieldList{
				List: []*goast.Field{
					{
						Names: []*goast.Ident{goast.NewIdent("loc")},
						Type:  goast.NewIdent("Localizer"),
					},
					{
						Names: []*goast.Ident{goast.NewIdent("ok")},
						Type:  goast.NewIdent("bool"),
					},
				},
			},
		},
		Body: &goast.BlockStmt{
			List: []goast.Stmt{
				&goast.AssignStmt{
					Lhs: []goast.Expr{goast.NewIdent("loc"), goast.NewIdent("ok")},
					Tok: gotoken.ASSIGN,
					Rhs: []goast.Expr{
						&goast.IndexExpr{
							X:     goast.NewIdent("mapLangToLocalizer"),
							Index: goast.NewIdent("lang"),
						},
					},
				},
				&goast.ReturnStmt{
					Results: []goast.Expr{
						goast.NewIdent("loc"),
						goast.NewIdent("ok"),
					},
				},
			},
		},
	})
}

func generateGeneralFuncLang(locs []scope.Localization, decls *[]goast.Decl) {
	switchStmt := &goast.TypeSwitchStmt{
		Assign: &goast.ExprStmt{
			X: &goast.TypeAssertExpr{
				X: goast.NewIdent("loc"),
			},
		},
		Body: &goast.BlockStmt{},
	}

	funcDecl := &goast.FuncDecl{
		Name: goast.NewIdent("Language"),
		Type: &goast.FuncType{
			Params: &goast.FieldList{
				List: []*goast.Field{
					{
						Names: []*goast.Ident{goast.NewIdent("loc")},
						Type:  goast.NewIdent("Localizer"),
					},
				},
			},
			Results: &goast.FieldList{
				List: []*goast.Field{
					{Type: goast.NewIdent("string")},
				},
			},
		},
		Body: &goast.BlockStmt{
			List: []goast.Stmt{switchStmt},
		},
	}

	for i := 0; i < len(locs); i++ {
		switchStmt.Body.List = append(switchStmt.Body.List, &goast.CaseClause{
			List: []goast.Expr{
				goast.NewIdent(getLocalizerTypeName(&locs[i])),
			},
			Body: []goast.Stmt{
				&goast.ReturnStmt{
					Results: []goast.Expr{
						&goast.BasicLit{
							Kind:  gotoken.STRING,
							Value: strconv.Quote(locs[i].Lang.String()),
						},
					},
				},
			},
		})
	}

	switchStmt.Body.List = append(switchStmt.Body.List, &goast.CaseClause{
		Body: []goast.Stmt{
			&goast.ReturnStmt{
				Results: []goast.Expr{
					&goast.BasicLit{
						Kind:  gotoken.STRING,
						Value: `""`,
					},
				},
			},
		},
	})

	*decls = append(*decls, funcDecl)
}

func generateMessages(loc *scope.Localization) (file *goast.File) {
	file = &goast.File{
		Name:  goast.NewIdent(common.Config.PackageName),
		Decls: []goast.Decl{},
		Doc: &goast.CommentGroup{
			List: []*goast.Comment{
				{Text: "// Code generated by l10n-go; DO NOT EDIT."},
				{Text: ""},
			},
		},
	}

	var decls []goast.Decl

	for i := 0; i < len(loc.Scopes); i++ {
		ms := &loc.Scopes[i]
		if ms.IsSimple() {
			generateSimpleMessage(loc, ms, &decls)
		} else {
			generateMessage(loc, ms, &decls)
		}
	}

	generateMessagesImportDecl(loc, &file.Decls)
	generateMessagesTypeDecl(loc, &file.Decls)

	file.Decls = append(file.Decls, decls...)

	return file
}

func generateMessagesImportDecl(loc *scope.Localization, decls *[]goast.Decl) {
	importDecl := &goast.GenDecl{
		Tok:   gotoken.IMPORT,
		Specs: []goast.Spec{},
	}

	for _, imp := range loc.Imports {
		importDecl.Specs = append(importDecl.Specs, &goast.ImportSpec{
			Path: &goast.BasicLit{
				Kind:  gotoken.STRING,
				Value: strconv.Quote(imp.Import),
			},
		})
	}

	if len(importDecl.Specs) != 0 {
		*decls = append(*decls, importDecl)
	}
}

func generateMessagesTypeDecl(loc *scope.Localization, decls *[]goast.Decl) {
	typeDecl := &goast.GenDecl{
		Tok: gotoken.TYPE,
		Specs: []goast.Spec{
			&goast.TypeSpec{
				Name: goast.NewIdent(getLocalizerTypeName(loc)),
				Type: &goast.StructType{
					Fields: &goast.FieldList{},
				},
			},
		},
	}

	*decls = append(*decls, typeDecl)
}

func generateMessage(loc *scope.Localization, ms *scope.MessageScope, decls *[]goast.Decl) {
	const builderName = "b0"

	loc.AddImport(ast.GoImport{Import: "strings", Package: "strings"})

	funcDecl := &goast.FuncDecl{
		Name: goast.NewIdent(getMessageFuncName(ms)),
		Recv: &goast.FieldList{
			List: []*goast.Field{
				{
					Names: []*goast.Ident{goast.NewIdent(getLocalizerName(loc))},
					Type:  goast.NewIdent(getLocalizerTypeName(loc)),
				},
			},
		},
		Type: &goast.FuncType{
			Params: &goast.FieldList{},
			Results: &goast.FieldList{
				List: []*goast.Field{
					{Type: goast.NewIdent("string")},
				},
			},
		},
		Body: &goast.BlockStmt{
			List: []goast.Stmt{
				&goast.AssignStmt{
					Lhs: []goast.Expr{
						goast.NewIdent(builderName),
					},
					Tok: gotoken.DEFINE,
					Rhs: []goast.Expr{
						&goast.CallExpr{
							Fun: goast.NewIdent("new"),
							Args: []goast.Expr{
								&goast.SelectorExpr{
									X:   goast.NewIdent("strings"),
									Sel: goast.NewIdent("Builder"),
								},
							},
						},
					},
				},
			},
		},
	}

	for i := 0; i < len(ms.Arguments); i++ {
		funcDecl.Type.Params.List = append(funcDecl.Type.Params.List, &goast.Field{
			Names: []*goast.Ident{goast.NewIdent(ms.Arguments[i].Name)},
			Type:  getPackageFieldType(&ms.Arguments[i]),
		})
	}

	values := []ast.Value{&ms.Plural, ms.String}

	for _, val := range values {
		if !val.IsZero() {
			generateValue(loc, ms, val, builderName, &funcDecl.Body.List)
			break
		}
	}

	funcDecl.Body.List = append(funcDecl.Body.List, &goast.ReturnStmt{
		Results: []goast.Expr{
			&goast.CallExpr{
				Fun: &goast.SelectorExpr{
					X:   goast.NewIdent(builderName),
					Sel: goast.NewIdent("String"),
				},
			},
		},
	})

	for i := 0; i < len(ms.Variables); i++ {
		generateVariableFunc(loc, ms, &ms.Variables[i], decls)
	}

	*decls = append(*decls, funcDecl)
}

func generateSimpleMessage(loc *scope.Localization, ms *scope.MessageScope, decls *[]goast.Decl) {
	funcDecl := &goast.FuncDecl{
		Name: goast.NewIdent(getMessageFuncName(ms)),
		Recv: &goast.FieldList{
			List: []*goast.Field{
				{
					Names: []*goast.Ident{goast.NewIdent(getLocalizerName(loc))},
					Type:  goast.NewIdent(getLocalizerTypeName(loc)),
				},
			},
		},
		Type: &goast.FuncType{
			Params: &goast.FieldList{},
			Results: &goast.FieldList{
				List: []*goast.Field{
					{Type: goast.NewIdent("string")},
				},
			},
		},
		Body: &goast.BlockStmt{
			List: []goast.Stmt{},
		},
	}

	for i := 0; i < len(ms.Arguments); i++ {
		funcDecl.Type.Params.List = append(funcDecl.Type.Params.List, &goast.Field{
			Names: []*goast.Ident{goast.NewIdent(ms.Arguments[i].Name)},
			Type:  getPackageFieldType(&ms.Arguments[i]),
		})
	}

	values := []ast.Value{&ms.Plural, ms.String}

	for _, val := range values {
		if !val.IsZero() {
			generateValue(loc, ms, val, "", &funcDecl.Body.List)
			break
		}
	}

	for i := 0; i < len(ms.Variables); i++ {
		generateVariableFunc(loc, ms, &ms.Variables[i], decls)
	}

	*decls = append(*decls, funcDecl)
}

func generatePlural(
	loc *scope.Localization,
	ms *scope.MessageScope,
	plural *ast.Plural,
	builderName string,
	list *[]goast.Stmt,
) {
	values := []struct {
		Value  ast.Value
		Op     gotoken.Token
		Number string
	}{
		{plural.Zero, gotoken.EQL, "0"},
		{plural.One, gotoken.EQL, "1"},
		{plural.Many, gotoken.GTR, "1"},
		{plural.Other, gotoken.ILLEGAL, ""},
	}

	switchStmt := &goast.SwitchStmt{
		Body: &goast.BlockStmt{},
	}

	for _, value := range values {
		if value.Value.IsZero() {
			continue
		}

		caseClause := &goast.CaseClause{}

		if value.Op != gotoken.ILLEGAL {
			caseClause.List = []goast.Expr{
				&goast.BinaryExpr{
					X:  goast.NewIdent(plural.Arg),
					Op: value.Op,
					Y: &goast.BasicLit{
						Kind:  gotoken.INT,
						Value: value.Number,
					},
				},
			}
		}

		generateValue(loc, ms, value.Value, builderName, &caseClause.Body)
		switchStmt.Body.List = append(switchStmt.Body.List, caseClause)
	}

	*list = append(*list, switchStmt)
}

func generateFormatParts(
	loc *scope.Localization,
	ms *scope.MessageScope,
	parts ast.FormatParts,
	builderName string,
	list *[]goast.Stmt,
) {
	if builderName == "" {
		generateSimpleFormatParts(loc, ms, parts, list)
		return
	}

	for _, part := range parts {
		switch part := part.(type) {
		case ast.Text:
			generateText(loc, ms, part, builderName, list)
		case ast.ArgInfo:
			idx := scope.ArgumentIndex(ms.Arguments, part.Name)
			generateArgument(loc, ms, &ms.Arguments[idx], &part, builderName, list)
		case ast.VarInfo:
			idx := scope.VariableScopeIndex(ms.Variables, part.Name)
			generateVariableCall(loc, ms, &ms.Variables[idx], builderName, list)
		}
	}
}

func generateSimpleFormatParts(
	_ *scope.Localization,
	_ *scope.MessageScope,
	parts ast.FormatParts,
	list *[]goast.Stmt,
) {
	var b strings.Builder

	for _, part := range parts {
		if part, ok := part.(ast.Text); ok {
			b.WriteString(string(part))
		}
	}

	*list = append(*list, &goast.ReturnStmt{
		Results: []goast.Expr{
			&goast.BasicLit{
				Kind:  gotoken.STRING,
				Value: strconv.Quote(b.String()),
			},
		},
	})
}

func generateText(
	_ *scope.Localization,
	_ *scope.MessageScope,
	text ast.Text,
	builderName string,
	list *[]goast.Stmt,
) {
	*list = append(*list, &goast.ExprStmt{
		X: &goast.CallExpr{
			Fun: &goast.SelectorExpr{
				X:   goast.NewIdent(builderName),
				Sel: goast.NewIdent("WriteString"),
			},
			Args: []goast.Expr{
				&goast.BasicLit{
					Kind:  gotoken.STRING,
					Value: strconv.Quote(string(text)),
				},
			},
		},
	})
}

func generateArgument(
	loc *scope.Localization,
	_ *scope.MessageScope,
	arg *scope.Argument,
	info *ast.ArgInfo,
	builderName string,
	list *[]goast.Stmt,
) {
	if info.FmtInfo.HasOptions() {
		generateArgumentFormat(loc, arg, info, builderName, list)
		return
	}

	if arg.GoType.Type == "any" {
		generateArgumentAny(loc, arg, builderName, list)
		return
	}

	callExpr := &goast.CallExpr{
		Fun: &goast.SelectorExpr{
			X:   goast.NewIdent(builderName),
			Sel: goast.NewIdent("WriteString"),
		},
	}

	*list = append(*list, &goast.ExprStmt{
		X: callExpr,
	})

	switch arg.GoType.Type {
	case "string":
		callExpr.Args = []goast.Expr{goast.NewIdent(arg.Name)}
	case "int":
		generateArgumentItoa(loc, arg, callExpr)
	case "float64":
		generateArgumentFormatFloat(loc, arg, callExpr)
	case "Stringer":
		generateArgumentStringer(loc, arg, callExpr)
	}
}

func generateArgumentStringer(_ *scope.Localization, arg *scope.Argument, callExpr *goast.CallExpr) {
	callExpr.Args = []goast.Expr{
		&goast.CallExpr{
			Fun: &goast.SelectorExpr{
				X:   goast.NewIdent(arg.Name),
				Sel: goast.NewIdent("String"),
			},
		},
	}
}

func generateArgumentFormat(
	loc *scope.Localization,
	arg *scope.Argument,
	info *ast.ArgInfo,
	builderName string,
	list *[]goast.Stmt,
) {
	fmtStr := info.FmtInfo.GoFormat(arg.GoType)
	loc.AddImport(ast.GoImport{Import: "fmt", Package: "fmt"})

	callExpr := &goast.CallExpr{
		Fun: &goast.SelectorExpr{
			X:   goast.NewIdent("fmt"),
			Sel: goast.NewIdent("Fprintf"),
		},
		Args: []goast.Expr{
			goast.NewIdent(builderName),
			&goast.BasicLit{
				Kind:  gotoken.STRING,
				Value: strconv.Quote(fmtStr),
			},
			goast.NewIdent(arg.Name),
		},
	}

	*list = append(*list, &goast.ExprStmt{
		X: callExpr,
	})
}

func generateArgumentItoa(loc *scope.Localization, arg *scope.Argument, callExpr *goast.CallExpr) {
	loc.AddImport(ast.GoImport{Import: "strconv", Package: "strconv"})
	callExpr.Args = []goast.Expr{
		&goast.CallExpr{
			Fun: &goast.SelectorExpr{
				X:   goast.NewIdent("strconv"),
				Sel: goast.NewIdent("Itoa"),
			},
			Args: []goast.Expr{goast.NewIdent(arg.Name)},
		},
	}
}

func generateArgumentFormatFloat(loc *scope.Localization, arg *scope.Argument, callExpr *goast.CallExpr) {
	loc.AddImport(ast.GoImport{Import: "strconv", Package: "strconv"})
	callExpr.Args = []goast.Expr{
		&goast.CallExpr{
			Fun: &goast.SelectorExpr{
				X:   goast.NewIdent("strconv"),
				Sel: goast.NewIdent("FormatFloat"),
			},
			Args: []goast.Expr{
				goast.NewIdent(arg.Name),
				&goast.BasicLit{
					Kind:  gotoken.CHAR,
					Value: `'f'`,
				},
				&goast.BasicLit{
					Kind:  gotoken.INT,
					Value: `6`,
				},
				&goast.BasicLit{
					Kind:  gotoken.INT,
					Value: `64`,
				},
			},
		},
	}
}

func generateArgumentAny(
	loc *scope.Localization,
	arg *scope.Argument,
	builderName string,
	list *[]goast.Stmt,
) {
	loc.AddImport(ast.GoImport{Import: "fmt", Package: "fmt"})

	callExpr := &goast.CallExpr{
		Fun: &goast.SelectorExpr{
			X:   goast.NewIdent("fmt"),
			Sel: goast.NewIdent("Fprint"),
		},
		Args: []goast.Expr{
			goast.NewIdent(builderName),
			goast.NewIdent(arg.Name),
		},
	}

	*list = append(*list, &goast.ExprStmt{
		X: callExpr,
	})
}

func generateVariableCall(
	loc *scope.Localization,
	ms *scope.MessageScope,
	variable *scope.VariableScope,
	builderName string,
	list *[]goast.Stmt,
) {
	callExpr := &goast.CallExpr{
		Fun: &goast.SelectorExpr{
			X:   goast.NewIdent(getLocalizerName(loc)),
			Sel: goast.NewIdent(getVariableFuncName(ms, variable)),
		},
		Args: []goast.Expr{
			goast.NewIdent(builderName),
		},
	}

	for _, name := range variable.ArgumentNames {
		callExpr.Args = append(callExpr.Args, goast.NewIdent(name))
	}

	*list = append(*list, &goast.ExprStmt{
		X: callExpr,
	})
}

func generateVariableFunc(
	loc *scope.Localization,
	ms *scope.MessageScope,
	variable *scope.VariableScope,
	decls *[]goast.Decl,
) {
	const builderName = "b0"

	builderField := &goast.Field{
		Names: []*goast.Ident{
			goast.NewIdent(builderName),
		},
		Type: &goast.StarExpr{
			X: &goast.SelectorExpr{
				X:   goast.NewIdent("strings"),
				Sel: goast.NewIdent("Builder"),
			},
		},
	}

	funcDecl := &goast.FuncDecl{
		Name: goast.NewIdent(getVariableFuncName(ms, variable)),
		Recv: &goast.FieldList{
			List: []*goast.Field{
				{
					Names: []*goast.Ident{goast.NewIdent(getLocalizerName(loc))},
					Type:  goast.NewIdent(getLocalizerTypeName(loc)),
				},
			},
		},
		Type: &goast.FuncType{
			Params: &goast.FieldList{
				List: []*goast.Field{builderField},
			},
		},
		Body: &goast.BlockStmt{},
	}

	values := []ast.Value{&variable.Plural, variable.String}

	for _, value := range values {
		if value.IsZero() {
			continue
		}

		for _, name := range variable.ArgumentNames {
			argIdx := scope.ArgumentIndex(ms.Arguments, name)
			funcDecl.Type.Params.List = append(funcDecl.Type.Params.List, &goast.Field{
				Names: []*goast.Ident{goast.NewIdent(name)},
				Type:  getPackageFieldType(&ms.Arguments[argIdx]),
			})
		}

		generateValue(loc, ms, value, builderName, &funcDecl.Body.List)
	}

	*decls = append(*decls, funcDecl)
}

func getPackageFieldType(arg *scope.Argument) goast.Expr {
	if arg.GoType.Package == "" {
		return goast.NewIdent(arg.GoType.Type)
	}

	return &goast.SelectorExpr{
		X:   goast.NewIdent(arg.GoType.Package),
		Sel: goast.NewIdent(arg.GoType.Type),
	}
}

func getLocalizerName(loc *scope.Localization) string {
	return loc.Lang.String() + "_l"
}

func getLocalizerTypeName(loc *scope.Localization) string {
	return loc.Lang.String() + "_Localizer"
}

func getMessageFuncName(ms *scope.MessageScope) string {
	return ms.Name
}

func getVariableFuncName(ms *scope.MessageScope, variable *scope.VariableScope) string {
	return ms.Name + "_" + variable.Name
}

func generateValue(
	loc *scope.Localization,
	ms *scope.MessageScope,
	value ast.Value,
	builderName string,
	list *[]goast.Stmt,
) {
	switch v := value.(type) {
	case *ast.Plural:
		generatePlural(loc, ms, v, builderName, list)
	case ast.FormatParts:
		generateFormatParts(loc, ms, v, builderName, list)
	}
}
