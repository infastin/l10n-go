package printer

import (
	"bytes"
	"go/ast"
	"go/token"
	"io"
	"strings"
)

type astPrinter struct {
	b     *bytes.Buffer
	level int
}

func (p *astPrinter) indentLine() {
	p.b.WriteString(strings.Repeat("\t", p.level))
}

func (p *astPrinter) next() *astPrinter {
	return &astPrinter{
		b:     p.b,
		level: p.level + 1,
	}
}

func (p *astPrinter) writeFile(f *ast.File) {
	if f.Doc != nil {
		for _, comment := range f.Doc.List {
			p.b.WriteString(comment.Text)
			p.b.WriteByte('\n')
		}
	}

	p.b.WriteString("package ")
	p.writeIdent(f.Name)
	p.b.WriteByte('\n')
}

func (p *astPrinter) writeGenDecl(d *ast.GenDecl) {
	p.b.WriteString(d.Tok.String())
	p.b.WriteByte(' ')

	if len(d.Specs) == 1 {
		p.writeSpec(d.Specs[0])
		return
	}

	p.b.WriteString("(\n")

	next := p.next()
	for _, spec := range d.Specs {
		next.indentLine()
		next.writeSpec(spec)
		next.b.WriteByte('\n')
	}

	p.indentLine()
	p.b.WriteString(")")
}

func (p *astPrinter) writeFuncDecl(f *ast.FuncDecl) {
	p.b.WriteString("func")
	p.b.WriteByte(' ')

	if f.Recv != nil {
		p.b.WriteByte('(')
		p.writeField(f.Recv.List[0])
		p.b.WriteString(") ")
	}

	p.writeIdent(f.Name)
	p.writeFuncParams(f.Type.Params)

	if f.Type.Results != nil {
		p.writeFuncResults(f.Type.Results)
	}

	if f.Body != nil {
		p.b.WriteByte(' ')
		p.writeBlockStmt(f.Body)
	}
}

func (p *astPrinter) writeFuncParams(params *ast.FieldList) {
	p.b.WriteByte('(')

	for i, field := range params.List {
		if i != 0 {
			p.b.WriteString(", ")
		}
		p.writeField(field)
	}

	p.b.WriteString(") ")
}

func (p *astPrinter) writeFuncResults(r *ast.FieldList) {
	if len(r.List) == 1 && r.List[0].Names == nil {
		p.writeExpr(r.List[0].Type)
		return
	}

	p.b.WriteByte('(')

	for i, field := range r.List {
		if i != 0 {
			p.b.WriteString(", ")
		}
		p.writeField(field)
	}

	p.b.WriteByte(')')
}

func (p *astPrinter) writeSpec(s ast.Spec) {
	switch s := s.(type) {
	case *ast.ImportSpec:
		p.writeImportSpec(s)
	case *ast.TypeSpec:
		p.writeTypeSpec(s)
	case *ast.ValueSpec:
		p.writeValueSpec(s)
	}
}

func (p *astPrinter) writeImportSpec(s *ast.ImportSpec) {
	if s.Name != nil {
		p.writeIdent(s.Name)
		p.b.WriteByte(' ')
	}

	p.writeBasicLit(s.Path)
}

func (p *astPrinter) writeTypeSpec(s *ast.TypeSpec) {
	p.writeIdent(s.Name)
	p.b.WriteByte(' ')
	p.writeExpr(s.Type)
}

func (p *astPrinter) writeValueSpec(s *ast.ValueSpec) {
	for i, name := range s.Names {
		if i != 0 {
			p.b.WriteString(", ")
		}
		p.writeIdent(name)
	}

	if s.Type != nil {
		p.b.WriteByte(' ')
		p.writeExpr(s.Type)
	}

	if len(s.Values) == 0 {
		return
	}

	p.b.WriteString(" = ")

	for i, val := range s.Values {
		if i != 0 {
			p.b.WriteString(", ")
		}
		p.writeExpr(val)
	}
}

func (p *astPrinter) writeField(f *ast.Field) {
	for i, name := range f.Names {
		if i != 0 {
			p.b.WriteString(", ")
		}
		p.writeIdent(name)
	}

	if f.Type != nil {
		p.b.WriteByte(' ')
		p.writeExpr(f.Type)
	}
}

func (p *astPrinter) writeExpr(e ast.Expr) {
	switch e := e.(type) {
	case *ast.Ident:
		p.writeIdent(e)
	case *ast.SelectorExpr:
		p.writeSelectorExpr(e)
	case *ast.StarExpr:
		p.writeStarExpr(e)
	case *ast.StructType:
		p.writeStructType(e)
	case *ast.InterfaceType:
		p.writeInterfaceType(e)
	case *ast.FuncType:
		p.writeFuncType(e)
	case *ast.CallExpr:
		p.writeCallExpr(e)
	case *ast.BasicLit:
		p.writeBasicLit(e)
	case *ast.BinaryExpr:
		p.writeBinaryExpr(e)
	case *ast.UnaryExpr:
		p.writeUnaryExpr(e)
	case *ast.IndexExpr:
		p.writeIndexExpr(e)
	case *ast.CompositeLit:
		p.writeComposeLit(e)
	case *ast.KeyValueExpr:
		p.writeKeyValueExpr(e)
	case *ast.MapType:
		p.writeMapType(e)
	case *ast.ArrayType:
		p.writeArrayType(e)
	case *ast.TypeAssertExpr:
		p.writeTypeAssertExpr(e)
	}
}

