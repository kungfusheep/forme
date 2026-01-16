package tui

// Three-phase layout system: Update → Layout → Draw
//
// Update (top→down): Distribute available widths to children
// Layout (bottom→up): Children calculate heights, parents position children
// Draw (top→down): Render with viewport culling
//
// Based on the validated TypeScript canvas library pattern.

// FlexNode represents a node in the layout tree.
type FlexNode struct {
	// Tree structure
	parent   *FlexNode
	children []*FlexNode
	level    int // depth in tree (root = 1)

	// Sizing
	percentWidth float32 // 0-1 range, 0 = use content/explicit width
	flexGrow     float32 // >0 = take share of remaining space
	explicitW    int16   // explicit width (0 = use calculated)
	explicitH    int16   // explicit height (0 = use calculated)
	minW, minH   int16   // minimum dimensions

	// Calculated geometry (set during layout phases)
	X, Y int16
	W, H int16

	// Layout configuration
	layout Layout    // nil = no layout (leaf node)
	gap    int8      // gap between children
	hPad   int8      // horizontal padding
	vPad   int8      // vertical padding
	border BorderStyle

	// Content (for leaf nodes)
	kind    FlexNodeKind
	content any // string, *string, []Span, etc.
	style   Style
}

type FlexNodeKind uint8

const (
	FlexNodeContainer FlexNodeKind = iota
	FlexNodeText
	FlexNodeRichText
	FlexNodeProgress
	FlexNodeMeter
	FlexNodeBar
	FlexNodeLeader
)

// Layout defines how a container positions its children.
type Layout interface {
	// DistributeWidths is called during Update phase (top→down).
	// Parent has W set, distribute to children based on percentWidth.
	DistributeWidths(node *FlexNode)

	// LayoutChildren is called during Layout phase (bottom→up).
	// Children already have their W and H set. Position them and calculate parent H.
	LayoutChildren(node *FlexNode)
}

// VerticalLayout stacks children vertically.
type VerticalLayout struct {
	Gap    int8
	HPad   int8 // horizontal padding
	TopPad int8
}

func (l VerticalLayout) DistributeWidths(node *FlexNode) {
	availableW := node.W - int16(l.HPad)*2
	availableH := node.H
	if node.border.TopLeft != 0 {
		availableW -= 2 // border takes 1 char each side
		availableH -= 2
	}

	for _, child := range node.children {
		// Distribute width
		if child.percentWidth > 0 {
			child.W = int16(float32(availableW) * child.percentWidth)
		} else if child.explicitW > 0 {
			child.W = child.explicitW
		} else {
			// In vertical layout, children default to full width
			child.W = availableW
		}
		// Enforce minimum width
		if child.W < child.minW {
			child.W = child.minW
		}

		// Apply explicit height if set
		// Flex children do NOT get height here - they'll be sized during Layout phase
		if child.explicitH > 0 {
			child.H = child.explicitH
		}
		// Otherwise, H stays 0 and will be calculated from content (bottom-up)
	}
}

func (l VerticalLayout) LayoutChildren(node *FlexNode) {
	// Child positions are RELATIVE to the content area (inside border/padding)
	// Border offset is applied during draw, not here
	x := int16(l.HPad)

	gap := l.Gap
	if gap == 0 {
		gap = node.gap
	}

	// Calculate available inner height
	availableH := node.H
	if node.border.TopLeft != 0 {
		availableH -= 2 // top and bottom border
	}

	// First pass: calculate total content height and total flex
	var contentH int16
	var totalFlex float32
	for i, child := range node.children {
		if child.flexGrow > 0 {
			totalFlex += child.flexGrow
			// Flex children start with their minimum/content height
			contentH += child.H
		} else {
			contentH += child.H
		}
		if i < len(node.children)-1 {
			contentH += int16(gap)
		}
	}

	// Distribute remaining space to flex children
	remaining := availableH - contentH - int16(l.TopPad)
	if remaining > 0 && totalFlex > 0 {
		for _, child := range node.children {
			if child.flexGrow > 0 {
				extra := int16(float32(remaining) * (child.flexGrow / totalFlex))
				child.H += extra
			}
		}
	}

	// Second pass: position children
	y := int16(l.TopPad)
	for i, child := range node.children {
		child.X = x
		child.Y = y
		y += child.H
		if i < len(node.children)-1 {
			y += int16(gap)
		}
	}

	// If parent height wasn't set, calculate from content
	if node.H == 0 {
		node.H = y
		if node.border.TopLeft != 0 {
			node.H += 2
		}
	}

	// Enforce minimum height
	if node.H < node.minH {
		node.H = node.minH
	}
}

