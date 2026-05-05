// * generate by claude sonnet 4.5
package calculator

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"math"
	"math/big"
	"strconv"
)

func Calc(expression string) (string, error) {
	if expression == "" {
		return "", fmt.Errorf("expression is required")
	}

	parseExpr, err := parser.ParseExpr(expression)
	if err != nil {
		return "", fmt.Errorf("parser.ParseExpr: %w", err)
	}

	result, err := eval(parseExpr)
	if err != nil {
		return "", err
	}

	if result.isInt {
		return result.i.String(), nil
	}
	if result.f == math.Trunc(result.f) && !math.IsInf(result.f, 0) {
		return strconv.FormatInt(int64(result.f), 10), nil
	}
	return strconv.FormatFloat(result.f, 'f', -1, 64), nil
}

type value struct {
	isInt bool
	i     *big.Int
	f     float64
}

func intVal(i *big.Int) value  { return value{isInt: true, i: i} }
func floatVal(f float64) value { return value{f: f} }

func (v value) toFloat() float64 {
	if v.isInt {
		f, _ := new(big.Float).SetInt(v.i).Float64()
		return f
	}
	return v.f
}

func eval(node ast.Expr) (value, error) {
	switch n := node.(type) {
	case *ast.BasicLit:
		if n.Kind == token.INT {
			i := new(big.Int)
			if _, ok := i.SetString(n.Value, 10); !ok {
				return value{}, fmt.Errorf("invalid number: %s", n.Value)
			}
			return intVal(i), nil
		}
		if n.Kind == token.FLOAT {
			f, err := strconv.ParseFloat(n.Value, 64)
			if err != nil {
				return value{}, fmt.Errorf("invalid number: %s", n.Value)
			}
			return floatVal(f), nil
		}
		return value{}, fmt.Errorf("unsupported literal type: %s", n.Kind)

	case *ast.UnaryExpr:
		v, err := eval(n.X)
		if err != nil {
			return value{}, err
		}
		switch n.Op {
		case token.SUB:
			if v.isInt {
				return intVal(new(big.Int).Neg(v.i)), nil
			}
			return floatVal(-v.f), nil
		case token.ADD:
			return v, nil
		}
		return value{}, fmt.Errorf("unsupported unary operator: %s", n.Op)

	case *ast.BinaryExpr:
		left, err := eval(n.X)
		if err != nil {
			return value{}, err
		}
		right, err := eval(n.Y)
		if err != nil {
			return value{}, err
		}
		if left.isInt && right.isInt {
			switch n.Op {
			case token.ADD:
				return intVal(new(big.Int).Add(left.i, right.i)), nil
			case token.SUB:
				return intVal(new(big.Int).Sub(left.i, right.i)), nil
			case token.MUL:
				return intVal(new(big.Int).Mul(left.i, right.i)), nil
			case token.QUO:
				if right.i.Sign() == 0 {
					return value{}, fmt.Errorf("division by zero")
				}
				return intVal(new(big.Int).Quo(left.i, right.i)), nil
			case token.REM:
				if right.i.Sign() == 0 {
					return value{}, fmt.Errorf("modulo by zero")
				}
				return intVal(new(big.Int).Rem(left.i, right.i)), nil
			case token.XOR:
				return floatVal(math.Pow(left.toFloat(), right.toFloat())), nil
			}
			return value{}, fmt.Errorf("unsupported operator: %s", n.Op)
		}
		lf, rf := left.toFloat(), right.toFloat()
		switch n.Op {
		case token.ADD:
			return floatVal(lf + rf), nil
		case token.SUB:
			return floatVal(lf - rf), nil
		case token.MUL:
			return floatVal(lf * rf), nil
		case token.QUO:
			if rf == 0 {
				return value{}, fmt.Errorf("division by zero")
			}
			return floatVal(lf / rf), nil
		case token.REM:
			if rf == 0 {
				return value{}, fmt.Errorf("modulo by zero")
			}
			return floatVal(math.Mod(lf, rf)), nil
		case token.XOR:
			return floatVal(math.Pow(lf, rf)), nil
		}
		return value{}, fmt.Errorf("unsupported operator: %s", n.Op)

	case *ast.ParenExpr:
		return eval(n.X)

	case *ast.CallExpr:
		return evalFunc(n)
	}

	return value{}, fmt.Errorf("unsupported expression type: %T", node)
}

func evalFunc(call *ast.CallExpr) (value, error) {
	ident, ok := call.Fun.(*ast.Ident)
	if !ok {
		return value{}, fmt.Errorf("unsupported function call")
	}

	if len(call.Args) == 0 {
		return value{}, fmt.Errorf("function %s requires arguments", ident.Name)
	}

	switch ident.Name {
	case "min", "max":
		return evalMinMax(ident.Name, call.Args)
	}

	a, err := eval(call.Args[0])
	if err != nil {
		return value{}, err
	}
	arg := a.toFloat()

	switch ident.Name {
	case "sqrt":
		if arg < 0 {
			return value{}, fmt.Errorf("sqrt of negative number")
		}
		return floatVal(math.Sqrt(arg)), nil
	case "abs":
		return floatVal(math.Abs(arg)), nil
	case "ceil":
		return floatVal(math.Ceil(arg)), nil
	case "floor":
		return floatVal(math.Floor(arg)), nil
	case "round":
		return floatVal(math.Round(arg)), nil
	case "log":
		if arg <= 0 {
			return value{}, fmt.Errorf("log of non-positive number")
		}
		return floatVal(math.Log(arg)), nil
	case "log2":
		if arg <= 0 {
			return value{}, fmt.Errorf("log2 of non-positive number")
		}
		return floatVal(math.Log2(arg)), nil
	case "log10":
		if arg <= 0 {
			return value{}, fmt.Errorf("log10 of non-positive number")
		}
		return floatVal(math.Log10(arg)), nil
	case "sin":
		return floatVal(math.Sin(arg)), nil
	case "cos":
		return floatVal(math.Cos(arg)), nil
	case "tan":
		return floatVal(math.Tan(arg)), nil
	case "pow":
		if len(call.Args) < 2 {
			return value{}, fmt.Errorf("pow requires 2 arguments")
		}
		e, err := eval(call.Args[1])
		if err != nil {
			return value{}, err
		}
		return floatVal(math.Pow(arg, e.toFloat())), nil
	}

	return value{}, fmt.Errorf("unknown function: %s", ident.Name)
}

func evalMinMax(name string, args []ast.Expr) (value, error) {
	vals := make([]value, len(args))
	allInt := true
	for i, a := range args {
		v, err := eval(a)
		if err != nil {
			return value{}, err
		}
		vals[i] = v
		if !v.isInt {
			allInt = false
		}
	}
	if allInt {
		best := vals[0].i
		for _, v := range vals[1:] {
			cmp := v.i.Cmp(best)
			if (name == "min" && cmp < 0) || (name == "max" && cmp > 0) {
				best = v.i
			}
		}
		return intVal(new(big.Int).Set(best)), nil
	}
	best := vals[0].toFloat()
	for _, v := range vals[1:] {
		f := v.toFloat()
		if name == "min" {
			best = math.Min(best, f)
		} else {
			best = math.Max(best, f)
		}
	}
	return floatVal(best), nil
}
