package model

// The embedded type name takes the import path and structure name.
type EmbeddedTypeName struct {
	ImportPath, StructName string
}

// Convert the type to a properly formatted import line.
func (s *EmbeddedTypeName) String() string {
	return s.ImportPath + "." + s.StructName
}
