package tui

import "cmp"

// Condition builder for type-safe conditionals.
// The generic *T parameter enforces pointer-passing at compile time.
type Condition[T comparable] struct {
	ptr *T
}

// If starts a conditional chain. Compile-time enforces pointer:
//
//	If(&state.Count).Eq(0)    // works
//	If(state.Count).Eq(0)     // compile error: int is not *int
func If[T comparable](ptr *T) *Condition[T] {
	return &Condition[T]{ptr: ptr}
}

// Eq checks equality: *ptr == val
func (c *Condition[T]) Eq(val T) *ConditionEval[T] {
	return &ConditionEval[T]{
		ptr: c.ptr,
		op:  condOpEq,
		val: val,
	}
}

// Ne checks inequality: *ptr != val
func (c *Condition[T]) Ne(val T) *ConditionEval[T] {
	return &ConditionEval[T]{
		ptr: c.ptr,
		op:  condOpNe,
		val: val,
	}
}

// OrdCondition extends Condition for ordered types (int, float, string).
type OrdCondition[T cmp.Ordered] struct {
	ptr *T
}

// IfOrd starts a conditional chain for ordered types (supports Gt, Lt, etc).
func IfOrd[T cmp.Ordered](ptr *T) *OrdCondition[T] {
	return &OrdCondition[T]{ptr: ptr}
}

// Eq checks equality
func (c *OrdCondition[T]) Eq(val T) *OrdConditionEval[T] {
	return &OrdConditionEval[T]{ptr: c.ptr, op: condOpEq, val: val}
}

// Ne checks inequality
func (c *OrdCondition[T]) Ne(val T) *OrdConditionEval[T] {
	return &OrdConditionEval[T]{ptr: c.ptr, op: condOpNe, val: val}
}

// Gt checks greater than: *ptr > val
func (c *OrdCondition[T]) Gt(val T) *OrdConditionEval[T] {
	return &OrdConditionEval[T]{ptr: c.ptr, op: condOpGt, val: val}
}

// Lt checks less than: *ptr < val
func (c *OrdCondition[T]) Lt(val T) *OrdConditionEval[T] {
	return &OrdConditionEval[T]{ptr: c.ptr, op: condOpLt, val: val}
}

// Gte checks greater than or equal: *ptr >= val
func (c *OrdCondition[T]) Gte(val T) *OrdConditionEval[T] {
	return &OrdConditionEval[T]{ptr: c.ptr, op: condOpGte, val: val}
}

// Lte checks less than or equal: *ptr <= val
func (c *OrdCondition[T]) Lte(val T) *OrdConditionEval[T] {
	return &OrdConditionEval[T]{ptr: c.ptr, op: condOpLte, val: val}
}

type condOp int

const (
	condOpEq condOp = iota
	condOpNe
	condOpGt
	condOpLt
	condOpGte
	condOpLte
)

// ConditionEval holds a comparable condition ready for Then/Else
type ConditionEval[T comparable] struct {
	ptr  *T
	op   condOp
	val  T
	then any
	els  any
}

// Then specifies what to render when true
func (e *ConditionEval[T]) Then(node any) *ConditionEval[T] {
	e.then = node
	return e
}

// Else specifies what to render when false
func (e *ConditionEval[T]) Else(node any) *ConditionEval[T] {
	e.els = node
	return e
}

// evaluate checks the condition at runtime
func (e *ConditionEval[T]) evaluate() bool {
	v := *e.ptr
	switch e.op {
	case condOpEq:
		return v == e.val
	case condOpNe:
		return v != e.val
	default:
		return false
	}
}

func (e *ConditionEval[T]) getThen() any { return e.then }
func (e *ConditionEval[T]) getElse() any { return e.els }

// OrdConditionEval holds an ordered condition ready for Then/Else
type OrdConditionEval[T cmp.Ordered] struct {
	ptr  *T
	op   condOp
	val  T
	then any
	els  any
}

// Then specifies what to render when true
func (e *OrdConditionEval[T]) Then(node any) *OrdConditionEval[T] {
	e.then = node
	return e
}

// Else specifies what to render when false
func (e *OrdConditionEval[T]) Else(node any) *OrdConditionEval[T] {
	e.els = node
	return e
}

// evaluate checks the condition at runtime
func (e *OrdConditionEval[T]) evaluate() bool {
	v := *e.ptr
	switch e.op {
	case condOpEq:
		return v == e.val
	case condOpNe:
		return v != e.val
	case condOpGt:
		return v > e.val
	case condOpLt:
		return v < e.val
	case condOpGte:
		return v >= e.val
	case condOpLte:
		return v <= e.val
	default:
		return false
	}
}

func (e *OrdConditionEval[T]) getThen() any { return e.then }
func (e *OrdConditionEval[T]) getElse() any { return e.els }

// ConditionNode interface for the compiler to detect condition nodes
type ConditionNode interface {
	evaluate() bool
	getThen() any
	getElse() any
}

// Ensure our types implement ConditionNode
var _ ConditionNode = (*ConditionEval[int])(nil)
var _ ConditionNode = (*OrdConditionEval[int])(nil)
