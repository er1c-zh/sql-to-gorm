package main

import (
	"bytes"
	"flag"
	"fmt"
	"strings"

	res "github.com/antlr/antlr4/doc/resources"
	"github.com/antlr/antlr4/runtime/Go/antlr"
	gen "github.com/er1c-zh/sql-to-gorm/antlr4_gen"
)

var (
	path string
)

func main() {
	flag.StringVar(&path, "file", "", "path to sql file")
	flag.Parse()

	input, err := antlr.NewFileStream(path)
	if err != nil {
		fmt.Printf("NewFileStream fail: %s", err.Error())
		flag.Usage()
		return
	}
	//input := antlr.NewInputStream(src)
	lexer := gen.NewMySqlLexer(res.NewCaseChangingStream(input, true))
	stream := antlr.NewCommonTokenStream(lexer, 0)
	p := gen.NewMySqlParser(stream)
	// p.AddErrorListener(antlr.NewDiagnosticErrorListener(true))
	p.BuildParseTrees = true
	tree := p.Root()

	l := &Listener{
		GoModelFile: GoModelFile{
			Import: map[string]interface{}{},
		},
	}
	antlr.ParseTreeWalkerDefault.Walk(l, tree)

	fmt.Printf("%s", l.ToGorm())
}

// GoModelFile model file
type GoModelFile struct {
	TableList []*Table
	Import    map[string]interface{}
	Package   string
}

func (f GoModelFile) ToGorm() string {
	buf := new(bytes.Buffer)
	pkg := f.Package
	if pkg == "" {
		pkg = "models"
	}
	buf.WriteString(fmt.Sprintf("package %s\n", pkg))

	if len(f.Import) > 0 {
		buf.WriteString("import (\n")
		for _import := range f.Import {
			buf.WriteString(fmt.Sprintf("    \"%s\"\n", _import))
		}
		buf.WriteString(")\n")
	}

	for _, t := range f.TableList {
		buf.WriteString(t.ToGorm())
		buf.WriteString("\n")
	}

	return buf.String()
}

type Table struct {
	Name string
	Cols []*Col
}

func (t Table) ToGorm() string {
	buf := new(bytes.Buffer)
	buf.WriteByte('\n')
	buf.WriteString("type ")
	buf.WriteString("T")
	buf.WriteString(strings.Trim(t.Name, "`"))
	buf.WriteString(" struct {")
	buf.WriteByte('\n')
	cols := make([]string, 0, len(t.Cols))
	for _, col := range t.Cols {
		cols = append(cols, col.ToGorm())
	}
	buf.WriteString(strings.Join(cols, "\n"))
	buf.WriteByte('\n')
	buf.WriteString("}")
	buf.WriteByte('\n')
	return buf.String()
}

type Col struct {
	Name     string
	DataType string
	NotNull  bool
	Default  string
}

func (c Col) ToGorm() string {
	return fmt.Sprintf("    %s %s \"gorm:%s\"",
		strings.Trim(c.Name, "`"), c.DataType, c.Name)
}

type Listener struct {
	*gen.BaseMySqlParserListener
	CurrentTable *Table
	CurrentCol   *Col

	GoModelFile
}

func (l *Listener) EnterColumnCreateTable(ctx *gen.ColumnCreateTableContext) {
	if l.CurrentTable != nil {
		panic("last table not done")
	}
	l.CurrentTable = &Table{Name: ctx.TableName().GetText()}
}
func (l *Listener) ExitColumnCreateTable(ctx *gen.ColumnCreateTableContext) {
	l.TableList = append(l.TableList, l.CurrentTable)
	l.CurrentTable = nil
}

func (l *Listener) EnterColumnDeclaration(ctx *gen.ColumnDeclarationContext) {
	if l.CurrentCol != nil {
		panic("last col not done")
	}
	l.CurrentCol = &Col{
		Name:    ctx.Uid().GetText(),
		NotNull: false,
		Default: "",
	}
}
func (l *Listener) ExitColumnDeclaration(ctx *gen.ColumnDeclarationContext) {
	if l.CurrentTable == nil {
		panic("col done but no table")
	}
	l.CurrentTable.Cols = append(l.CurrentTable.Cols, l.CurrentCol)
	l.CurrentCol = nil
}

