Cji
===

pronounced `chi`, is an inline middleware chain for Goji web apps.

cji allows you to use middlewares for a single route without creating a new handler with a SubRouter.

For instance

```go
m := web.New()
m.Use(someMiddleware)
m.Get("/one", handlerOne)

admin := web.New()
m.Handle("/admin", admin)
admin.Use(middleware.SubRouter)
admin.Use(PasswordMiddleware)
admin.Use(HttpsOnlyMiddleware)
admin.Get("/", AdminRoot)
```

Becomes:

```go
m := web.New()
m.Use(someMiddleware)
m.Get("/one", handlerOne)
m.Get("/admin", cji.Use(PasswordMiddleware, HttpsOnlyMiddleware).On(AdminRoot))
```

We find this useful for middlewares that lookup objects in the database and handle authorization

```go
m.Get("/posts/:postId", cji.Use(PostContext).On(GetPost))

func PostContext(c *web.C, h http.Handler) http.Handler {
    fn := func(w http.ResponseWriter, r *http.Request) {
        user := c.Env["user"].(*data.User)
        postId = c.URLParams["postId"])
        post, err := //look up post in db, and make sure user has permissions
        if err != nil {
            w.WriteHeader(403)
            w.Write([]byte("Unauthorized"))
            return
        }
        c.Env["post"] = post
        h.ServeHTTP(w, r)
    }
    return http.HandlerFunc(fn)
}

func GetPost(c web.C, w http.ResponseWriter, r *http.Request) {
    post := c.Env["post"].(*data.Post)
    h.JSON(w, 200, post)
}
```

Authors: [@pkieltyka](https://github.com/pkieltyka) & [@mveytsman](https://github.com/mveytsman)
