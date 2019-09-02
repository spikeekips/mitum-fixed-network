package main

import (
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	"github.com/knocknote/vitess-sqlparser/sqlparser"
	"golang.org/x/xerrors"
)

var puncs = strings.Join(
	func() []string {
		var es []string
		for _, s := range []string{"=", "<", ">", "<=", ">=", "!=", "<=>", "->", "->>"} {
			es = append(es, regexp.QuoteMeta(s))
		}

		return es
	}(),
	"|",
)
var connts = strings.Join(
	[]string{"in", "not in", "like", "not like", "regexp", "not regexp"},
	"|",
)

var re_column = regexp.MustCompile(
	fmt.Sprintf(`[\s]*([\w][\w\.]*)%s`,
		fmt.Sprintf(`([\s]*(%s)|[\s]+(%s))`, puncs, connts),
	),
)

var reflectNilKind reflect.Kind = reflect.UnsafePointer + 1

type ConditionParser struct {
}

func NewConditionParser() ConditionParser {
	return ConditionParser{}
}

func (cp ConditionParser) Parse(condition string) (Condition, error) {
	nc := re_column.ReplaceAllString(condition, "`$1`$2")

	stmt, err := sqlparser.Parse(fmt.Sprintf("select * from a where %s", nc))
	if err != nil {
		return nil, err
	}

	var where *sqlparser.Where
	_ = stmt.WalkSubtree(func(n sqlparser.SQLNode) (bool, error) {
		switch t := n.(type) {
		case *sqlparser.Where:
			where = t
			return false, xerrors.Errorf("found")
		}

		return true, nil
	})
	if where == nil {
		return nil, xerrors.Errorf("where not found")
	}

	return cp.parseSQLNode(where)
}

func (cp ConditionParser) parseSQLNode(expr sqlparser.SQLNode) (Condition, error) {
	switch t := expr.(type) {
	case *sqlparser.AndExpr:
		return cp.parseJointExpr("and", t)
	case *sqlparser.OrExpr:
		return cp.parseJointExpr("or", t)
	case *sqlparser.ComparisonExpr:
		return cp.parseComparisonExpr(t)
	}

	jc := NewJointConditions("")

	var conditions []Condition
	err := expr.WalkSubtree(func(n sqlparser.SQLNode) (bool, error) {
		cc, err := cp.parseSQLNode(n)
		if err != nil {
			return false, err
		}
		if cc == nil {
			return false, nil
		}

		conditions = append(conditions, cc)

		return false, nil
	})
	if err != nil {
		return nil, err
	}

	if len(conditions) < 1 {
		return nil, nil
	} else if len(conditions) == 1 {
		return conditions[0], nil
	}

	return jc.Add(NewJointConditions("", conditions...)), nil
}

func (cp ConditionParser) parseComparisonExpr(expr *sqlparser.ComparisonExpr) (Condition, error) {
	var colName string
	var val []interface{}
	var kind reflect.Kind

	err := expr.WalkSubtree(func(n sqlparser.SQLNode) (bool, error) {
		switch t := n.(type) {
		case *sqlparser.ColName:
			c, err := cp.parseColName(t)
			if err != nil {
				return false, err
			}
			colName = c
		case *sqlparser.SQLVal:
			v, k, err := cp.parseSQLVal(t)
			if err != nil {
				return false, err
			}
			val = append(val, v)
			kind = k
		case sqlparser.ValTuple:
			v, k, err := cp.parseValTuple(t)
			if err != nil {
				return false, err
			}
			val = v
			kind = k
		case *sqlparser.NullVal:
			kind = reflectNilKind
		}

		return false, nil
	})
	if err != nil {
		return nil, err
	}

	switch expr.Operator {
	case "regexp", "not regexp":
		re, err := regexp.Compile(val[0].(string))
		if err != nil {
			return nil, err
		}

		val = []interface{}{re}
	}

	return NewComparison(colName, expr.Operator, val, kind), nil
}

func (cp ConditionParser) parseJointExpr(joint string, expr sqlparser.SQLNode) (Condition, error) {
	var left, right sqlparser.Expr
	switch t := expr.(type) {
	case *sqlparser.AndExpr:
		left = t.Left
		right = t.Right
	case *sqlparser.OrExpr:
		left = t.Left
		right = t.Right
	default:
		return nil, xerrors.Errorf("AndExpr or OrExpr must be given: %T found ", t)
	}

	jc := NewJointConditions(joint)
	for _, n := range []sqlparser.SQLNode{left, right} {
		cc, err := cp.parseSQLNode(n)
		if err != nil {
			return nil, err
		}
		if cc == nil {
			continue
		}

		jc = jc.Add(cc)
	}

	return jc, nil
}

func (cp ConditionParser) parseColName(expr *sqlparser.ColName) (string, error) {
	colName := expr.Name.String()
	if len(colName) < 1 {
		return "", xerrors.Errorf("ColName not found")
	}

	return colName, nil
}

func (cp ConditionParser) parseSQLVal(expr *sqlparser.SQLVal) (interface{}, reflect.Kind, error) {
	var v interface{}
	var kind reflect.Kind

	var err error
	switch expr.Type {
	case sqlparser.StrVal:
		v = string(expr.Val)
		kind = reflect.String
	case sqlparser.IntVal:
		v, err = strconv.ParseInt(string(expr.Val), 10, 64)
		kind = reflect.Int64
	case sqlparser.FloatVal:
		v, err = strconv.ParseFloat(string(expr.Val), 64)
		kind = reflect.Float64
		//case sqlparser.HexNum:
		//case sqlparser.HexVal:
		//case sqlparser.ValArg:
	default:
		return nil, kind, xerrors.Errorf("unsupported value found; %v", string(expr.Val))
	}

	return v, kind, err
}

func (cp ConditionParser) parseValTuple(expr sqlparser.ValTuple) ([]interface{}, reflect.Kind, error) {
	var exprs sqlparser.Exprs
	_ = expr.WalkSubtree(func(n sqlparser.SQLNode) (bool, error) {
		switch t := n.(type) {
		case sqlparser.Exprs:
			exprs = t
			return false, xerrors.Errorf("found")
		}

		return false, nil
	})

	var kind reflect.Kind = reflect.Invalid
	var values []interface{}

	err := exprs.WalkSubtree(func(n sqlparser.SQLNode) (bool, error) {
		switch t := n.(type) {
		case *sqlparser.SQLVal:
			v, k, err := cp.parseSQLVal(t)
			if err != nil {
				return false, err
			}
			values = append(values, v)
			if kind == reflect.Invalid {
				kind = k
			} else if kind != k {
				return false, xerrors.Errorf("value tuple should have same type of values; %v != %v", kind, k)
			}

			return false, nil
		}

		return false, nil
	})
	if err != nil {
		return nil, kind, err
	} else if len(values) < 1 {
		return nil, kind, xerrors.Errorf("values found")
	}

	return values, kind, nil
}