// HorizontalLayout arranges children horizontally.
type HorizontalLayout struct {
	Gap    int8
	VPad   int8 // vertical padding
	LeftPad int8
}

func (l HorizontalLayout) DistributeWidths(node *FlexNode) {
	availableW := node.W - int16(l.LeftPad)*2
	availableH := node.H
	if node.border.TopLeft != 0 {
		availableW -= 2
		availableH -= 2
	}

	// Calculate total fixed width and total flex
	var fixedW int16
	var totalFlex float32
	gap := l.Gap
	if gap == 0 {
		gap = node.gap
	}

	for i, child := range node.children {
		if child.percentWidth > 0 {
			totalFlex += child.percentWidth
		} else if child.explicitW > 0 {
			fixedW += child.explicitW
		}
		if i > 0 {
			fixedW += int16(gap)
		}
	}

	remaining := availableW - fixedW

	// Distribute flex space
	for _, child := range node.children {
		if child.percentWidth > 0 {
			child.W = int16(float32(remaining) * (child.percentWidth / totalFlex))
		} else if child.explicitW > 0 {
			child.W = child.explicitW
		}
		if child.W < child.minW {
			child.W = child.minW
		}

		// Apply explicit height if set
		if child.explicitH > 0 {
			child.H = child.explicitH
		}
		// Otherwise, H stays 0 and will be calculated from content (bottom-up)
	}
}

func (l HorizontalLayout) LayoutChildren(node *FlexNode) {
	// Child positions are RELATIVE to the content area (inside border/padding)
	// Border offset is applied during draw, not here
	x := int16(l.LeftPad)
	y := int16(l.VPad)

	gap := l.Gap
	if gap == 0 {
		gap = node.gap
	}

	var maxH int16
	for i, child := range node.children {
		child.X = x
		child.Y = y
		x += child.W
		if i < len(node.children)-1 {
			x += int16(gap)
		}
		if child.H > maxH {
			maxH = child.H
		}
	}

	// Parent height = max child height + padding + border
	node.H = maxH + int16(l.VPad)*2
	if node.border.TopLeft != 0 {
		node.H += 2
	}
	if node.H < node.minH {
		node.H = node.minH
	}
}

// FlexTree manages the three-phase layout process.
type FlexTree struct {
	root     *FlexNode
	byLevel  [][]*FlexNode // nodes grouped by level for efficient traversal
	maxLevel int
}

// NewFlexTree creates a new layout tree with the given root.
func NewFlexTree(root *FlexNode) *FlexTree {
	t := &FlexTree{
		root:    root,
		byLevel: make([][]*FlexNode, 32),
	}

	// Initialize level buckets
	for i := range t.byLevel {
		t.byLevel[i] = make([]*FlexNode, 0, 8)
	}

	// Index nodes by level
	t.indexNode(root, 1)

	// Trim unused levels
	for t.maxLevel > 0 && len(t.byLevel[t.maxLevel]) == 0 {
		t.maxLevel--
	}

	return t
}

func (t *FlexTree) indexNode(node *FlexNode, level int) {
	node.level = level

	if level >= len(t.byLevel) {
		// Extend if needed
		newLevels := make([][]*FlexNode, level+16)
		copy(newLevels, t.byLevel)
		for i := len(t.byLevel); i < len(newLevels); i++ {
			newLevels[i] = make([]*FlexNode, 0, 8)
		}
		t.byLevel = newLevels
	}

	t.byLevel[level] = append(t.byLevel[level], node)
	if level > t.maxLevel {
		t.maxLevel = level
	}

	for _, child := range node.children {
		child.parent = node
		t.indexNode(child, level+1)
	}
}

