# Conditionals & Dynamic Content

## If

For boolean conditions:

```go
showPanel := true

If(&showPanel).Then(VBox(
    Text("Panel visible"),
))
```

With else branch:

```go
isError := false

If(&isError).Then(
    Text("Error!").FG(Red),
).Else(
    Text("OK").FG(Green),
)
```

For value comparisons:

```go
count := 0

If(&count).Eq(0).Then(Text("Empty"))
If(&count).Ne(0).Then(Text("Has items"))
```

For ordered types (int, float, string):

```go
IfOrd(&count).Gt(10).Then(Text("Many"))
IfOrd(&count).Lt(5).Then(Text("Few"))
IfOrd(&count).Gte(0).Then(Text("Non-negative"))
IfOrd(&count).Lte(100).Then(Text("In range"))
```

## Switch

Multiple cases:

```go
mode := 0

Switch(&mode).
    Case(0, Text("Home")).
    Case(1, Text("Settings")).
    Case(2, Text("Help")).
    Default(Text("Unknown"))
```

Works with any comparable type:

```go
status := "loading"

Switch(&status).
    Case("loading", Spinner(&frame)).
    Case("ready", Text("Ready").FG(Green)).
    Case("error", Text("Error").FG(Red)).
    Default(Text("Unknown"))
```

## ForEach

Render a list:

```go
items := []string{"Apple", "Banana", "Cherry"}

ForEach(&items, func(item *string) any {
    return Text(item)
})
```

With complex items:

```go
type Item struct {
    Name  string
    Price float64
}

items := []Item{
    {Name: "Apple", Price: 1.50},
    {Name: "Banana", Price: 0.75},
}

ForEach(&items, func(item *Item) any {
    return HBox(
        Text(&item.Name),
        Space().Char('.'),
        Text(fmt.Sprintf("$%.2f", item.Price)),
    )
})
```

## Combining

Nested conditionals:

```go
VBox(
    If(&showHeader).Then(Text("Header")),

    Switch(&view).
        Case(0, VBox(
            ForEach(&items, func(item *Item) any {
                return If(&item.Visible).Then(Text(&item.Name))
            }),
        )).
        Case(1, Text("Settings")),

    If(&showFooter).Then(Text("Footer")),
)
```

## Rendering

Pointers are read at render time. Rendering happens automatically after input handlers complete:

```go
showDetails := false
items := []Item{...}

app.SetView(VBox(
    If(&showDetails).Then(Text("Details panel")),
    ForEach(&items, func(item *Item) any {
        return Text(&item.Name)
    }),
))

// Toggle in handler - re-render happens after handler completes
app.Handle("d", func(_ riffkey.Match) {
    showDetails = !showDetails
})

// Modify list - re-render happens after handler completes
app.Handle("a", func(_ riffkey.Match) {
    items = append(items, Item{Name: "New"})
})
```
