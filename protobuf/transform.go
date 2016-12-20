package protobuf

import (
	"bytes"
	"strings"
	"unicode"

	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"

	"github.com/src-d/proteus/report"
	"github.com/src-d/proteus/scanner"
)

// Transformer is in charge of converting scanned Go entities to protobuf
// entities as well as mapping between Go and Protobuf types.
// Take into account that custom mappings are used first to check for the
// corresponding type mapping, and then the default mappings to give the user
// ability to override any kind of type.
type Transformer struct {
	mappings TypeMappings
	plugins  plugins
}

// NewTransformer creates a new transformer instance.
func NewTransformer() *Transformer {
	return &Transformer{
		mappings: make(TypeMappings),
		plugins:  plugins{},
	}
}

func (t *Transformer) AddPlugin(p Plugin) {
	t.plugins.add(p)
}

// SetMappings will set the custom mappings of the transformer. If nil is
// provided, the change will be ignored.
func (t *Transformer) SetMappings(m TypeMappings) {
	if m == nil {
		return
	}
	t.mappings = m
}

// Transform converts a scanned package to a protobuf package.
func (t *Transformer) Transform(p *scanner.Package) *Package {
	pkg := &Package{
		Name:    toProtobufPkg(p.Path),
		Path:    p.Path,
		Options: make(Options),
	}

	for _, s := range p.Structs {
		msg := t.transformStruct(pkg, s)
		pkg.Messages = append(pkg.Messages, msg)
	}

	for _, e := range p.Enums {
		enum := t.transformEnum(pkg, e)
		pkg.Enums = append(pkg.Enums, enum)
	}

	t.plugins.Package(pkg, p)
	return pkg
}

func (t *Transformer) transformEnum(pkg *Package, e *scanner.Enum) *Enum {
	enum := &Enum{Name: e.Name, Options: make(Options)}

	for i, v := range e.Values {
		val := &EnumValue{
			Name:    toUpperSnakeCase(v),
			Value:   uint(i),
			Options: make(Options),
		}

		enum.Values = append(enum.Values, val)
		t.plugins.EnumValue(pkg, val, v)
	}

	t.plugins.Enum(pkg, enum, e)
	return enum
}

func (t *Transformer) transformStruct(pkg *Package, s *scanner.Struct) *Message {
	msg := &Message{Name: s.Name, Options: make(Options)}

	for i, f := range s.Fields {
		field := t.transformField(pkg, f, i+1)
		if field == nil {
			msg.Reserve(uint(i) + 1)
			report.Warn("field %q of struct %q has an invalid type, ignoring field but reserving its position", f.Name, s.Name)
			continue
		}

		msg.Fields = append(msg.Fields, field)
		t.plugins.Field(pkg, field, f)
	}

	t.plugins.Message(pkg, msg, s)
	return msg
}

func (t *Transformer) transformField(pkg *Package, field *scanner.Field, pos int) *Field {
	var typ Type
	var repeated = field.Type.IsRepeated()

	// []byte is the only repeated type that maps to
	// a non-repeated type in protobuf, so we handle
	// it a bit differently.
	if isByteSlice(field.Type) {
		typ = NewBasic("bytes")
		repeated = false
	} else {
		typ = t.transformType(pkg, field.Type)
		if typ == nil {
			return nil
		}
	}

	return &Field{
		Name:     toLowerSnakeCase(field.Name),
		Pos:      pos,
		Type:     typ,
		Repeated: repeated,
		Options:  make(Options),
	}
}

func (t *Transformer) transformType(pkg *Package, typ scanner.Type) Type {
	switch ty := typ.(type) {
	case *scanner.Named:
		protoType := t.findMapping(ty.String())
		if protoType != nil {
			pkg.Import(protoType)
			return protoType.Type()
		}

		pkg.ImportFromPath(ty.Path)
		return NewNamed(toProtobufPkg(ty.Path), ty.Name)
	case *scanner.Basic:
		protoType := t.findMapping(ty.Name)
		if protoType != nil {
			pkg.Import(protoType)
			return protoType.Type()
		}

		report.Warn("basic type %q is not defined in the mappings, ignoring", ty.Name)
	case *scanner.Map:
		return NewMap(
			t.transformType(pkg, ty.Key),
			t.transformType(pkg, ty.Value),
		)
	}

	return nil
}

func (t *Transformer) findMapping(name string) *ProtoType {
	typ := t.mappings[name]
	if typ == nil {
		typ = defaultMappings[name]
	}

	return typ
}

func isByteSlice(typ scanner.Type) bool {
	if t, ok := typ.(*scanner.Basic); ok && typ.IsRepeated() {
		return t.Name == "byte"
	}
	return false
}

func toProtobufPkg(path string) string {
	pkg := strings.Map(func(r rune) rune {
		if r == '/' || r == '.' {
			return '.'
		}

		if unicode.IsLetter(r) || unicode.IsNumber(r) {
			return r
		}

		return ' '
	}, path)
	pkg = strings.Replace(pkg, " ", "", -1)
	t := transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)
	pkg, _, _ = transform.String(t, pkg)
	return pkg
}

func toLowerSnakeCase(s string) string {
	var buf bytes.Buffer
	var lastWasUpper bool
	for i, r := range s {
		if unicode.IsUpper(r) && i != 0 && !lastWasUpper {
			buf.WriteRune('_')
		}
		lastWasUpper = unicode.IsUpper(r)
		buf.WriteRune(unicode.ToLower(r))
	}
	return buf.String()
}

func toUpperSnakeCase(s string) string {
	return strings.ToUpper(toLowerSnakeCase(s))
}
