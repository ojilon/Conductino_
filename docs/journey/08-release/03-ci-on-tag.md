# 08-03 CI on tag + pre-releases

Goal / Non-goal / Where in code / Concept / Flow / Rewrite options / Exercise / Done when.

Goal: tag pushes build + test. Where: `.github/workflows/ci.yml`, `release.yml`, `release-prep/06-ci-and-github-releases.md` (in main).
Exercise: push a tag, watch CI build frontend before Go (embed needs dist).
Done when: red CI blocks release.
