# TUI Gotchas & Learnings

Common pitfalls and their solutions, collected from real usage.

## SelectionList Render: Use Pointers to Struct Fields

When using `SelectionList` with a custom `Render` function, the returned `forme.Text`
should use a **pointer to a struct field**, not a computed string value.

### Won't work - text appears blank:

```go
ed.browserList = &forme.SelectionList{
    Items:    &ed.entries,
    Selected: &ed.selected,
    Render: func(entry *Entry) any {
        // BAD: computed string, not a pointer
        label := fmt.Sprintf("%s - %s", entry.Icon, entry.Name)
        return forme.Text{Content: label}
    },
}
```

### Works correctly:

```go
type Entry struct {
    Name         string
    DisplayLabel string  // pre-computed label stored in struct
}

// compute labels when building entries
for i := range entries {
    entries[i].DisplayLabel = fmt.Sprintf("+ %s", entries[i].Name)
}

ed.browserList = &forme.SelectionList{
    Items:    &ed.entries,
    Selected: &ed.selected,
    Render: func(entry *Entry) any {
        // GOOD: pointer to persistent struct field
        return forme.Text{Content: &entry.DisplayLabel}
    },
}
```

### Why this happens

The template system captures pointer offsets at compile time. When you pass a computed
string directly, there's no stable memory location for the framework to read from during
rendering. By storing the value in a struct field and passing a pointer, the reactive
binding works correctly.

This matches how the `Items` and `Selected` fields work - they're pointers to your data,
not copies of it.

### The pattern

If your list items need derived/formatted display text:
1. Add a field to your item struct (e.g., `DisplayLabel string`)
2. Compute and store the value when creating/updating items
3. Pass `&item.DisplayLabel` in your Render function
