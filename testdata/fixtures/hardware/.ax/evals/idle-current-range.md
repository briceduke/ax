---
id: idle-current-range
related: []
runs: 3
---
task: >
  Idle current stays in the expected band.
grader:
  kind: script
  number:
    file: measurements/idle-ma.txt
    min: 3
    max: 6
