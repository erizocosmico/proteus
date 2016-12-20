package protobuf

import "github.com/src-d/proteus/scanner"

// Plugin defines the contract that all plugins must implement.
// It basically defines some callbacks that will be invoked
// throughout the whole transforming process.
// All methods will be called AFTER the element is transformed.
// So all you can do is modify it, not remove it from the package, etc.
type Plugin interface {
	// Package is invoked after all the package has been transformed
	// into a protobuf package representation.
	Package(pkg *Package, old *scanner.Package)

	// Message is invoked after a struct is processed. Along with the
	// message the package will be passed.
	Message(pkg *Package, m *Message, old *scanner.Struct)

	// Field is invoked after a message field is processed.
	// Along with the message field the package will be passed.
	Field(pkg *Package, f *Field, old *scanner.Field)

	// Enum is invoked after an enum is processed. Along with the enum
	// the package will be passed.
	Enum(pkg *Package, e *Enum, old *scanner.Enum)

	// EnumValue is invoked after an enum value is processed. Along with
	// the enum value the package will be passed.
	EnumValue(pkg *Package, v *EnumValue, old string)
}

// plugins is a collection of plugins and a Plugin itself. Will call all the
// methods of the plugins it contains when its methods are called.
type plugins []Plugin

func (p *plugins) add(pl Plugin) {
	*p = append(*p, pl)
}

func (p plugins) Package(pkg *Package, old *scanner.Package) {
	for _, pl := range p {
		pl.Package(pkg, old)
	}
}

func (p plugins) Message(pkg *Package, msg *Message, old *scanner.Struct) {
	for _, pl := range p {
		pl.Message(pkg, msg, old)
	}
}

func (p plugins) Field(pkg *Package, f *Field, old *scanner.Field) {
	for _, pl := range p {
		pl.Field(pkg, f, old)
	}
}

func (p plugins) Enum(pkg *Package, e *Enum, old *scanner.Enum) {
	for _, pl := range p {
		pl.Enum(pkg, e, old)
	}
}

func (p plugins) EnumValue(pkg *Package, v *EnumValue, old string) {
	for _, pl := range p {
		pl.EnumValue(pkg, v, old)
	}
}
