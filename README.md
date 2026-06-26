# Overview
Start a new browser instance and making a GET request to go.dev.

```go
bb := surf.NewBrowser()
err := bb.Open("http://go.dev")
if err != nil {
	panic(err)
}

// Outputs: "The Go Programming Language"
fmt.Println(bb.Title())
```

If you need to you can add additional request headers.

```go
bb := surf.NewBrowser()
bb.AddRequestHeader("Accept", "text/html")
bb.AddRequestHeader("Accept-Charset", "utf8")

err := bb.Open("http://go.dev")
if err != nil {
	panic(err)
}

fmt.Println(bb.Title())
```

Just like a real web browser, Surf maintains a history that you can move back through. You can also
bookmark pages and come back to them later.

```go
// Bookmark the page so we can come back to it later.
err = bb.Bookmark("go_dev")
if err != nil {
	panic(err)
}

// Now move back to the go.dev site.
bb.Back()

// And then back to go_dev using our bookmark.
bb.OpenBookmark("go_dev")
```
