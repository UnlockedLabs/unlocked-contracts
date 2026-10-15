# unlocked-contracts

Shared Go types and contracts for `unlocked-core` and every UnlockEd feature.

Features depend on this small, versioned module instead of on all of core.
It holds plain types and constants, plus small clients for asking core
things, with no dependencies outside the standard library.

| Package | What it holds |
|---|---|
| `features` | Feature-flag names (`features.OpenContent`, …), `features.All`, and `features.Checker`, which asks core whether a feature is on at a facility |
| `jobs` | Job names (`jobs.GetCourses`, …) and each job's NATS subject (`PubName()`) |
| `users` | User roles (`users.SystemAdmin`, …) and `users.AdminRoles` |

Every value here is stored in a database, sent over NATS or kept in Kratos
identity traits. Changing one breaks running sites, so the tests pin each one.

## Use it

```bash
go get github.com/UnlockedLabs/unlocked-contracts@v0.1.0
```

```go
import "github.com/UnlockedLabs/unlocked-contracts/jobs"

subject := jobs.GetCourses.PubName() // "tasks.get_courses"
```

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
