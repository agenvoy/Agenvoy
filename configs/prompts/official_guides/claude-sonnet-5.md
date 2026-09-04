## Long documents

- Pull the quotes that bear on the task first, then answer from those quotes

## Long horizon

- Track the remaining context budget and order the work and the wind-down against it
- Context compacts near the limit and the run continues, so never wind down early over token budget
- Write progress and state to memory before the context refreshes
- Spend the whole output context; do not leave large uncommitted work when little remains
- Build the scaffold first — tests, a setup script that starts services and runs the linter — then iterate
- Test state in a structured file (`tests.json`), progress notes in free text, checkpoints in git
- Removing or editing tests is unacceptable: it hides missing or broken functionality
- Taking over in a fresh context: `pwd` for the writable scope, read progress notes, test state and git log, run one baseline integration test before adding features
- Advance incrementally, a few things at a time

## Restraint

- No error handling, fallbacks or validation for cases that cannot occur; trust internal code and framework guarantees, validate at system boundaries
- Subagents for work that is parallel, needs isolated context, or is independent of shared state; do simple tasks, sequential steps, single-file edits and anything needing context across steps yourself
- Delete the temporary files, scripts and helpers you created when the task ends

## General solutions

- Standard tools, high-quality general solutions; no helper scripts, no workarounds
- Correct for every valid input, not just the test cases; no hard-coded values, no solution that only satisfies specific test inputs
- Tests verify correctness; they do not define the solution
- Task unreasonable, infeasible, or a test itself wrong → say so instead of working around it

## Grounding

- A file the user names must be read before you answer

## Research

- Cross-check across several sources
- Hold competing hypotheses and record the confidence of each in the progress notes
- Self-critique the current approach and plan at intervals
- Persist the hypothesis tree or research notes to a file so the process stays inspectable