// Execute runs all three phases and renders to the buffer.
func (t *FlexTree) Execute(buf *Buffer, w, h int16) {
	if t.root == nil {
		return
	}

	// Set root dimensions
	t.root.W = w
	t.root.H = h

	// Phase 1: Update (top→down) - distribute widths
	for level := 1; level <= t.maxLevel; level++ {
		for _, node := range t.byLevel[level] {
			if node.layout != nil {
				node.layout.DistributeWidths(node)
			}
		}
	}

	// Phase 2: Layout (bottom→up) - calculate heights and positions
	// Process deepest levels first
	for level := t.maxLevel; level >= 1; level-- {
		for _, node := range t.byLevel[level] {
			// Leaf nodes calculate their own height
			if node.layout == nil {
				t.measureLeaf(node)
			} else {
				// Containers: children already have W and H
				node.layout.LayoutChildren(node)
			}
		}
	}

	// Phase 3: Draw (top→down) - render with absolute positions
	t.draw(buf, t.root, 0, 0, w, h)
}

func (t *FlexTree) measureLeaf(node *FlexNode) {
	switch node.kind {
	case FlexNodeText:
		// Height is 1 for single-line text
		node.H = 1
		// Width from content if not set
		if node.W == 0 {
			switch v := node.content.(type) {
			case string:
				node.W = int16(len(v))
			case *string:
				node.W = int16(len(*v))
			}
		}

	case FlexNodeRichText:
		node.H = 1
		if node.W == 0 {
			if spans, ok := node.content.([]Span); ok {
				w := 0
				for _, s := range spans {
					w += len(s.Text)
				}
				node.W = int16(w)
			}
		}

	case FlexNodeProgress, FlexNodeMeter, FlexNodeBar:
		node.H = 1
		if node.W == 0 {
			node.W = 20 // default width
		}

	case FlexNodeLeader:
		node.H = 1
		// Width should be set explicitly or from parent
	}
}

func (t *FlexTree) draw(buf *Buffer, node *FlexNode, absX, absY, clipW, clipH int16) {
	// Calculate absolute position
	x := absX + node.X
	y := absY + node.Y

	// Viewport culling
	if x >= clipW || y >= clipH || x+node.W < 0 || y+node.H < 0 {
		return
	}

	// Draw border if present
	if node.border.TopLeft != 0 {
		buf.DrawBorder(int(x), int(y), int(node.W), int(node.H), node.border, node.style)
		// Draw title in top border if present
		if title, ok := node.content.(string); ok && title != "" {
			titleStr := "─ " + title + " "
			buf.WriteString(int(x)+1, int(y), titleStr, node.style)
		}
	}

	// Draw content for leaf nodes
	switch node.kind {
	case FlexNodeText:
		innerX := x
		innerY := y
		if node.border.TopLeft != 0 {
			innerX++
			innerY++
		}
		switch v := node.content.(type) {
		case string:
			buf.WriteString(int(innerX), int(innerY), v, node.style)
		case *string:
			buf.WriteString(int(innerX), int(innerY), *v, node.style)
		}

	case FlexNodeRichText:
		innerX := x
		innerY := y
		if node.border.TopLeft != 0 {
			innerX++
			innerY++
		}
		if spans, ok := node.content.([]Span); ok {
			buf.WriteSpans(int(innerX), int(innerY), spans, int(clipW))
		}

	case FlexNodeProgress:
		innerX := x
		innerY := y
		if node.border.TopLeft != 0 {
			innerX++
			innerY++
		}
		if ratio, ok := node.content.(float32); ok {
			buf.WriteProgressBar(int(innerX), int(innerY), int(node.W), ratio, node.style)
		}

	case FlexNodeMeter:
		innerX := x
		innerY := y
		if node.border.TopLeft != 0 {
			innerX++
			innerY++
		}
		if vals, ok := node.content.([2]int); ok {
			meterStr := Meter(vals[0], vals[1], int(node.W))
			buf.WriteString(int(innerX), int(innerY), meterStr, node.style)
		}

	case FlexNodeBar:
		innerX := x
		innerY := y
		if node.border.TopLeft != 0 {
			innerX++
			innerY++
		}
		if vals, ok := node.content.([2]int); ok {
			barStr := Bar(vals[0], vals[1])
			buf.WriteString(int(innerX), int(innerY), barStr, node.style)
		}

	case FlexNodeLeader:
		innerX := x
		innerY := y
		if node.border.TopLeft != 0 {
			innerX++
			innerY++
		}
		if parts, ok := node.content.([2]string); ok {
			leaderStr := LeaderStr(parts[0], parts[1], int(node.W))
			buf.WriteString(int(innerX), int(innerY), leaderStr, node.style)
		}
	}

	// Recurse to children
	childAbsX := x
	childAbsY := y
	if node.border.TopLeft != 0 {
		childAbsX++
		childAbsY++
	}

	for _, child := range node.children {
		t.draw(buf, child, childAbsX, childAbsY, clipW, clipH)
	}
}

