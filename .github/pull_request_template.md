## Summary
Closes #

<!-- What changed and why, in a few lines. -->

## Verified Acceptance Criteria
<!-- Copy the issue's acceptance criteria. Tick [x] only items you verified by running something. -->
- [ ] AC1:

## Verification Evidence
<!-- Short command output. Delete lines that do not apply. -->
- `go test -race ./...`
- `mise run lint`
- `govulncheck ./...`
- `commitlint --from=<base> --to=HEAD`

## Not verified yet
<!-- List anything you could not run, and why. Write "None" if everything was verified. -->
None
