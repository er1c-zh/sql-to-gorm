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
	path     string
	_package string
)

func Init() {
	flag.StringVar(&path, "file", "", "path to sql file")
	flag.StringVar(&_package, "package", "models", "go file package")
	flag.Parse()
}

func main() {
	Init()

	input, err := antlr.NewFileStream(path)
	if err != nil {
		fmt.Printf("NewFileStream fail: %s", err.Error())
		flag.Usage()
		return
	}

	lexer := gen.NewMySqlLexer(res.NewCaseChangingStream(input, true))
	stream := antlr.NewCommonTokenStream(lexer, 0)
	p := gen.NewMySqlParser(stream)
	// p.AddErrorListener(antlr.NewDiagnosticErrorListener(true))
	p.BuildParseTrees = true

	option := DefaultOption()
	option.Package = _package
	ln := NewListener(option)
	antlr.ParseTreeWalkerDefault.Walk(ln, p.Root())

	fmt.Printf("%s", ln.ToGorm())
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
	buf.WriteString(fmt.Sprintf("type %s struct {\n", t.Name))
	cols := make([]string, 0, len(t.Cols))
	for _, col := range t.Cols {
		cols = append(cols, col.ToGorm())
	}
	buf.WriteString(strings.Join(cols, "\n"))
	buf.WriteString("\n}\n")
	return buf.String()
}

type Col struct {
	Name     string
	DataType string
	NotNull  bool
	Default  string
	Comment  string
}

func (c Col) ToGorm() string {
	comment := c.Name
	if c.Comment != "" {
		comment = c.Comment
	}
	tagList := make([]string, 0, 1)
	tagList = append(tagList, fmt.Sprintf("column:%s", c.Name))
	if c.Default != "" {
		tagList = append(tagList, fmt.Sprintf("default:%s", c.Default))
	}

	return fmt.Sprintf("    %s %s `gorm:\"%s\"` //%s",
		c.Name, c.DataType, strings.Join(tagList, ";"), comment)
}

type Listener struct {
	*gen.BaseMySqlParserListener
	CurrentTable *Table
	CurrentCol   *Col

	GoModelFile
}

type Option struct {
	Package string
}

func DefaultOption() Option {
	return Option{
		Package: "models",
	}
}

func NewListener(option Option) *Listener {
	ln := &Listener{
		GoModelFile: GoModelFile{
			Import: map[string]interface{}{},
		},
	}
	ln.Package = option.Package
	return ln
}

func (l *Listener) EnterColumnCreateTable(ctx *gen.ColumnCreateTableContext) {
	if l.CurrentTable != nil {
		panic("last table not done")
	}

	tableNameList := strings.Split(
		strings.Trim(ctx.TableName().GetText(), "`"), ".")

	l.CurrentTable = &Table{
		Name: tableNameList[len(tableNameList)-1],
	}
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
		Name:    strings.Trim(ctx.Uid().GetText(), "`"),
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
// comment //////////////////////////////////
/////////////////////////////////////////////
func (l *Listener) EnterCommentColumnConstraint(c *gen.CommentColumnConstraintContext) {
	if l.CurrentCol == nil {
		return
	}
	l.CurrentCol.Comment = c.STRING_LITERAL().GetText()
}

/////////////////////////////////////////////
// default //////////////////////////////////
/////////////////////////////////////////////
func (l *Listener) EnterDefaultColumnConstraint(c *gen.DefaultColumnConstraintContext) {
	if l.CurrentCol == nil {
		return
	}
	l.CurrentCol.Default = c.DefaultValue().GetText()
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
