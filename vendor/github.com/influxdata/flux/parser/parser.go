// +build !parser_debug

package parser

//go:generate pigeon -optimize-parser -o flux.go flux.peg

import (
	"github.com/influxdata/flux/ast"
)

// NewAST parses Flux query and produces an ast.Program
func NewAST(flux string, opts ...Option) (*ast.Program, error) {
	f, err := Parse("", []byte(flux), opts...)
	if err != nil {
		return nil, err
	}
	return f.(*ast.Program), nil
}
