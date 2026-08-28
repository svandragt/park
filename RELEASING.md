# Releasing

`park` has no version string in the code and no build automation. A release is a
git tag plus a matching GitHub release. Users install from source with
`go install`, so tags exist to mark and document a point in history.

## Versioning

Tags are `MAJOR.MINOR` (`0.2`, `1.0`). Bump the minor for new commands or
features; bump the major for a stable milestone or a breaking change to the
database schema or CLI.

## Steps

1. Make sure `main` is green and pushed. CI runs `go test` on every push.
2. Pick the version and check what changed since the last tag:

   ```bash
   git describe --tags --abbrev=0
   git log --no-show-signature --oneline <last-tag>..HEAD
   ```

3. Create an annotated tag with a short changelog and push it:

   ```bash
   git tag -a 1.0 -m "park 1.0

   - ...
   "
   git push origin 1.0
   ```

4. Create the GitHub release from the tag. Write the notes to a file first to
   keep newlines intact:

   ```bash
   gh release create 1.0 --title "1.0" --notes-file notes.md --latest
   ```

   Add `--prerelease` instead of `--latest` for a preview tag.

## Notes style

Lead with a one-line summary, then a "What's new" list grouped by command or
theme. End with a compare link:

```
**Full Changelog**: https://github.com/svandragt/park/compare/<prev>...<this>
```
