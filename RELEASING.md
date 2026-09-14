# Releasing

`park` reports its version via the module version Go records at install time, and
there is no build automation. A release is a git tag plus a matching GitHub
release. Users install from source with `go install`, so a tag is what
`park version` reports for anyone who installs that tagged version.

## Versioning

Tags are `vMAJOR.MINOR.PATCH` (`v1.1.0`). Go's module system only recognises a
tag as a version when it has the `v` prefix and all three numbers. Without
them, `go install github.com/svandragt/park@latest` cannot resolve a release,
falls back to a pseudo-version of the latest commit, and `park version` never
reports a release number.

Bump the minor for new commands or features. Bump the major for a stable
milestone, or a breaking change to the database schema or CLI. Bump the patch
for fixes alone.

Tags up to `1.0` predate this rule and have no `v` prefix. Leave them as they
are: Go ignores them, and the releases attached to them stay readable.

## Steps

1. Make sure `main` is green and pushed. CI runs `go test` on every push.
2. Pick the version and check what changed since the last tag:

   ```bash
   git describe --tags --abbrev=0
   git log --no-show-signature --oneline <last-tag>..HEAD
   ```

3. Create an annotated tag with a short changelog and push it:

   ```bash
   git tag -a v1.1.0 -m "park v1.1.0

   - ...
   "
   git push origin v1.1.0
   ```

4. Create the GitHub release from the tag. Write the notes to a file first to
   keep newlines intact:

   ```bash
   gh release create v1.1.0 --title "v1.1.0" --notes-file notes.md --latest
   ```

   Add `--prerelease` instead of `--latest` for a preview tag.

## Notes style

Lead with a one-line summary, then a "What's new" list grouped by command or
theme. End with a compare link:

```
**Full Changelog**: https://github.com/svandragt/park/compare/<prev>...<this>
```
