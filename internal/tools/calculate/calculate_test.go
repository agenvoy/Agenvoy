package calculate

import (
	"strings"
	"testing"
)

func TestCalc(t *testing.T) {
	tests := []struct {
		name    string
		expr    string
		want    string
		wantErr bool
	}{
		// empty
		{"empty", "", "", true},
		// invalid syntax
		{"invalid syntax", "(", "", true},
		// integers: basic ops
		{"add ints", "2+3", "5", false},
		{"sub ints", "10-3", "7", false},
		{"mul ints", "4*5", "20", false},
		{"div ints exact", "10/2", "5", false},
		{"div ints truncated", "10/3", "3", false},
		{"mod ints", "10%3", "1", false},
		// integer errors
		{"div by zero int", "5/0", "", true},
		{"mod by zero int", "5%0", "", true},
		// power via XOR operator
		{"power int", "2^3", "8", false},
		// float arithmetic
		{"add floats", "1.5+2.5", "4", false},
		{"div float result", "5.0/2.0", "2.5", false},
		{"div float exact", "6.0/2.0", "3", false},
		// float errors
		{"div by zero float", "5.0/0", "", true},
		{"mod by zero float", "5.0%0", "", true},
		// mixed int/float
		{"mixed add", "1+0.5", "1.5", false},
		// unary
		{"unary neg int", "-5", "-5", false},
		{"unary pos int", "+5", "5", false},
		{"unary neg float", "-1.5", "-1.5", false},
		// parentheses
		{"paren", "(2+3)*4", "20", false},
		{"nested paren", "((2+3)*2)+1", "11", false},
		// big integer
		{"big int", "1000000000*1000000000", "1000000000000000000", false},
		// functions: sqrt
		{"sqrt perfect", "sqrt(4)", "2", false},
		{"sqrt float", "sqrt(2)", "", false}, // just check no error
		{"sqrt negative", "sqrt(-1)", "", true},
		// functions: abs
		{"abs negative", "abs(-5)", "5", false},
		{"abs positive", "abs(3)", "3", false},
		// functions: ceil/floor/round
		{"ceil", "ceil(1.2)", "2", false},
		{"floor", "floor(1.9)", "1", false},
		{"round up", "round(1.5)", "2", false},
		{"round down", "round(1.4)", "1", false},
		// functions: log
		{"log of 1", "log(1)", "0", false},
		{"log zero", "log(0)", "", true},
		{"log negative", "log(-1)", "", true},
		{"log2 of 8", "log2(8)", "3", false},
		{"log10 of 100", "log10(100)", "2", false},
		// functions: trig
		{"sin 0", "sin(0)", "0", false},
		{"cos 0", "cos(0)", "1", false},
		{"tan 0", "tan(0)", "0", false},
		// functions: pow
		{"pow 2 args", "pow(2,3)", "8", false},
		{"pow 1 arg", "pow(2)", "", true},
		// unknown function
		{"unknown fn", "unknown(5)", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Calc(tt.expr)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Calc(%q) error = %v, wantErr %v", tt.expr, err, tt.wantErr)
			}
			if !tt.wantErr && tt.want != "" && got != tt.want {
				t.Errorf("Calc(%q) = %q, want %q", tt.expr, got, tt.want)
			}
		})
	}
}

func TestCalc_FunctionNoArgs(t *testing.T) {
	// functions with no args should error at evalFunc level
	_, err := Calc("sqrt()")
	if err == nil {
		t.Error("expected error when sqrt called with no args")
	}
	if !strings.Contains(err.Error(), "requires arguments") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestCalc_UnsupportedBranches(t *testing.T) {
	// unsupported literal type (CHAR — syntactically valid, fails at eval)
	if _, err := Calc("'a'"); err == nil {
		t.Error("expected error for char literal")
	}

	// unsupported unary operator (^ is XOR/complement in Go AST)
	if _, err := Calc("^5"); err == nil {
		t.Error("expected error for unsupported unary op ^")
	}

	// unsupported binary operator — int path
	if _, err := Calc("2|3"); err == nil {
		t.Error("expected error for unsupported binary op | on ints")
	}

	// unsupported binary operator — float path
	if _, err := Calc("1.5|2.5"); err == nil {
		t.Error("expected error for unsupported binary op | on floats")
	}

	// evalFunc with non-Ident fun (SelectorExpr: math.Sqrt)
	if _, err := Calc("math.Sqrt(4)"); err == nil {
		t.Error("expected error for non-ident function call")
	}

	// unsupported expression type (Ident — not a literal, unary, binary, paren or call)
	if _, err := Calc("x"); err == nil {
		t.Error("expected error for bare identifier")
	}

	// eval error propagated through evalFunc arg evaluation
	if _, err := Calc("sqrt(x)"); err == nil {
		t.Error("expected error when function arg fails to eval")
	}

	// BinaryExpr: left eval error
	if _, err := Calc("x+2"); err == nil {
		t.Error("expected error when left operand fails to eval")
	}

	// BinaryExpr: right eval error
	if _, err := Calc("2+x"); err == nil {
		t.Error("expected error when right operand fails to eval")
	}

	// UnaryExpr: inner eval error
	if _, err := Calc("-(x)"); err == nil {
		t.Error("expected error when unary operand fails to eval")
	}
}
