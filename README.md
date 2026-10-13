# unlocked-contracts

Shared Go types and contracts for `unlocked-core` and every UnlockEd feature.

Features depend on this small, versioned module instead of on all of core.
It holds only plain types and constants, with no dependencies.

| Package | What it holds |
|---|---|
| `features` | Feature-flag names (`features.OpenContent`, …) and `features.All` |
| `jobs` | Job names (`jobs.GetCourses`, …) and each job's NATS subject (`PubName()`) |
| `types` | User roles (`types.SystemAdmin`, …) and `types.AdminRoles` |

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

After the PR merges, tag `main` and push the tag:

```bash
git checkout main && git pull
git tag v0.1.0
git push origin v0.1.0
```

Then bump the version in core's `go.mod` with `go get github.com/UnlockedLabs/unlocked-contracts@v0.1.0`.
