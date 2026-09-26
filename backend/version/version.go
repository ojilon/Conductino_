// Package version is the single source of truth for the app version
// (docs/release-prep/01-versioning-and-tags.md).
//
// Semver MAJOR.MINOR.PATCH; tags look like v0.0.1. Bump this constant in
// the same commit you write docs/release-prep/release-notes/vX.Y.Z.md
// and move CHANGELOG.md entries out of ## Unreleased.
//
// CI stamps the same value into the Windows binary via
// build/windows/info.json templating ({{.Info.ProductVersion}} <- wails.json
// info.productVersion, which must match Version here).
package version

// Version is the current app version (no leading "v"; tags add it).
const Version = "0.0.1"
