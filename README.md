# unlocked-contracts

Shared Go types and contracts for `unlocked-core` and every UnlockEd feature.

Features depend on this small, versioned module instead of on all of core.
It holds plain types and constants, plus small clients for asking core
things, with no dependencies outside the standard library.

| Package | What it holds |
|---|---|
| `features` | Feature-flag names (`features.OpenContent`, …), `features.All`, and `features.Checker`, which asks core whether a feature is on at a facility |
| `identity` | `identity.Claims`, who is logged in, how to read them from a request's context, and `identity.SessionChecker`, which checks the login for a product service |
| `jobs` | Job names (`jobs.GetCourses`, …) and each job's NATS subject (`PubName()`) |
| `users` | User roles (`users.SystemAdmin`, …) and `users.AdminRoles` |

Every value here is stored in a database, sent over NATS or kept in Kratos
identity traits. Changing one breaks running sites, so the tests pin each one.

## Use it

```bash
go get github.com/UnlockedLabs/unlocked-contracts@v0.2.1
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
  `csrf_token` cookie's value (set by core) in the `X-CSRF-Token` header:
  403.

Errors use core's JSON shape, `{"message": "...", "data": null}`.

- **It fails closed.** If Kratos or core can't answer, the request gets a 503
  (logged with `slog`), never a pass. Unlike core, it doesn't clear the
  browser's cookies, so a blip doesn't log anyone out.
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
