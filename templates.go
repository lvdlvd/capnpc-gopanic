package main

import (
	"fmt"
	"strings"
	"text/template"
)

var templates = template.Must(template.New("").Funcs(template.FuncMap{
	"capnp":   g_imports.capnp,
	"math":    g_imports.math,
	"server":  g_imports.server,
	"context": g_imports.context,
	"strconv": g_imports.strconv,
	"title":   strings.Title,
	"hasDiscriminant": func(f field) bool {
		return f.DiscriminantValue() != Field_noDiscriminant
	},
	"discriminantOffset": func(n *node) uint32 {
		return n.StructGroup().DiscriminantOffset() * 2
	},
}).Parse(`
{{define "enum"}}{{with .Annotations.Doc}}// {{.}}
{{end}}type {{.Node.Name}} uint16

{{with .EnumValues}}
// Values of {{$.Node.Name}}.
const (
{{range .}}{{.FullName}} {{$.Node.Name}} = {{.Val}}
{{end}}
)

// String returns the enum's constant name.
func (c {{$.Node.Name}}) String() string {
	switch c {
	{{range .}}{{if .Tag}}case {{.FullName}}: return {{printf "%q" .Tag}}
	{{end}}{{end}}
	default: return ""
	}
}

// {{$.Node.Name}}FromString returns the enum value with a name,
// or the zero value if there's no such value.
func {{$.Node.Name}}FromString(c string) {{$.Node.Name}} {
	switch c {
	{{range .}}{{if .Tag}}case {{printf "%q" .Tag}}: return {{.FullName}}
	{{end}}{{end}}
	default: return 0
	}
}
{{end}}

type {{.Node.Name}}_List struct { {{capnp}}.List }

func New{{.Node.Name}}_List(s *{{capnp}}.Segment, sz int32) {{.Node.Name}}_List {
	l, err := {{capnp}}.NewUInt16List(s, sz)
	if err != nil {
		panic(err)
	}
	return {{.Node.Name}}_List{l.List}
}

func (l {{.Node.Name}}_List) At(i int) {{.Node.Name}} {
	ul := {{capnp}}.UInt16List{List: l.List}
	return {{.Node.Name}}(ul.At(i))
}

func (l {{.Node.Name}}_List) Set(i int, v {{.Node.Name}}) {
	ul := {{capnp}}.UInt16List{List: l.List}
	ul.Set(i, uint16(v))
}
{{end}}


{{define "structTypes"}}{{with .Annotations.Doc}}// {{.}}
{{end}}type {{.Node.Name}} {{if .IsBase}}struct{ {{capnp}}.Struct }{{else}}{{.BaseNode.Name}}{{end}}
{{end}}


{{define "newStructFunc"}}
func New{{.Node.Name}}(s *{{capnp}}.Segment) {{.Node.Name}} {
	st, err := {{capnp}}.NewStruct(s, {{.Node.ObjectSize}})
	if err != nil {
		panic(err)
	}
	return {{.Node.Name}}{st}
}

func NewRoot{{.Node.Name}}(s *{{capnp}}.Segment) {{.Node.Name}} {
	st, err := {{capnp}}.NewRootStruct(s, {{.Node.ObjectSize}})
	if err != nil {
		panic(err)
	}
	return {{.Node.Name}}{st}
}

func ReadRoot{{.Node.Name}}(msg *{{capnp}}.Message) {{.Node.Name}} {
	root, err := msg.RootPtr()
	if err != nil {
		panic(err)
	}
	return {{.Node.Name}}{root.Struct()}
}
{{end}}


{{define "structFuncs"}}
{{if gt .Node.StructGroup.DiscriminantCount 0}}
func (s {{.Node.Name}}) Which() {{.Node.Name}}_Which {
	return {{.Node.Name}}_Which(s.Struct.Uint16({{discriminantOffset .Node}}))
}
{{end}}
{{end}}


{{define "settag"}}{{if hasDiscriminant .Field}}s.Struct.SetUint16({{discriminantOffset .Node}}, {{.Field.DiscriminantValue}}){{end}}{{end}}


{{define "hasfield"}}
func (s {{.Node.Name}}) Has{{.Field.Name|title}}() bool {
	p, err := s.Struct.Ptr({{.Field.Slot.Offset}})
	return p.IsValid() || err != nil 
}
{{end}}


{{define "structGroup"}}func (s {{.Node.Name}}) {{.Field.Name|title}}() {{.Group.Name}} { return {{.Group.Name}}(s) }
{{if hasDiscriminant .Field}}
func (s {{.Node.Name}}) Set{{.Field.Name|title}}() { {{template "settag" .}} }
{{end}}{{end}}


{{define "structVoidField"}}{{if hasDiscriminant .Field}}
func (s {{.Node.Name}}) Set{{.Field.Name|title}}() {
	{{template "settag" .}}
}
{{end}}{{end}}


{{define "structBoolField"}}
func (s {{.Node.Name}}) {{.Field.Name|title}}() bool {
	return {{if .Default}}!{{end}}s.Struct.Bit({{.Field.Slot.Offset}})
}

func (s {{.Node.Name}}) Set{{.Field.Name|title}}(v bool) {
	{{template "settag" .}}
	s.Struct.SetBit({{.Field.Slot.Offset}}, {{if .Default}}!{{end}}v)
}
{{end}}


{{define "structUintField"}}
func (s {{.Node.Name}}) {{.Field.Name|title}}() uint{{.Bits}} {
	return s.Struct.Uint{{.Bits}}({{.Offset}}){{with .Default}} ^ {{.}}{{end}}
}

func (s {{.Node.Name}}) Set{{.Field.Name|title}}(v uint{{.Bits}}) {
	{{template "settag" .}}
	s.Struct.SetUint{{.Bits}}({{.Offset}}, v{{with .Default}}^{{.}}{{end}})
}
{{end}}


{{define "structIntField"}}
func (s {{.Node.Name}}) {{.Field.Name|title}}() {{.ReturnType}} {
	return {{.ReturnType}}(s.Struct.Uint{{.Bits}}({{.Offset}}){{with .Default}} ^ {{.}}{{end}})
}

func (s {{.Node.Name}}) Set{{.Field.Name|title}}(v {{.ReturnType}}) {
	{{template "settag" .}}
	s.Struct.SetUint{{.Bits}}({{.Offset}}, uint{{.Bits}}(v){{with .Default}}^{{.}}{{end}})
}
{{end}}


{{define "structFloatField"}}
func (s {{.Node.Name}}) {{.Field.Name|title}}() float{{.Bits}} {
	return {{math}}.Float{{.Bits}}frombits(s.Struct.Uint{{.Bits}}({{.Offset}}){{with .Default}} ^ {{printf "%#x" .}}{{end}})
}

func (s {{.Node.Name}}) Set{{.Field.Name|title}}(v float{{.Bits}}) {
	{{template "settag" .}}
	s.Struct.SetUint{{.Bits}}({{.Offset}}, {{math}}.Float{{.Bits}}bits(v){{with .Default}}^{{printf "%#x" .}}{{end}})
}
{{end}}


{{define "structTextField"}}
func (s {{.Node.Name}}) {{.Field.Name|title}}() string {
	p, err := s.Struct.Ptr({{.Field.Slot.Offset}})
	if err != nil {
		panic(err)
	}
	{{with .Default}}
	return p.TextDefault({{printf "%q" .}})
	{{else}}
	return p.Text()
	{{end}}
}

{{template "hasfield" .}}

func (s {{.Node.Name}}) {{.Field.Name|title}}Bytes() []byte {
	p, err := s.Struct.Ptr({{.Field.Slot.Offset}})
	if err != nil {
		panic(err)
	}
	{{with .Default}}
	return p.DataDefault([]byte({{printf "%q" .}}))
	{{else}}
	return p.Data()
	{{end}}
}

func (s {{.Node.Name}}) Set{{.Field.Name|title}}(v string) error {
	{{template "settag" .}}
	t, err := {{capnp}}.NewText(s.Struct.Segment(), v)
	if err != nil {
		return err
	}
	return s.Struct.SetPtr({{.Field.Slot.Offset}}, t.List.ToPtr())
}
{{end}}


{{define "structDataField"}}
func (s {{.Node.Name}}) {{.Field.Name|title}}() {{.FieldType}} {
	p, err := s.Struct.Ptr({{.Field.Slot.Offset}})
	if err != nil {
		panic(err)
	}
	{{with .Default}}
	return {{$.FieldType}}(p.DataDefault({{printf "%#v" .}}))
	{{else}}
	return {{.FieldType}}(p.Data())
	{{end}}
}

{{template "hasfield" .}}

func (s {{.Node.Name}}) Set{{.Field.Name|title}}(v {{.FieldType}}) error {
	{{template "settag" .}}
	d, err := {{capnp}}.NewData(s.Struct.Segment(), []byte(v))
	if err != nil {
		return err
	}
	return s.Struct.SetPtr({{.Field.Slot.Offset}}, d.List.ToPtr())
}
{{end}}


{{define "structStructField"}}
func (s {{.Node.Name}}) {{.Field.Name|title}}() {{.FieldType}} {
	p, err := s.Struct.Ptr({{.Field.Slot.Offset}})
	if err != nil {
		panic(err)
	}
	{{if .Default.IsValid}}
	ss, err := p.StructDefault({{.Default}})
	if err != nil {
		panic(err)
	}
	return {{.FieldType}}{Struct: ss}
	{{else}}
	return {{.FieldType}}{Struct: p.Struct()}
	{{end}}
}

{{template "hasfield" .}}

func (s {{.Node.Name}}) Set{{.Field.Name|title}}(v {{.FieldType}}) error {
	{{template "settag" .}}
	return s.Struct.SetPtr({{.Field.Slot.Offset}}, v.Struct.ToPtr())
}

// New{{.Field.Name|title}} sets the {{.Field.Name}} field to a newly
// allocated {{.FieldType}} struct, preferring placement in s's segment.
func (s {{.Node.Name}}) New{{.Field.Name|title}}() {{.FieldType}} {
	{{template "settag" .}}
	ss, err := {{.TypeNode.RemoteNew .Node}}(s.Struct.Segment())
	if err != nil {
		panic(err)
	}
	if err := s.Struct.SetPtr({{.Field.Slot.Offset}}, ss.Struct.ToPtr()); err != nil {
		panic(err)
	}
	return ss
}
{{end}}


{{define "structPointerField"}}
func (s {{.Node.Name}}) {{.Field.Name|title}}() {{capnp}}.Pointer {
	{{if .Default.IsValid}}
	p, err := s.Struct.Pointer({{.Field.Slot.Offset}})
	if err != nil {
		panic(err)
	}
	pp, err := {{capnp}}.PointerDefault(p, {{.Default}})
	if err != nil {
		panic(err)
	}
	return pp
	{{else}}
	p, err := s.Struct.Pointer({{.Field.Slot.Offset}})
	if err != nil {
		panic(err)
	}	
	return p
	{{end}}
}

{{template "hasfield" .}}

func (s {{.Node.Name}}) {{.Field.Name|title}}Ptr() {{capnp}}.Ptr {
	{{if .Default.IsValid}}
	p, err := s.Struct.Ptr({{.Field.Slot.Offset}})
	if err != nil {
		panic(err)
	}
	pp, err := p.Default({{.Default}})
	if err != nil {
		panic(err)
	}	
	return pp
	{{else}}
	pp, err := s.Struct.Ptr({{.Field.Slot.Offset}})
	if err != nil {
		panic(err)
	}	
	return pp
	{{end}}
}

func (s {{.Node.Name}}) Set{{.Field.Name|title}}(v {{capnp}}.Pointer) error {
	{{template "settag" .}}
	return s.Struct.SetPointer({{.Field.Slot.Offset}}, v)
}

func (s {{.Node.Name}}) Set{{.Field.Name|title}}Ptr(v {{capnp}}.Ptr) error {
	{{template "settag" .}}
	return s.Struct.SetPtr({{.Field.Slot.Offset}}, v)
}
{{end}}


{{define "structListField"}}
func (s {{.Node.Name}}) {{.Field.Name|title}}() {{.FieldType}} {
	p, err := s.Struct.Ptr({{.Field.Slot.Offset}})
	if err != nil {
		panic(err)
	}
	{{if .Default.IsValid}}
	l, err := p.ListDefault({{.Default}})
	if err != nil {
		panic(err)
	}
	return {{.FieldType}}{List: l}
	{{else}}
	return {{.FieldType}}{List: p.List()}
	{{end}}
}

{{template "hasfield" .}}

func (s {{.Node.Name}}) Set{{.Field.Name|title}}(v {{.FieldType}}) error {
	{{template "settag" .}}
	return s.Struct.SetPtr({{.Field.Slot.Offset}}, v.List.ToPtr())
}
{{end}}


{{define "structInterfaceField"}}
func (s {{.Node.Name}}) {{.Field.Name|title}}() {{.FieldType}} {
	p, err := s.Struct.Ptr({{.Field.Slot.Offset}})
	if err != nil {
		panic(err)
	}
	return {{.FieldType}}{Client: p.Interface().Client()}
}

{{template "hasfield" .}}

func (s {{.Node.Name}}) Set{{.Field.Name|title}}(v {{.FieldType}}) error {
	{{template "settag" .}}
	seg := s.Segment()
	if seg == nil {
		{{/* TODO(light): error? */}}
		return nil
	}
	var in capnp.Interface
	if v.Client != nil {
		in = {{capnp}}.NewInterface(seg, seg.Message().AddCap(v.Client))
	}
	return s.Struct.SetPtr({{.Field.Slot.Offset}}, in.ToPtr())
}
{{end}}


{{define "structList"}}// {{.Node.Name}}_List is a list of {{.Node.Name}}.
type {{.Node.Name}}_List struct{ {{capnp}}.List }

// New{{.Node.Name}} creates a new list of {{.Node.Name}}.
func New{{.Node.Name}}_List(s *{{capnp}}.Segment, sz int32) {{.Node.Name}}_List {
	l, err := {{capnp}}.NewCompositeList(s, {{.Node.ObjectSize}}, sz)
	if err != nil  {
		panic(err)
	}
	return {{.Node.Name}}_List{l}
}

func (s {{.Node.Name}}_List) At(i int) {{.Node.Name}} { return {{.Node.Name}}{ s.List.Struct(i) } }
func (s {{.Node.Name}}_List) Set(i int, v {{.Node.Name}}) error { return s.List.SetStruct(i, v.Struct) }
{{end}}


{{define "structEnums"}}type {{.Node.Name}}_Which uint16

const (
{{range .Fields}}	{{$.Node.Name}}_Which_{{.Name}} {{$.Node.Name}}_Which = {{.DiscriminantValue}}
{{end}}
)

func (w {{.Node.Name}}_Which) String() string {
	const s = {{.EnumString.ValueString|printf "%q"}}
	switch w {
	{{range $i, $f := .Fields}}case {{$.Node.Name}}_Which_{{.Name}}:
		return s{{$.EnumString.SliceFor $i}}
	{{end}}
	}
	return "{{.Node.Name}}_Which(" + {{strconv}}.FormatUint(uint64(w), 10) + ")"
}

{{end}}


{{define "annotation"}}const {{.Node.Name}} = uint64({{.Node.Id|printf "%#x"}})
{{end}}


{{define "promise"}}// {{.Node.Name}}_Promise is a wrapper for a {{.Node.Name}} promised by a client call.
type {{.Node.Name}}_Promise struct { *{{capnp}}.Pipeline }

func (p {{.Node.Name}}_Promise) Struct() ({{.Node.Name}}, error) {
	s, err := p.Pipeline.Struct()
	return {{.Node.Name}}{s}, err
}
{{end}}


{{define "promiseFieldStruct"}}
func (p {{.Node.Name}}_Promise) {{.Field.Name|title}}() {{.Struct.RemoteName .Node}}_Promise {
	return {{.Struct.RemoteName .Node}}_Promise{Pipeline: p.Pipeline.{{if .Default.IsValid}}GetPipelineDefault({{.Field.Slot.Offset}}, {{.Default}}){{else}}GetPipeline({{.Field.Slot.Offset}}){{end}} }
}
{{end}}


{{define "promiseFieldAnyPointer"}}
func (p {{.Node.Name}}_Promise) {{.Field.Name|title}}() *{{capnp}}.Pipeline {
	return p.Pipeline.GetPipeline({{.Field.Slot.Offset}})
}
{{end}}


{{define "promiseFieldInterface"}}
func (p {{.Node.Name}}_Promise) {{.Field.Name|title}}() {{.Interface.RemoteName .Node}} {
	return {{.Interface.RemoteName .Node}}{Client: p.Pipeline.GetPipeline({{.Field.Slot.Offset}}).Client()}
}
{{end}}


{{define "promiseGroup"}}func (p {{.Node.Name}}_Promise) {{.Field.Name|title}}() {{.Group.Name}}_Promise { return {{.Group.Name}}_Promise{p.Pipeline} }
{{end}}


{{define "interfaceClient"}}{{with .Annotations.Doc}}// {{.}}
{{end}}type {{.Node.Name}} struct { Client {{capnp}}.Client }

{{range .Methods}}
func (c {{$.Node.Name}}) {{.Name|title}}(ctx {{context}}.Context, params func({{.Params.RemoteName $.Node}}) error, opts ...{{capnp}}.CallOption) {{.Results.RemoteName $.Node}}_Promise {
	if c.Client == nil {
		return {{.Results.RemoteName $.Node}}_Promise{Pipeline: {{capnp}}.NewPipeline({{capnp}}.ErrorAnswer({{capnp}}.ErrNullClient))}
	}
	call := &{{capnp}}.Call{
		Ctx: ctx,
		Method: {{capnp}}.Method{
			{{template "_interfaceMethod" .}}
		},
		Options: {{capnp}}.NewCallOptions(opts),
	}
	if params != nil {
		call.ParamsSize = {{.Params.ObjectSize}}
		call.ParamsFunc = func(s {{capnp}}.Struct) error { return params({{.Params.RemoteName $.Node}}{Struct: s}) }
	}
	return {{.Results.RemoteName $.Node}}_Promise{Pipeline: {{capnp}}.NewPipeline(c.Client.Call(call))}
}
{{end}}
{{end}}


{{define "interfaceServer"}}type {{.Node.Name}}_Server interface {
	{{range .Methods}}
	{{.Name|title}}({{.Interface.RemoteName $.Node}}_{{.Name}}) error
	{{end}}
}

func {{.Node.Name}}_ServerToClient(s {{.Node.Name}}_Server) {{.Node.Name}} {
	c, _ := s.({{server}}.Closer)
	return {{.Node.Name}}{Client: {{server}}.New({{.Node.Name}}_Methods(nil, s), c)}
}

func {{.Node.Name}}_Methods(methods []{{server}}.Method, s {{.Node.Name}}_Server) []{{server}}.Method {
	if cap(methods) == 0 {
		methods = make([]{{server}}.Method, 0, {{len .Methods}})
	}
	{{range .Methods}}
	methods = append(methods, {{server}}.Method{
		Method: {{capnp}}.Method{
			{{template "_interfaceMethod" .}}
		},
		Impl: func(c {{context}}.Context, opts {{capnp}}.CallOptions, p, r {{capnp}}.Struct) error {
			call := {{.Interface.RemoteName $.Node}}_{{.Name}}{c, opts, {{.Params.RemoteName $.Node}}{Struct: p}, {{.Results.RemoteName $.Node}}{Struct: r} }
			return s.{{.Name|title}}(call)
		},
		ResultsSize: {{.Results.ObjectSize}},
	})
	{{end}}
	return methods
}
{{range .Methods}}{{if eq .Interface.Id $.Node.Id}}
// {{$.Node.Name}}_{{.Name}} holds the arguments for a server call to {{$.Node.Name}}.{{.Name}}.
type {{$.Node.Name}}_{{.Name}} struct {
	Ctx     {{context}}.Context
	Options {{capnp}}.CallOptions
	Params  {{.Params.RemoteName $.Node}}
	Results {{.Results.RemoteName $.Node}}
}
{{end}}{{end}}
{{end}}


{{define "_interfaceMethod"}}
			InterfaceID: {{.Interface.Id|printf "%#x"}},
			MethodID: {{.ID}},
			InterfaceName: {{.Interface.DisplayName|printf "%q"}},
			MethodName: {{.OriginalName|printf "%q"}},
{{end}}

{{define "structValue"}}{{.Typ.RemoteName .Node}}{Struct: {{capnp}}.MustUnmarshalRootPtr({{.Value}}).Struct()}{{end}}

{{define "pointerValue"}}{{capnp}}.MustUnmarshalRootPtr({{.Value}}){{end}}

{{define "listValue"}}{{.Typ}}{List: {{capnp}}.MustUnmarshalRootPtr({{.Value}}).List()}{{end}}
`))