/////////////////////////////////////////////
// string ///////////////////////////////////
/////////////////////////////////////////////
func (l *Listener) EnterStringDataType(c *gen.StringDataTypeContext) {
	l.ParseDataType(c.GetTypeName().GetText(), []Rule{
		{contain: "", _type: "string"},
	})
}
func (l *Listener) EnterNationalStringDataType(c *gen.NationalStringDataTypeContext) {
	l.ParseDataType(c.GetTypeName().GetText(), []Rule{
		{contain: "", _type: "string"},
	})
}
func (l *Listener) EnterNationalVaryingStringDataType(c *gen.NationalVaryingStringDataTypeContext) {
	l.ParseDataType(c.GetTypeName().GetText(), []Rule{
		{contain: "", _type: "string"},
	})
}

/////////////////////////////////////////////
// Dimension ////////////////////////////////
/////////////////////////////////////////////

func (l *Listener) EnterDimensionDataType(c *gen.DimensionDataTypeContext) {
	l.ParseDataType(c.GetTypeName().GetText(), []Rule{
		{contain: "int", _type: "int64"},
		{contain: "timestamp", _type: "int64"},
		{contain: "datetime", _type: "time.Time{}", repo: []string{"time"}},
		{contain: "year", _type: "time.Time{}", repo: []string{"time"}},
		{contain: "", _type: "float64"},
	})
}

/////////////////////////////////////////////
// simple data type//////////////////////////
/////////////////////////////////////////////

func (l *Listener) EnterSimpleDataType(c *gen.SimpleDataTypeContext) {
	l.ParseDataType(c.GetTypeName().GetText(), []Rule{
		{contain: "date", _type: "time.Time{}", repo: []string{"time"}},
		{contain: "bool", _type: "bool"},
		{contain: "serial", _type: "int64"},
		{contain: "", _type: "string"},
	})
}

/////////////////////////////////////////////
// collection data type//////////////////////
/////////////////////////////////////////////

func (l *Listener) EnterCollectionDataType(c *gen.CollectionDataTypeContext) {
	l.ParseDataType(c.GetTypeName().GetText(), []Rule{
		{contain: "", _type: "string"},
	})
}

/////////////////////////////////////////////
// spatial data type/////////////////////////
/////////////////////////////////////////////
func (l *Listener) EnterSpatialDataType(c *gen.SpatialDataTypeContext) {
	l.ParseDataType(c.GetTypeName().GetText(), []Rule{
		{contain: "", _type: "string"},
	})
}

/////////////////////////////////////////////
// long varchar data type////////////////////
/////////////////////////////////////////////

func (l *Listener) EnterLongVarcharDataType(c *gen.LongVarcharDataTypeContext) {
	l.ParseDataType(c.GetTypeName().GetText(), []Rule{
		{contain: "", _type: "string"},
	})
}

/////////////////////////////////////////////
// long varbinary data type//////////////////
/////////////////////////////////////////////
func (l *Listener) EnterLongVarbinaryDataType(c *gen.LongVarbinaryDataTypeContext) {
	l.ParseDataType(c.GetText(), []Rule{
		{contain: "", _type: "string"},
	})
}

/////////////////////////////////////////////
/////////////////////////////////////////////
/////////////////////////////////////////////

func (l *Listener) SetDataType(_t string) {
	if l.CurrentCol == nil {
		fmt.Printf("[WARN] get data type but no col: %s", _t)
		return
	}
	l.CurrentCol.DataType = _t
}

type Rule struct {
	contain string
	_type   string
	repo    []string
}

func (l *Listener) ParseDataType(_t string, rule []Rule) {
	typeName := strings.ToLower(_t)
	f := func(src string, contain string, _type string, repo []string) bool {
		if strings.Contains(src, contain) {
			l.SetDataType(_type)
			if len(repo) > 0 {
				for _, _import := range repo {
					l.Import[_import] = struct{}{}
				}
			}
			return true
		}
		return false
	}
	for _, item := range rule {
		if f(typeName, item.contain, item._type, item.repo) {
			break
		}
	}
}

const (
	src = `create table dt_table;`
)
