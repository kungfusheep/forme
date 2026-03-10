package glyph

import (
	"cmp"
	"unsafe"
)

// Condition provides type-safe conditional rendering.
// T must be comparable; use IfOrd for ordered comparisons (Gt, Lt, etc).
type Condition[T comparable] struct {
	ptr *T
}

// If conditionally renders content based on a pointer value.
// Requires a pointer, compile-time enforced via generics.
//
//	If(&visible).Then(content)              // show when true
//	If(&mode).Eq("edit").Then(editor)       // show when equal
//	If(&count).Ne(0).Then(badge).Else(Text("empty"))
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

// Then is shorthand for checking truthiness (not equal to zero value).
// For bool: If(&flag).Then(node) renders when flag is true
// For int: If(&count).Then(node) renders when count is non-zero
// For string: If(&str).Then(node) renders when str is non-empty
func (c *Condition[T]) Then(node any) *ConditionEval[T] {
	var zero T
	return &ConditionEval[T]{
		ptr:  c.ptr,
		op:   condOpNe,
		val:  zero,
		then: node,
	}
}

// OrdCondition extends Condition with ordering comparisons (Gt, Lt, Gte, Lte).
// Use IfOrd instead of If when you need numeric/string range checks.
type OrdCondition[T cmp.Ordered] struct {
	ptr *T
}

// IfOrd conditionally renders content with ordering comparisons (Gt, Lt, Gte, Lte).
// T must satisfy cmp.Ordered (int, float64, string, etc).
//
//	IfOrd(&cpu).Gt(90.0).Then(Text("HOT").FG(Red))
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