func (p *astPrinter) writeIdent(i *ast.Ident) {
	p.b.WriteString(i.Name)
}

func (p *astPrinter) writeSelectorExpr(s *ast.SelectorExpr) {
	p.writeExpr(s.X)
	p.b.WriteByte('.')
	p.writeIdent(s.Sel)
}

func (p *astPrinter) writeBasicLit(l *ast.BasicLit) {
	p.b.WriteString(l.Value)
}

func (p *astPrinter) writeStarExpr(s *ast.StarExpr) {
	p.b.WriteByte('*')
	p.writeExpr(s.X)
}

func (p *astPrinter) writeStructType(s *ast.StructType) {
	p.b.WriteString("struct")

	if len(s.Fields.List) == 0 {
		p.b.WriteString("{}")
		return
	}

	p.b.WriteString(" {\n")

	next := p.next()
	for _, field := range s.Fields.List {
		next.indentLine()
		next.writeField(field)
		next.b.WriteByte('\n')
	}

	p.indentLine()
	p.b.WriteByte('}')
}

func (p *astPrinter) writeInterfaceType(i *ast.InterfaceType) {
	p.b.WriteString("interface")

	if len(i.Methods.List) == 0 {
		p.b.WriteString("{}")
		return
	}

	p.b.WriteString(" {\n")

	next := p.next()
	for _, field := range i.Methods.List {
		next.indentLine()
		next.writeField(field)
		next.b.WriteByte('\n')
	}

	p.indentLine()
	p.b.WriteByte('}')
}

func (p *astPrinter) writeFuncType(f *ast.FuncType) {
	if f.Func == token.NoPos {
		p.b.Truncate(p.b.Len() - 1)
	}

	p.writeFuncParams(f.Params)

	if f.Results != nil {
		p.writeFuncResults(f.Results)
	}
}

func (p *astPrinter) writeCallExpr(c *ast.CallExpr) {
	p.writeExpr(c.Fun)
	p.b.WriteByte('(')

	for i, expr := range c.Args {
		if i != 0 {
			p.b.WriteString(", ")
		}
		p.writeExpr(expr)
	}

	p.b.WriteByte(')')
}

func (p *astPrinter) writeBinaryExpr(b *ast.BinaryExpr) {
	p.writeExpr(b.X)
	p.b.WriteByte(' ')
	p.b.WriteString(b.Op.String())
	p.b.WriteByte(' ')
	p.writeExpr(b.Y)
}

func (p *astPrinter) writeUnaryExpr(u *ast.UnaryExpr) {
	p.b.WriteString(u.Op.String())
	p.writeExpr(u.X)
}

func (p *astPrinter) writeIndexExpr(i *ast.IndexExpr) {
	p.writeExpr(i.X)
	p.b.WriteByte('[')
	p.writeExpr(i.Index)
	p.b.WriteByte(']')
}

func (p *astPrinter) writeComposeLit(c *ast.CompositeLit) {
	if c.Type != nil {
		p.writeExpr(c.Type)
	}

	if len(c.Elts) == 0 {
		p.b.WriteString("{}")
		return
	}

	p.b.WriteString("{\n")

	next := p.next()
	for _, expr := range c.Elts {
		next.indentLine()
		next.writeExpr(expr)
		next.b.WriteString(",\n")
	}

	p.indentLine()
	p.b.WriteByte('}')
}

func (p *astPrinter) writeKeyValueExpr(kv *ast.KeyValueExpr) {
	p.writeExpr(kv.Key)
	p.b.WriteString(": ")
	p.writeExpr(kv.Value)
}

func (p *astPrinter) writeMapType(m *ast.MapType) {
	p.b.WriteString("map[")
	p.writeExpr(m.Key)
	p.b.WriteByte(']')
	p.writeExpr(m.Value)
}

func (p *astPrinter) writeArrayType(a *ast.ArrayType) {
	p.b.WriteByte('[')

	if a.Len != nil {
		p.writeExpr(a.Len)
	}

	p.b.WriteByte(']')
	p.writeExpr(a.Elt)
}

func (p *astPrinter) writeTypeAssertExpr(t *ast.TypeAssertExpr) {
	p.writeExpr(t.X)
	p.b.WriteString(".(")

	if t.Type != nil {
		p.writeExpr(t.Type)
	} else {
		p.b.WriteString("type")
	}

	p.b.WriteByte(')')
}

