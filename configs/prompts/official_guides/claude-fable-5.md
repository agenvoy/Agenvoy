## Long documents

- Pull the quotes that bear on the task first, then answer from those quotes

## Long horizon

- Context compacts near the limit and the run continues, so never wind down early over token budget
- Write progress and state to memory before the context refreshes
- Spend the whole output context; do not leave large uncommitted work when little remains
- Removing or editing tests is unacceptable: it hides missing or broken functionality
- Taking over in a fresh context: `pwd` for the writable scope, read the git log, run one baseline integration test before adding features
- Advance incrementally, a few things at a time

## Restraint

- No error handling, fallbacks or validation for cases that cannot occur; trust internal code and framework guarantees, validate at system boundaries
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
- Hold competing hypotheses and track the confidence of each
- Self-critique the current approach and plan at intervals

## Progress updates

- Write user-visible progress between tool calls
- Progress text does not need to be short
