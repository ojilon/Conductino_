# 08-02 Wails build + NSIS installer (serious installer)

Goal / Non-goal / Where in code / Concept / Flow / Rewrite options / Exercise / Done when.

Goal: real Windows installer. Where: `release-prep/03-build-and-installer.md`, `build/windows/installer/project.nsi`, `info.json`, `wails.json build:dir`.
Steps: frontend build -> embed -> `wails build` -> NSIS (directory choice, portable zip) -> sign (later).
Exercise: build installer, install on clean VM/folder, uninstall cleanly.
Done when: installer offers drive choice + portable zip works.
