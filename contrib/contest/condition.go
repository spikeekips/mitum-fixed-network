package main

import (
	"fmt"
	"strconv"

	"github.com/knocknote/vitess-sqlparser/sqlparser"
	"golang.org/x/xerrors"
)

type ConditionParser struct {
	condition string
}

func NewConditionParser(condition string) *ConditionParser {
	return &ConditionParser{condition: condition}
}

func (cp *ConditionParser) Parse() (ConditionChecker, error) {
	stmt, err := sqlparser.Parse(fmt.Sprintf("select * from a where %s", cp.condition))
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

func (cp *ConditionParser) parseSQLNode(expr sqlparser.SQLNode) (ConditionChecker, error) {
	switch t := expr.(type) {
	case *sqlparser.AndExpr:
		return cp.parseJointExpr("and", t)
	case *sqlparser.OrExpr:
		return cp.parseJointExpr("or", t)
	case *sqlparser.ComparisonExpr:
		return cp.parseComparisonExpr(t)
	}

	jc := NewJointConditions("")

	var checkers []ConditionChecker
	err := expr.WalkSubtree(func(n sqlparser.SQLNode) (bool, error) {
		cc, err := cp.parseSQLNode(n)
		if err != nil {
			return false, err
		}
		if cc == nil {
			return false, nil
		}

		checkers = append(checkers, cc)

		return false, nil
	})
	if err != nil {
		return nil, err
	}

	if len(checkers) < 1 {
		return nil, nil
	} else if len(checkers) == 1 {
		return checkers[0], nil
	}

	return jc.Add(NewJointConditions("", checkers...)), nil
}

func (cp *ConditionParser) parseComparisonExpr(expr *sqlparser.ComparisonExpr) (ConditionChecker, error) {
	var colName string
	var val []interface{}

	err := expr.WalkSubtree(func(n sqlparser.SQLNode) (bool, error) {
		switch t := n.(type) {
		case *sqlparser.ColName:
			c, err := cp.parseColName(t)
			if err != nil {
				return false, err
			}
			colName = c
		case *sqlparser.SQLVal:
			v, err := cp.parseSQLVal(t)
			if err != nil {
				return false, err
			}
			val = append(val, v)
		case sqlparser.ValTuple:
			v, err := cp.parseValTuple(t)
			if err != nil {
				return false, err
			}
			val = v
		}

		return false, nil
	})
	if err != nil {
		return nil, err
	}

	return NewComparison(colName, expr.Operator, val), nil
}

func (cp *ConditionParser) parseJointExpr(joint string, expr sqlparser.SQLNode) (ConditionChecker, error) {
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

func (cp *ConditionParser) parseColName(expr *sqlparser.ColName) (string, error) {
	var colName string
	_ = expr.WalkSubtree(func(n sqlparser.SQLNode) (bool, error) {
		switch t := n.(type) {
		case sqlparser.ColIdent:
			colName = t.String()
			return false, xerrors.Errorf("found")
		}

		return false, nil
	})
	if len(colName) < 1 {
		return "", xerrors.Errorf("ColName not found")
	}

	return colName, nil
}

func (cp *ConditionParser) parseSQLVal(expr *sqlparser.SQLVal) (interface{}, error) {
	var v interface{}
	var err error
	switch expr.Type {
	case sqlparser.StrVal:
		v = string(expr.Val)
	case sqlparser.IntVal:
		v, err = strconv.ParseInt(string(expr.Val), 10, 64)
	case sqlparser.FloatVal:
		v, err = strconv.ParseFloat(string(expr.Val), 64)
		//case sqlparser.HexNum:
		//case sqlparser.HexVal:
		//case sqlparser.ValArg:
	default:
		return nil, xerrors.Errorf("unsupported value found; %v", string(expr.Val))
	}

	return v, err
}

func (cp *ConditionParser) parseValTuple(expr sqlparser.ValTuple) ([]interface{}, error) {
	var exprs sqlparser.Exprs
	_ = expr.WalkSubtree(func(n sqlparser.SQLNode) (bool, error) {
		switch t := n.(type) {
		case sqlparser.Exprs:
			exprs = t
			return false, xerrors.Errorf("found")
		}

		return false, nil
	})

	var values []interface{}
	err := exprs.WalkSubtree(func(n sqlparser.SQLNode) (bool, error) {
		switch t := n.(type) {
		case *sqlparser.SQLVal:
			v, err := cp.parseSQLVal(t)
			if err != nil {
				return false, err
			}
			values = append(values, v)

			return false, nil
		}

		return false, nil
	})
	if err != nil {
		return nil, err
	} else if len(values) < 1 {
		return nil, xerrors.Errorf("values found")
	}

	return values, nil
}
