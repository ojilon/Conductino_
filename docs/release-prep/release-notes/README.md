# Release notes inputs (for CI)

**Purpose.** Files in this folder are the **preferred body** for GitHub Releases created by the tag workflow ([06-ci-and-github-releases.md](../06-ci-and-github-releases.md)).

## How to add notes for a version

1. Copy [TEMPLATE.md](TEMPLATE.md) to `vMAJOR.MINOR.PATCH.md` matching the tag (example: tag `v0.0.1` → file `v0.0.1.md`).
2. Fill Added / Changed / Fixed (short bullets).  
3. Also move items from root `CHANGELOG.md` `## Unreleased` into a version heading when you tag.  
4. Commit the notes **before** or **in the same commit** you tag (the tagged tree must contain the file if CI reads it from the tag).

## Naming rule

| Git tag | Notes file |
|---------|------------|
| `v0.0.1` | `v0.0.1.md` |
| `v0.1.0` | `v0.1.0.md` |

If the file is missing, CI falls back to `CHANGELOG.md` or a stub (see plan 06).

## Agent rules

- Do not put secrets or API keys in release notes.  
- Keep each file small (GitHub UI limits; readability).  
- Pre-release vs full release is a **Release flag**, not a different filename.  
