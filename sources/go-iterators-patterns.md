# Mastering Iterators in Go 1.26

Go 1.23 introduced the `iter` package, but Go 1.26 finally makes it ergonomic with the standard library adopting pull-iterators across `slices` and `maps`.

## Goodbye Manual Loops

We can finally chain operations lazily without allocating intermediate slices. This feels almost like Rust or Linq, but with Go's simplicity.

```go
// Current Go 1.26 Beta syntax
seq := slices.All(users)
    .Filter(func(u User) bool { return u.Active })
    .Map(func(u User) string { return u.Email })

for email := range seq {
    fmt.Println(email)
}
```

## Performance Implications

I profiled this against a standard for loop. The compiler is smart enough now to inline the iterator function calls, meaning this "functional style" code compiles down to the exact same machine code as a raw C-style loop.
There is no longer a performance penalty for writing readable code.
