package model

// methodCall describes a call to c.Render(..)
// It documents the argument names used, in order to propagate them to RenderArgs.
type MethodCall struct {
	Path  string // e.g. "myapp/app/controllers.(*Application).Action"
	Line  int
	Names []string
}

// MethodSpec holds the information of one Method.
type MethodSpec struct {
	Name        string        // Name of the method, e.g. "Index"
	Args        []*MethodArg  // Argument descriptors
	RenderCalls []*MethodCall // Descriptions of Render() invocations from this Method.
}

// MethodArg holds the information of one argument.
type MethodArg struct {
	Name       string   // Name of the argument.
	TypeExpr   TypeExpr // The name of the type, e.g. "int", "*pkg.UserType"
	ImportPath string   // If the arg is of an imported type, this is the import path.
}
