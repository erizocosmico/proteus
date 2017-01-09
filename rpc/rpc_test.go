package rpc

import (
	"bytes"
	"go/ast"
	"go/importer"
	"go/parser"
	"go/printer"
	"go/token"
	"go/types"
	"testing"

	"github.com/src-d/proteus/protobuf"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type RPCSuite struct {
	suite.Suite
	g *Generator
}

func (s *RPCSuite) SetupTest() {
	s.g = NewGenerator()
}

const expectedImplType = "type Foo struct {\n}"

func (s *RPCSuite) TestDeclImplType() {
	output, err := render(s.g.declImplType("Foo"))
	s.Nil(err)
	s.Equal(expectedImplType, output)
}

const expectedConstructor = `func NewFoo() *Foo {
	return &Foo{}
}`

func (s *RPCSuite) TestDeclConstructor() {
	output, err := render(s.g.declConstructor("Foo", "NewFoo"))
	s.Nil(err)
	s.Equal(expectedConstructor, output)
}

const expectedFuncNotGenerated = `func (s *FooServer) DoFoo(ctx context.Context, in *fake.Foo) (result *fake.Bar, err error) {
	result = new(fake.Bar)
	result = DoFoo(in)
	return
}`

const expectedFuncGenerated = `func (s *FooServer) DoFoo(ctx context.Context, in *FooRequest) (result *FooResponse, err error) {
	result = new(FooResponse)
	result.Result1 = DoFoo(in.Arg1, in.Arg2, in.Arg3)
	return
}`

const expectedFuncGeneratedWithError = `func (s *FooServer) DoFoo(ctx context.Context, in *FooRequest) (result *FooResponse, err error) {
	result = new(FooResponse)
	result.Result1, err = DoFoo(in.Arg1, in.Arg2, in.Arg3)
	return
}`

const expectedMethod = `func (s *FooServer) Fooer_DoFoo(ctx context.Context, in *FooRequest) (result *FooResponse, err error) {
	result = new(FooResponse)
	result.Result1, err = s.Fooer.DoFoo(in.Arg1, in.Arg2, in.Arg3)
	return
}`

const expectedMethodExternalInput = `func (s *FooServer) T_Foo(ctx context.Context, in *ast.BlockStmt) (result *T_FooResponse, err error) {
	result = new(T_FooResponse)
	result.Result1 = s.T.Foo(in)
	return
}`

const expectedFuncEmptyInAndOut = `func (s *FooServer) Empty(ctx context.Context, in *Empty) (result *Empty, err error) {
	Empty()
	return
}`

const expectedFuncEmptyInAndOutWithError = `func (s *FooServer) Empty(ctx context.Context, in *Empty) (result *Empty, err error) {
	err = Empty()
	return
}`

func (s *RPCSuite) TestDeclMethod() {
	cases := []struct {
		name   string
		rpc    *protobuf.RPC
		output string
	}{
		{
			"func not generated",
			&protobuf.RPC{
				Name:   "DoFoo",
				Method: "DoFoo",
				Input:  protobuf.NewNamed("", "Foo"),
				Output: protobuf.NewNamed("", "Bar"),
			},
			expectedFuncNotGenerated,
		},
		{
			"func generated",
			&protobuf.RPC{
				Name:   "DoFoo",
				Method: "DoFoo",
				Input:  protobuf.NewGeneratedNamed("", "FooRequest"),
				Output: protobuf.NewGeneratedNamed("", "FooResponse"),
			},
			expectedFuncGenerated,
		},
		{
			"func generated with error",
			&protobuf.RPC{
				Name:     "DoFoo",
				Method:   "DoFoo",
				HasError: true,
				Input:    protobuf.NewGeneratedNamed("", "FooRequest"),
				Output:   protobuf.NewGeneratedNamed("", "FooResponse"),
			},
			expectedFuncGeneratedWithError,
		},
		{
			"method call",
			&protobuf.RPC{
				Name:     "Fooer_DoFoo",
				Method:   "DoFoo",
				Recv:     "Fooer",
				HasError: true,
				Input:    protobuf.NewGeneratedNamed("", "FooRequest"),
				Output:   protobuf.NewGeneratedNamed("", "FooResponse"),
			},
			expectedMethod,
		},
		{
			"method with external type input",
			&protobuf.RPC{
				Name:     "T_Foo",
				Method:   "Foo",
				Recv:     "T",
				HasError: false,
				Input:    protobuf.NewNamed("go.ast", "BlockStmt"),
				Output:   protobuf.NewGeneratedNamed("", "T_FooResponse"),
			},
			expectedMethodExternalInput,
		},
		{
			"func with empty input and output",
			&protobuf.RPC{
				Name:   "Empty",
				Method: "Empty",
				Input:  protobuf.NewGeneratedNamed("", "Empty"),
				Output: protobuf.NewGeneratedNamed("", "Empty"),
			},
			expectedFuncEmptyInAndOut,
		},
		{
			"func with empty input and output with error",
			&protobuf.RPC{
				Name:     "Empty",
				Method:   "Empty",
				HasError: true,
				Input:    protobuf.NewGeneratedNamed("", "Empty"),
				Output:   protobuf.NewGeneratedNamed("", "Empty"),
			},
			expectedFuncEmptyInAndOutWithError,
		},
	}

	proto := &protobuf.Package{
		Messages: []*protobuf.Message{
			&protobuf.Message{
				Name:   "FooRequest",
				Fields: make([]*protobuf.Field, 3),
			},
			&protobuf.Message{
				Name:   "FooResponse",
				Fields: make([]*protobuf.Field, 1),
			},
			&protobuf.Message{
				Name:   "T_FooResponse",
				Fields: make([]*protobuf.Field, 1),
			},
			&protobuf.Message{
				Name: "Empty",
			},
		},
	}

	ctx := &context{
		implName: "FooServer",
		proto:    proto,
		pkg:      s.fakePkg(),
	}

	for _, c := range cases {
		output, err := render(s.g.declMethod(ctx, c.rpc))
		s.Nil(err, c.name, c.name)
		s.Equal(c.output, output, c.name)
	}
}

func TestServiceImplName(t *testing.T) {
	require.Equal(t, "fooServiceServer", serviceImplName(&protobuf.Package{
		Name: "foo",
	}))
}

func TestConstructorName(t *testing.T) {
	require.Equal(t, "NewFooServiceServer", constructorName(&protobuf.Package{
		Name: "foo",
	}))
}

const testPkg = `package fake

import "go/ast"

type Foo struct{}
type Bar struct {}

func DoFoo(in *Foo) *Bar {
	return nil
}

func MoreFoo(a int) *ast.BlockStmt {
	return nil
}

type T struct{}

func (*T) Foo(s *ast.BlockStmt) int {
	return 0
}
`

func (s *RPCSuite) fakePkg() *types.Package {
	fs := token.NewFileSet()

	f, err := parser.ParseFile(fs, "src.go", testPkg, 0)
	if err != nil {
		panic(err)
	}

	config := types.Config{
		FakeImportC: true,
		Importer:    importer.Default(),
	}

	pkg, err := config.Check("", fs, []*ast.File{f}, nil)
	s.Nil(err)
	return pkg
}

func render(decl ast.Decl) (string, error) {
	var buf bytes.Buffer
	if err := printer.Fprint(&buf, token.NewFileSet(), decl); err != nil {
		return "", err
	}

	return buf.String(), nil
}

func TestRPCSuite(t *testing.T) {
	suite.Run(t, new(RPCSuite))
}