// Builder helpers for creating flex nodes

// FCol creates a vertical container.
func FCol(children ...*FlexNode) *FlexNode {
	return &FlexNode{
		kind:     FlexNodeContainer,
		layout:   VerticalLayout{},
		children: children,
	}
}

// FRow creates a horizontal container.
func FRow(children ...*FlexNode) *FlexNode {
	return &FlexNode{
		kind:     FlexNodeContainer,
		layout:   HorizontalLayout{},
		children: children,
	}
}

// FText creates a text node.
func FText(content any) *FlexNode {
	return &FlexNode{
		kind:    FlexNodeText,
		content: content,
	}
}

// FRich creates a rich text node.
func FRich(spans []Span) *FlexNode {
	return &FlexNode{
		kind:    FlexNodeRichText,
		content: spans,
	}
}

// FMeter creates a meter display node.
func FMeter(value, max int) *FlexNode {
	return &FlexNode{
		kind:    FlexNodeMeter,
		content: [2]int{value, max},
	}
}

// FBar creates a bar display node.
func FBar(filled, total int) *FlexNode {
	return &FlexNode{
		kind:    FlexNodeBar,
		content: [2]int{filled, total},
	}
}

// FLeader creates a leader (label...value) node.
func FLeader(label, value string) *FlexNode {
	return &FlexNode{
		kind:    FlexNodeLeader,
		content: [2]string{label, value},
	}
}

// FPanel creates a bordered panel with a title.
// Title appears in the top border: ┌─ TITLE ─────┐
func FPanel(title string, children ...*FlexNode) *FlexNode {
	return &FlexNode{
		kind:     FlexNodeContainer,
		layout:   VerticalLayout{},
		children: children,
		border:   BorderSingle,
		content:  title, // Store title for rendering
	}
}

// FLED creates an LED indicator node.
func FLED(on bool) *FlexNode {
	var text string
	if on {
		text = "●"
	} else {
		text = "○"
	}
	return &FlexNode{
		kind:    FlexNodeText,
		content: text,
	}
}

// FLEDs creates multiple LED indicators.
func FLEDs(states ...bool) *FlexNode {
	return &FlexNode{
		kind:    FlexNodeText,
		content: LEDs(states...),
	}
}

// Chainable modifiers

func (n *FlexNode) Gap(g int8) *FlexNode {
	n.gap = g
	return n
}

func (n *FlexNode) Pad(h, v int8) *FlexNode {
	n.hPad = h
	n.vPad = v
	return n
}

func (n *FlexNode) Border(b BorderStyle) *FlexNode {
	n.border = b
	return n
}

func (n *FlexNode) Width(w int16) *FlexNode {
	n.explicitW = w
	return n
}

func (n *FlexNode) Height(h int16) *FlexNode {
	n.explicitH = h
	return n
}

func (n *FlexNode) MinWidth(w int16) *FlexNode {
	n.minW = w
	return n
}

func (n *FlexNode) MinHeight(h int16) *FlexNode {
	n.minH = h
	return n
}

func (n *FlexNode) Percent(p float32) *FlexNode {
	n.percentWidth = p
	return n
}

func (n *FlexNode) Grow(factor float32) *FlexNode {
	n.flexGrow = factor
	return n
}

func (n *FlexNode) Style(s Style) *FlexNode {
	n.style = s
	return n
}

func (n *FlexNode) Bold() *FlexNode {
	n.style.Attr = n.style.Attr.With(AttrBold)
	return n
}