func (p *astPrinter) writeStmt(s ast.Stmt) {
	switch s := s.(type) {
	case *ast.ExprStmt:
		p.writeExpr(s.X)
	case *ast.DeclStmt:
		p.writeDeclStmt(s)
	case *ast.SwitchStmt:
		p.writeSwitchStmt(s)
	case *ast.TypeSwitchStmt:
		p.writeTypeSwitchStmt(s)
	case *ast.CaseClause:
		p.writeCaseClause(s)
	case *ast.AssignStmt:
		p.writeAssignStmt(s)
	case *ast.ReturnStmt:
		p.writeReturnStmt(s)
	}
}

func (p *astPrinter) writeDeclStmt(s *ast.DeclStmt) {
	genDecl, ok := s.Decl.(*ast.GenDecl)
	if !ok {
		return
	}

	p.writeGenDecl(genDecl)
}

func (p *astPrinter) writeSwitchStmt(s *ast.SwitchStmt) {
	p.b.WriteString("switch ")

	if s.Init != nil {
		p.writeStmt(s.Init)
		p.b.WriteString("; ")
	}

	if s.Tag != nil {
		p.writeExpr(s.Tag)
		p.b.WriteByte(' ')
	}

	p.b.WriteString("{\n")

	for _, stmt := range s.Body.List {
		p.indentLine()
		p.writeStmt(stmt)
		p.b.WriteByte('\n')
	}

	p.indentLine()
	p.b.WriteByte('}')
}

func (p *astPrinter) writeTypeSwitchStmt(t *ast.TypeSwitchStmt) {
	p.b.WriteString("switch ")

	if t.Init != nil {
		p.writeStmt(t.Init)
		p.b.WriteString("; ")
	}

	if t.Assign != nil {
		p.writeStmt(t.Assign)
		p.b.WriteByte(' ')
	}

	p.b.WriteString("{\n")

	for _, stmt := range t.Body.List {
		p.indentLine()
		p.writeStmt(stmt)
		p.b.WriteByte('\n')
	}

	p.indentLine()
	p.b.WriteByte('}')
}

func (p *astPrinter) writeCaseClause(c *ast.CaseClause) {
	if c.List != nil {
		p.b.WriteString("case ")

		for i, expr := range c.List {
			if i != 0 {
				p.b.WriteString(", ")
			}
			p.writeExpr(expr)
		}
	} else {
		p.b.WriteString("default")
	}

	p.b.WriteString(":\n")

	next := p.next()
	for i, stmt := range c.Body {
		if i != 0 {
			next.b.WriteByte('\n')
		}

		next.indentLine()
		next.writeStmt(stmt)
	}
}

func (p *astPrinter) writeAssignStmt(a *ast.AssignStmt) {
	for i, expr := range a.Lhs {
		if i != 0 {
			p.b.WriteString(", ")
		}
		p.writeExpr(expr)
	}

	p.b.WriteByte(' ')
	p.b.WriteString(a.Tok.String())
	p.b.WriteByte(' ')

	for i, expr := range a.Rhs {
		if i != 0 {
			p.b.WriteString(", ")
		}
		p.writeExpr(expr)
	}
}

func (p *astPrinter) writeReturnStmt(r *ast.ReturnStmt) {
	p.b.WriteString("return")

	if len(r.Results) == 0 {
		return
	}

	p.b.WriteByte(' ')

	for i, expr := range r.Results {
		if i != 0 {
			p.b.WriteString(", ")
		}
		p.writeExpr(expr)
	}
}

func (p *astPrinter) writeBlockStmt(b *ast.BlockStmt) {
	p.b.WriteString("{\n")

	next := p.next()
	prevAssign := false

	for i, stmt := range b.List {
		switch stmt.(type) {
		case *ast.ReturnStmt:
			if i != 0 && len(b.List) != 2 {
				next.b.WriteByte('\n')
			}
			prevAssign = false
		case *ast.AssignStmt, *ast.DeclStmt:
			prevAssign = true
		default:
			if prevAssign {
				next.b.WriteByte('\n')
			}
			prevAssign = false
		}

		next.indentLine()
		next.writeStmt(stmt)
		next.b.WriteByte('\n')
	}

	p.indentLine()
	p.b.WriteByte('}')
}

func FprintAstFile(w io.Writer, f *ast.File) (err error) {
	p := &astPrinter{
		b:     bytes.NewBuffer(nil),
		level: 0,
	}

	p.writeFile(f)

	for i, decl := range f.Decls {
		if i != 0 {
			p.b.WriteByte('\n')
		}

		p.b.WriteByte('\n')

		switch decl := decl.(type) {
		case *ast.GenDecl:
			p.writeGenDecl(decl)
		case *ast.FuncDecl:
			p.writeFuncDecl(decl)
		}
	}

	_, err = io.Copy(w, p.b)
	return err
}