// ConditionEval holds a prepared condition awaiting Then/Else branches.
type ConditionEval[T comparable] struct {
	ptr    *T
	offset uintptr // offset from element base (for ForEach)
	op     condOp
	val    T
	then   any
	els    any
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

func (e *ConditionEval[T]) compare(v T) bool {
	switch e.op {
	case condOpEq:
		return v == e.val
	case condOpNe:
		return v != e.val
	default:
		return false
	}
}

func (e *ConditionEval[T]) evaluate() bool { return e.compare(*e.ptr) }

func (e *ConditionEval[T]) getThen() any { return e.then }
func (e *ConditionEval[T]) getElse() any { return e.els }

func (e *ConditionEval[T]) setOffset(offset uintptr) { e.offset = offset }
func (e *ConditionEval[T]) getPtrAddr() uintptr      { return uintptr(unsafe.Pointer(e.ptr)) }

// evaluateWithBase evaluates the condition using an adjusted pointer (for ForEach)
func (e *ConditionEval[T]) evaluateWithBase(base unsafe.Pointer) bool {
	if e.offset == 0 {
		return e.evaluate()
	}
	return e.compare(*(*T)(unsafe.Add(base, e.offset)))
}

// OrdConditionEval holds a prepared ordered condition awaiting Then/Else branches.
type OrdConditionEval[T cmp.Ordered] struct {
	ptr    *T
	offset uintptr // offset from element base (for ForEach)
	op     condOp
	val    T
	then   any
	els    any
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

func (e *OrdConditionEval[T]) compare(v T) bool {
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

func (e *OrdConditionEval[T]) evaluate() bool { return e.compare(*e.ptr) }

func (e *OrdConditionEval[T]) getThen() any { return e.then }
func (e *OrdConditionEval[T]) getElse() any { return e.els }

func (e *OrdConditionEval[T]) setOffset(offset uintptr) { e.offset = offset }
func (e *OrdConditionEval[T]) getPtrAddr() uintptr      { return uintptr(unsafe.Pointer(e.ptr)) }

// evaluateWithBase evaluates the condition using an adjusted pointer (for ForEach)
func (e *OrdConditionEval[T]) evaluateWithBase(base unsafe.Pointer) bool {
	if e.offset == 0 {
		return e.evaluate()
	}
	return e.compare(*(*T)(unsafe.Add(base, e.offset)))
}

// conditionNode interface for the compiler to detect condition nodes
type conditionNode interface {
	evaluate() bool
	evaluateWithBase(base unsafe.Pointer) bool // for ForEach
	setOffset(offset uintptr)                  // set offset for ForEach
	getPtrAddr() uintptr                       // get pointer address for offset calculation
	getThen() any
	getElse() any
}

// ensure our types implement conditionNode
var _ conditionNode = (*ConditionEval[int])(nil)
var _ conditionNode = (*OrdConditionEval[int])(nil)

// SwitchBuilder constructs a type-safe multi-way branch.
// Use Switch(&ptr) to start, .Case() for branches, .Default() or .End() to finalise.
type SwitchBuilder[T comparable] struct {
	ptr   *T
	cases []switchCase[T]
	def   any
}

type switchCase[T comparable] struct {
	val  T
	node any
}

// Switch starts a multi-way branch. Type-safe via generics:
//
//	Switch(&state.Tab).
//	    Case("home", homeView).
//	    Case("settings", settingsView).
//	    Default(notFoundView)
func Switch[T comparable](ptr *T) *SwitchBuilder[T] {
	return &SwitchBuilder[T]{ptr: ptr}
}

// Case adds a branch for when *ptr == val
func (s *SwitchBuilder[T]) Case(val T, node any) *SwitchBuilder[T] {
	s.cases = append(s.cases, switchCase[T]{val: val, node: node})
	return s
}

// Default sets the fallback when no case matches.
// Returns SwitchNode which implements the compiler interface.
func (s *SwitchBuilder[T]) Default(node any) *SwitchNode[T] {
	s.def = node
	return &SwitchNode[T]{
		ptr:   s.ptr,
		cases: s.cases,
		def:   s.def,
	}
}

// End finalizes without a default (renders nothing if no match)
func (s *SwitchBuilder[T]) End() *SwitchNode[T] {
	return &SwitchNode[T]{
		ptr:   s.ptr,
		cases: s.cases,
		def:   nil,
	}
}

// SwitchNode is the compiled form of a Switch, ready for template evaluation.
type SwitchNode[T comparable] struct {
	ptr    *T
	offset uintptr // offset from ForEach element base; 0 = use ptr directly
	cases  []switchCase[T]
	def    any
}

// switchNodeInterface for the compiler to detect switch nodes
type switchNodeInterface interface {
	evaluateSwitch() any                      // runtime: returns matching node
	getCaseNodes() []any                      // compile-time: all case nodes
	getDefaultNode() any                      // compile-time: default node
	getPtrAddr() uintptr                      // compile-time: address of condition pointer
	getMatchIndex() int                       // runtime: matching case index, or -1 for default
	getMatchIndexWithBase(unsafe.Pointer) int // runtime: matching case index using ForEach element base
	setPtrOffset(uintptr)                     // compile-time: record offset from element base
}

func (s *SwitchNode[T]) evaluateSwitch() any {
	v := *s.ptr
	for _, c := range s.cases {
		if v == c.val {
			return c.node
		}
	}
	return s.def
}

func (s *SwitchNode[T]) getCaseNodes() []any {
	nodes := make([]any, len(s.cases))
	for i, c := range s.cases {
		nodes[i] = c.node
	}
	return nodes
}

func (s *SwitchNode[T]) getDefaultNode() any {
	return s.def
}

func (s *SwitchNode[T]) getMatchIndex() int {
	return s.getMatchIndexWithBase(nil)
}

func (s *SwitchNode[T]) getMatchIndexWithBase(elemBase unsafe.Pointer) int {
	var ptr *T
	if elemBase != nil && s.offset != 0 {
		ptr = (*T)(unsafe.Pointer(uintptr(elemBase) + s.offset))
	} else {
		ptr = s.ptr
	}
	v := *ptr
	for i, c := range s.cases {
		if v == c.val {
			return i
		}
	}
	return -1
}

func (s *SwitchNode[T]) getPtrAddr() uintptr {
	return uintptr(unsafe.Pointer(s.ptr))
}

func (s *SwitchNode[T]) setPtrOffset(off uintptr) {
	s.offset = off
}

var _ switchNodeInterface = (*SwitchNode[int])(nil)
