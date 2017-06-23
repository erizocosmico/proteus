package subpkg

// Point ...
//proteus:generate
type Point struct {
	X int
	Y int
}

// Dist ...
func (p *Point) Dist(p2 Point) float64 {
	return .0
}

// NotGenerated ...
type NotGenerated struct{}

// Foo ...
func Foo(a int) (float64, error) {
	return float64(a), nil
}

// Generated ...
//proteus:generate api_path:"/generated" api_method:"get"
func Generated(a string) (bool, error) {
	return len(a) > 0, nil
}

// GeneratedMethod ...
//proteus:generate api_path:"/point/bar" api_method:"get"
func (p Point) GeneratedMethod(a int32) *Point {
	return &p
}

// GeneratedMethodOnPointer ...
//proteus:generate api_path:"/point/foo" api_method:"get"
func (p *Point) GeneratedMethodOnPointer(a bool) *Point {
	return p
}

// MyContainer ...
type MyContainer struct {
	name string
}

// Name ...
//proteus:generate api_path:"/container/name" api_method:"get"
func (c *MyContainer) Name() string {
	return c.Name()
}
