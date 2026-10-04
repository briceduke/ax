---
id: harness-format
related: [scaffold/verify-pinouts]
runs: 3
---
task: >
  After compile, log and core files still satisfy format checks
  even if the pinout scaffold is omitted.
grader:
  kind: script
  check: ax-format