type annotationParams struct {
	Node *node
}

type enumParams struct {
	Node        *node
	Annotations *annotations
	EnumValues  []enumval
}

type structTypesParams struct {
	Node        *node
	Annotations *annotations
	BaseNode    *node
}

func (p structTypesParams) IsBase() bool {
	return p.Node == p.BaseNode
}

type newStructParams struct {
	Node *node
}

type structFuncsParams struct {
	Node *node
}

type structGroupParams struct {
	Node  *node
	Group *node
	Field field
}

type structFieldParams struct {
	Node        *node
	Field       field
	Annotations *annotations
	FieldType   string
}

type structBoolFieldParams struct {
	structFieldParams
	Default bool
}

type structUintFieldParams struct {
	structFieldParams
	Bits    uint
	Default uint64
}

func (p structUintFieldParams) Offset() uint32 {
	return p.Field.Slot().Offset() * uint32(p.Bits/8)
}

type structIntFieldParams struct {
	structUintFieldParams
	EnumName string
}

func (p structIntFieldParams) ReturnType() string {
	if p.EnumName != "" {
		return p.EnumName
	}
	return fmt.Sprintf("int%d", p.Bits)
}

type structTextFieldParams struct {
	structFieldParams
	Default string
}

type structDataFieldParams struct {
	structFieldParams
	Default []byte
}

type structObjectFieldParams struct {
	structFieldParams
	TypeNode *node
	Default  staticDataRef
}

type structListParams struct {
	Node *node
}

type structEnumsParams struct {
	Node       *node
	Fields     []field
	EnumString enumString
}

type promiseTemplateParams struct {
	Node   *node
	Fields []field
}

type promiseGroupTemplateParams struct {
	Node  *node
	Field field
	Group *node
}

type promiseFieldStructTemplateParams struct {
	Node    *node
	Field   field
	Struct  *node
	Default staticDataRef
}

type promiseFieldAnyPointerTemplateParams struct {
	Node  *node
	Field field
}

type promiseFieldInterfaceTemplateParams struct {
	Node      *node
	Field     field
	Interface *node
}

type interfaceClientTemplateParams struct {
	Node        *node
	Annotations *annotations
	Methods     []interfaceMethod
}

type interfaceServerTemplateParams struct {
	Node        *node
	Annotations *annotations
	Methods     []interfaceMethod
}

type structValueTemplateParams struct {
	Node  *node
	Typ   *node
	Value staticDataRef
}

type pointerValueTemplateParams struct {
	Value staticDataRef
}

type listValueTemplateParams struct {
	Typ   string
	Value staticDataRef
}
