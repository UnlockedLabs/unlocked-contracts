## Pre-Submission PR Checklist

- [ ] No debug/console/fmt.Println statements
- [ ] Unnecessary development comments removed
- [ ] All acceptance criteria verified
- [ ] Functions according to **ticket specifications**
- [ ] Branch rebased with latest main
- [ ] No existing value renamed or removed (values are stored in databases and sent over NATS)
- [ ] Tests updated for every new value
- [ ] Breaking changes called out below, with the version bump they need

## Description of the change

Please provide a brief description of the changes included in this PR.

- **Related issues**: Link to related JIRA ticket that this closes.

## Release

- **Version after merge**: e.g. `v0.2.0`, or "no release" if nothing callers use changed.
- **Breaking change?**: yes / no. If yes, list what callers must change.
- **Repos to update after tagging**: e.g. `unlocked-core`.

## Additional context

Please include additional context or information that the reviewer needs to understand the PR. This includes:

- Known issues that the PR does not address
- What areas to focus on
- What decisions were considered and decided against.
