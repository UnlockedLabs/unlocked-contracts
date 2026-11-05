# unlocked-contracts

Shared Go types and contracts for `unlocked-core` and every UnlockEd feature.

Features depend on this small, versioned module instead of on all of core.
It holds plain types and constants, plus small clients for asking core
things, with no dependencies outside the standard library.

| Package | What it holds |
|---|---|
| `coreapi` | What a product service asks core: `coreapi.Users` and `coreapi.LiveProviders`, their types, an HTTP client for each and `coreapi.NewHandler` (core's side) |
| `features` | Feature-flag names (`features.OpenContent`, …), `features.All`, `features.Register`, which adds a product's own feature, and `features.Checker`, which asks core whether a feature is on at a facility |
| `identity` | `identity.Claims`, who is logged in, how to read them from a request's context, `identity.SessionChecker`, which checks the login for a product service, and `identity.RequireServiceToken`, which lets only callers with the service token through |
| `jobs` | Job names (`jobs.GetCourses`, …) and each job's NATS subject (`PubName()`) |
| `opencontent` | What core asks Open Content: the `OpenContent` interface and its types, `opencontent.Client` (core's side) and `opencontent.NewHandler` (Open Content's side), and the `Notification` Open Content sends to core's websocket |
| `programs` | What core asks Programs: the `Programs` interface and its types, `programs.Client` (core's side) and `programs.NewHandler` (Programs' side) |
| `providerplatforms` | What core asks Provider Platforms: the `ProviderPlatforms` interface and its types, `providerplatforms.Client` (core's side) and `providerplatforms.NewHandler` (Provider Platforms' side) |
| `users` | User roles (`users.SystemAdmin`, …) and `users.AdminRoles` |

Every value here is stored in a database, sent over NATS or kept in Kratos
identity traits. Changing one breaks running sites, so the tests pin each one.

## Use it

```bash
go get github.com/UnlockedLabs/unlocked-contracts@v0.5.0
```

```go
import "github.com/UnlockedLabs/unlocked-contracts/jobs"

subject := jobs.GetCourses.PubName() // "tasks.get_courses"
```

### Who is logged in

Core checks the login session and puts the user's `identity.Claims` on every
request it hands a module:

```go
import "github.com/UnlockedLabs/unlocked-contracts/identity"

claims, ok := identity.FromContext(r.Context())
if !ok {
	// no logged-in user: core only sends authenticated requests, so this is a bug
}
if claims.IsAdmin() {
	// …
}
```

### Check the login in a product service

A product in its own service gets its requests straight from the ingress, not
through core, so it checks the login itself. Make one `SessionChecker` at
startup and wrap every route that needs a logged-in user:

```go
import "github.com/UnlockedLabs/unlocked-contracts/identity"

sessions := identity.NewSessionChecker("http://kratos:4433", "http://server:8080", nil)
mux.Handle("GET /api/things", sessions.Require(listThings))

// in listThings: the same Claims core would build for this user
claims, _ := identity.FromContext(r.Context())
```

It checks what core checks, in the same order:
- **The login:** Kratos's `whoami` must say the `ory_kratos_session` cookie's
  session is active, and core's `GET /internal/v1/claims?kratos_id=…` must
  know the user and not have deactivated them. Otherwise: 401.
- **A password reset:** a user whose `password_reset` trait isn't `false`
  must reset it in core first: 403.
- **CSRF:** any request other than GET, HEAD or OPTIONS must send the
  `csrf_token` cookie's value in the `X-CSRF-Token` header: 403. Like core,
  it gives a logged-in browser that has no `csrf_token` cookie a new one, so
  the browser's next write works.

Errors use core's JSON shape, `{"message": "...", "data": null}`. A request
turned away as not logged in is logged with `slog` at Info, with the reason.

- **It needs a core with `/internal/v1/claims`** (EN-256). Against an older
  core, every request gets a 503 and an Error log, not a logout.
- **It fails closed.** If Kratos or core can't answer, the request gets a 503
  (logged with `slog` at Error, unless the browser had already hung up),
  never a pass. Unlike core, it doesn't clear the browser's cookies, so a
  blip doesn't log anyone out.
- **Nothing is cached:** each request asks Kratos and core, as core asks
  Kratos and its database, so a logout or a deactivation applies to the next
  request. Each call times out after 2 seconds.

### Check a feature

Only core knows the rules (statewide defaults, facility overrides, sub-features
off when their parent is off), so a module asks it. Make one `Checker` at
startup and share it:

```go
import "github.com/UnlockedLabs/unlocked-contracts/features"

checker := features.NewChecker("http://server:8080", nil) // core, inside the cluster

on, err := checker.Enabled(ctx, facilityID, features.OpenContent)
if err != nil {
	log.Warn("feature check: core didn't answer", "err", err) // log it, don't fail on it
}
if !on {
	// the feature is off at this facility
}
```

- It calls core's `GET /internal/v1/features?facility_id=N`, which is only
  reachable inside the cluster.
- Each facility's answer is reused for 30 seconds (`features.CacheTTL`), so a
  Feature Control change reaches every module within 30 seconds.
- **It fails safe.** If core can't answer, the last answer is kept, however
  old; with none, every feature reads as off. An error from `Enabled` only
  means core didn't answer just now, and the bool is still safe to act on.
  After a failed call it waits 30 seconds before asking again.
- Only one call to core per facility runs at a time; other requests for that
  facility wait for its answer. A request whose context is canceled stops
  waiting, but the call goes on for everyone else (2-second timeout).

### Add a feature

A product gets its own on/off switch in Feature Control by registering it at
startup. No core change, migration or frontend release is needed.

1. **Pick a name, a title and a description.** The name is permanent and must
   not be used by any other product: lowercase letters, digits and
   underscores, starting with a letter, up to 64 characters. The title (up to
   100 characters) and description (up to 500) are what Feature Control shows,
   and can change later.
2. **Declare the name** in your product:

   ```go
   const MyTool features.Access = "my_tool"
   ```

3. **Register it at startup, before serving, and fail startup if it errors:**

   ```go
   err := features.Register(ctx, "http://server:8080", nil, features.Feature{
   	Name:        MyTool,
   	Title:       "My Tool",
   	Description: "What My Tool does for residents",
   })
   if err != nil {
   	log.Fatal(err) // Kubernetes restarts the product until core answers
   }
   ```

4. **Gate the product** with a `Checker` (see [Check a feature](#check-a-feature)):
   `checker.Enabled(ctx, facilityID, MyTool)`.
5. **Deploy.** The feature shows up in Feature Control, off at every facility.
6. **Turn it on** per facility in Feature Control.

- `Register` calls core's `PUT /internal/v1/features/{name}`, which is only
  reachable inside the cluster. It makes one attempt, with a 2-second timeout.
- Registering again on every restart is safe: it only updates the title and
  description. It never turns the feature on or changes any facility.
- A failed registration can't turn anything on: until it succeeds, the feature
  isn't in core's list, so every check reads it as off.
- A bad name, a missing title or a description that's too long is an error
  carrying core's message. An empty name is an error before core is asked.
- A 404 means core is older than EN-257 and doesn't have the endpoint yet.
- Core's own features (`features.All`) can't be registered, so a product can't
  rename them: core answers 409.
- Only top-level features can be registered, not sub-features.
- With one attempt per start, a product that starts before core (after a
  whole site restarts) is restarted by Kubernetes with growing delays, up to
  5 minutes apart, until core answers.
- **To remove a feature** (rare), delete its row; its facility settings go with
  it: `DELETE FROM feature_flags WHERE name = 'my_tool';`. Stop registering it
  first, or the next restart adds it back.

### Ask Open Content

Open Content runs as its own service (EN-205), so core asks it questions over
HTTP. Both sides come from this package, so they always agree:

```go
import "github.com/UnlockedLabs/unlocked-contracts/opencontent"

// core: one Client at startup, used wherever it needs an opencontent.OpenContent
oc := opencontent.NewClient("http://open-content:8080", os.Getenv("OPEN_CONTENT_TOKEN"), nil)
libraries, err := oc.TopLibraries(ctx, userID, facilityID)

// Open Content: serve its answers (anything that implements OpenContent)
mux.Handle("/internal/v1/open-content/", opencontent.NewHandler(answers, os.Getenv("OPEN_CONTENT_TOKEN")))
```

- Each question is a `POST /internal/v1/open-content/<question>` with its
  arguments as JSON, answered in core's `{"message", "data"}` shape. The
  ingress never routes `/internal`, so only the cluster can reach it.
- Only core may ask: the Client sends the site's secret as a bearer token,
  and the handler answers a missing or wrong one, or every request if it has
  no token, with a 401. Core and Open Content get the same secret, such as
  `OPEN_CONTENT_TOKEN`. The demo reset deletes activity at every facility, so
  nothing else in the cluster should be able to call it.
- Any answer but a 200 is an error, with Open Content's message. Arguments
  that aren't valid JSON get a 400; an error from Open Content a 500, logged
  with `slog`.
- Each question times out after 10 seconds, the demo seed and reset after 5
  minutes, and sooner if the caller's context ends.
- `EndActivity` returns nothing: it logs a failure instead.

To send a message to a user's open browser tabs, Open Content publishes an
`opencontent.Notification` as JSON on `opencontent.NotifySubject`, and every
core server forwards it over its websocket. It's plain NATS, so a message no
server is listening for is dropped, like one for a user with no open tab.

### Ask Programs

Programs runs as its own service (EN-207), so core asks it questions over
HTTP, the same way it asks Open Content:

```go
import "github.com/UnlockedLabs/unlocked-contracts/programs"

// core: one Client at startup, used wherever it needs a programs.Programs
p := programs.NewClient("http://programs:8080", token, nil)
ids, err := p.EnrolledUserIDs(ctx, classID)

// Programs: serve its answers (anything that implements Programs)
mux.Handle("/internal/v1/programs/", programs.NewHandler(answers, token))
```

- Each question is a `POST /internal/v1/programs/<question>` with its
  arguments as JSON, answered in core's `{"message", "data"}` shape. The
  ingress never routes `/internal`, so only the cluster can reach it.
- Every call carries the shared service token as a bearer token. A missing or
  wrong token gets a 401; a handler given an empty token rejects everything.
- Any answer but a 200 is an error, with Programs' message. Arguments that
  aren't valid JSON get a 400; an error from Programs a 500, logged with
  `slog`.
- Each question times out after 10 seconds, and sooner if the caller's context
  ends.
- `EndEnrollmentsForTransfer` and `WithdrawEnrollments` change nothing when
  called again, so core can retry them.

### Ask Provider Platforms, and let it ask core

Provider Platforms runs as its own service (EN-206), so core asks it questions
over HTTP, and it asks core for user accounts and Programs' live LMS data. Each
side comes from one package, so they always agree:

```go
import (
	"github.com/UnlockedLabs/unlocked-contracts/coreapi"
	"github.com/UnlockedLabs/unlocked-contracts/providerplatforms"
)

// core: one Client, used wherever it needs a providerplatforms.ProviderPlatforms
pp := providerplatforms.NewClient("http://provider-platforms:8080", token, nil)
mappings, err := pp.UserMappings(ctx, userID)

// Provider Platforms: serve its answers
mux.Handle("/internal/v1/provider-platforms/", providerplatforms.NewHandler(answers, token))

// Provider Platforms: ask core
users := coreapi.NewUsersClient("http://server:8080", token, nil)
live := coreapi.NewLiveProvidersClient("http://server:8080", token, nil)

// core: serve its answers
mux.Handle("/internal/v1/core/", coreapi.NewHandler(coreUsers, corePrograms, token))
```

Core's proxy sends the service token as
`Authorization: Bearer <token>` and the caller's claims in the
`identity.ClaimsHeader` (`X-Identity-Claims`) header. Core writes it with
`identity.SetClaimsHeader(req, claims)`, and the service reads it with
`identity.ClaimsFromRequest(req)`, which returns false when it is missing or
not JSON.

- Each question is a `POST /internal/v1/provider-platforms/<question>` (or
  `/internal/v1/core/users/<method>`, `/internal/v1/core/live-providers/<method>`)
  with its arguments as JSON, answered in core's `{"message", "data"}` shape.
  The ingress never routes `/internal`, so only the cluster can reach it.
- **Routes:** `providerplatforms.Routes` lists the 30 `/api` routes Provider
  Platforms serves, each with its method and pattern, access (`Admin` or
  `AnyUser`), the path value that must be the caller unless they are an admin
  (`UserParam`), and its feature flag. Core's proxy enforces it; the service's
  route test checks its mux against it.
- **The service token:** every call carries the shared secret as
  `Authorization: Bearer <token>`, and every handler compares it in constant time. A
  missing or wrong token gets a 401; a handler given an empty token rejects
  everything.
- Any answer but a 200 is an error, with the service's message. Arguments that
  aren't valid JSON get a 400; an error from the answerer a 500, logged with
  `slog`. `Users.Get` for a missing user returns an error matching
  `coreapi.ErrUserNotFound` (a 404 on the wire), so core's `Users` must return
  that, not `gorm.ErrRecordNotFound`. `Users.Create` only accepts the student
  role or none through this contract; any other role gets a 400.
- Each question times out after 10 seconds, the demo seed and reset after 5
  minutes, and sooner if the caller's context ends.
- `LiveProviders.MappingsChanged` returns nothing: it logs a failure instead.

## Develop

```bash
go vet ./...
go test -race ./...
golangci-lint run
```

To try a change against core before releasing it, point core at your local
copy in its `go.mod`, and remove the line before you commit:

```
replace github.com/UnlockedLabs/unlocked-contracts => ../unlocked-contracts
```

## Versioning

[Semver](https://semver.org). While the version is `v0.x`:
- **Minor (`v0.2.0`):** a new package or anything else that breaks callers.
- **Patch (`v0.1.1`):** a new value or helper that doesn't.

From `v1.0.0`, a breaking change means a major bump.

## Release

Releases are tagged by hand for now. release-please is deferred until we need
more than the first few versions.

After the PR merges, tag `main` and push the tag:

```bash
git checkout main && git pull
git tag v0.1.0
git push origin v0.1.0
```

Then bump the version in core's `go.mod` with `go get github.com/UnlockedLabs/unlocked-contracts@v0.1.0`.
