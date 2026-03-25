## CORE RULES

1. Use layered architecture (CLI → Orchestrator → Parser → Aggregator → Reporter)

2. Do not mix responsibilities across layers

3. Use structs with methods (no procedural core logic)

4. Do not spawn unbounded goroutines

5. Do not read entire files into memory

6. Do not mutate shared data

7. Return errors, do not print in business logic

8. Keep functions small and focused

## Reporter Rules

- Reporter only formats data and writes output
- Reporter does NOT contain business logic
- Reporter does NOT compute statistics
- Reporter receives fully prepared data

- Use struct tags for JSON
- Use json.MarshalIndent
- Handle file errors properly

## CLI Rules

- CLI only parses input and triggers execution
- CLI does NOT contain business logic
- CLI validates input parameters only
- CLI prints results and errors

- Use flag package for arguments
- Provide defaults for all flags
- Validate input directory before execution


## Package Rules

- Each package has a single responsibility
- Packages must not leak responsibilities

- parser → only parsing
- scanner → only filesystem
- processor → only data processing
- reporter → only output formatting
- cli → only input handling

- main.go only orchestrates flow

---

## Dependency Rules

- parser MUST NOT import processor/reporter/cli
- scanner MUST NOT import processor/reporter
- processor may import parser
- reporter MUST NOT import parser/scanner
- cli MUST NOT import processor internals

---

## Architecture Rule

Data flows in one direction:

cli → scanner → processor → reporter

No reverse dependencies