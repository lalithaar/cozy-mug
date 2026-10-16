# cozy-mug
_Finally, a full stack framework in Go that just works._

> [!NOTE]
> I am working on polishing the dev experience and documentation, will update the README once the vocabulary is decided. Please feel free to check out the source code in the meantime.

Go gives you excellent primitives and decides nothing. So every project
re-assembles the same wiring by hand: routing, layouts, data access, sessions,
an error page, a request log. **cozy-mug decides that 90% for you** — the
batteries-included feel of Rails or Django, without their weight.

```go
r.Get("/users/{id}", func(c *core.Context) core.Response {
	u, err := loadUser(c)          // a loader: plain Go, returns (value, error)
	if err != nil {
		return c.Error(err)        // 404, 500, ... rendered for you
	}
	return c.Render(http.StatusOK, "users/show", &UserView{User: u})
}).As("user")                      // named route -> reverse URLs in templates
```

```go
type Post struct {
	data.Model `db:"posts"`
	ID        int64     `db:"id auto"`
	Title     string    `db:"title"`
	AuthorID  int64     `db:"author_id index"`
	Author    *User     `db:"ref:author_id"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
	Version   int64     `db:"version"`
}
```

```go
posts, err := db.Table[Post]().
	With(P.Author).                    // relations load in batches - no N+1
	Where(P.AuthorID.Eq(uid)).
	OrderBy(P.CreatedAt.Desc()).
	Limit(10).
	All(ctx)
```
