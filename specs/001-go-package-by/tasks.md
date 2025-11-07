# Tasks: Identify and Fix Long-Running Tests

**Input**: Design documents from `/specs/001-go-package-by/`
**Prerequisites**: plan.md, research.md, data-model.md, contracts/

## Execution Flow (main)
```
1. Load plan.md from feature directory
   → Extracted: Go 1.24.0, AST parsing, mock generation, CLI tool structure
2. Load design documents:
   → data-model.md: 14 entities extracted → model tasks
   → contracts/: 4 interface files → implementation tasks
   → research.md: Technology decisions → setup and implementation tasks
3. Generate tasks by category:
   → Setup: project structure, dependencies
   → Data Models: 14 entities from data-model.md
   → AST Utilities: Parsing, file analysis, function extraction
   → Pattern Detectors: 6 different pattern detectors
   → Mock Generation: Interface analysis, code generation, templates
   → Fix Application: Code modification, backup, validation
   → Reporting: 4 output formats (JSON, HTML, Markdown, Plain)
   → CLI: Command parsing, flag handling, main entry
   → Testing: Unit tests, integration tests, end-to-end tests
4. Apply task rules:
   → Different files = mark [P] for parallel
   → Same file = sequential (no [P])
   → Tests before implementation (TDD)
5. Number tasks sequentially (T001, T002...)
6. Generate dependency graph
7. Create parallel execution examples
```

## Format: `[ID] [P?] Description`
- **[P]**: Can run in parallel (different files, no dependencies)
- Include exact file paths in descriptions

## Path Conventions
- CLI tool: `cmd/test-analyzer/`
- Internal utilities: `cmd/test-analyzer/internal/`
- Integration tests: `tests/integration/test-analyzer/`
- Scripts: `scripts/`

## Phase 3.1: Setup
- [X] T001 Create project structure: `cmd/test-analyzer/` directory with `main.go`, `analyzer.go`, `fixer.go`, `reporter.go`
- [X] T002 Create internal subdirectories: `cmd/test-analyzer/internal/ast/`, `cmd/test-analyzer/internal/patterns/`, `cmd/test-analyzer/internal/mocks/`, `cmd/test-analyzer/internal/validation/`
- [X] T003 [P] Create `cmd/test-analyzer/errors.go` with custom error types following Op/Err/Code pattern
- [X] T004 [P] Create `cmd/test-analyzer/config.go` with configuration struct and functional options pattern
- [X] T005 [P] Create `scripts/analyze-tests.sh` convenience wrapper script

## Phase 3.2: Data Models (from data-model.md)
- [X] T006 Create `cmd/test-analyzer/types.go` with TestFile struct (Path, Package, Functions, HasIntegrationSuffix, AST)
- [X] T007 Create `cmd/test-analyzer/types.go` with TestFunction struct (Name, Type, File, LineStart, LineEnd, Issues, HasTimeout, TimeoutDuration, ExecutionTime, UsesActualImplementation, UsesMocks, MixedUsage)
- [X] T008 Create `cmd/test-analyzer/types.go` with TestType enum (Unit, Integration, Load) and detection logic
- [X] T009 Create `cmd/test-analyzer/types.go` with PerformanceIssue struct (Type, Severity, Location, Description, Context, Fixable, Fix)
- [X] T010 Create `cmd/test-analyzer/types.go` with IssueType enum (InfiniteLoop, MissingTimeout, LargeIteration, HighConcurrency, SleepDelay, ActualImplementationUsage, MixedMockRealUsage, MissingMock, BenchmarkHelperUsage, Other)
- [X] T011 Create `cmd/test-analyzer/types.go` with Severity enum (Low, Medium, High, Critical) and severity assignment logic
- [X] T012 Create `cmd/test-analyzer/types.go` with Location struct (Package, File, Function, LineStart, LineEnd, ColumnStart, ColumnEnd)
- [X] T013 Create `cmd/test-analyzer/types.go` with Fix struct (Issue, Type, Changes, Status, ValidationResult, BackupPath, AppliedAt)
- [X] T014 Create `cmd/test-analyzer/types.go` with FixType enum (AddTimeout, ReduceIterations, OptimizeSleep, AddLoopExit, ReplaceWithMock, CreateMock, UpdateTestFile)
- [X] T015 Create `cmd/test-analyzer/types.go` with FixStatus enum (Proposed, Applied, Validated, Failed, RolledBack)
- [X] T016 Create `cmd/test-analyzer/types.go` with CodeChange struct (File, LineStart, LineEnd, OldCode, NewCode, Description)
- [X] T017 Create `cmd/test-analyzer/types.go` with ValidationResult struct (Fix, InterfaceCompatible, TestsPass, ExecutionTimeImproved, OriginalExecutionTime, NewExecutionTime, Errors, TestOutput, ValidatedAt)
- [X] T018 Create `cmd/test-analyzer/types.go` with MockImplementation struct (ComponentName, InterfaceName, Package, FilePath, Code, InterfaceMethods, Status, RequiresManualCompletion, GeneratedAt)
- [X] T019 Create `cmd/test-analyzer/types.go` with MockStatus enum (Template, Complete, Validated)
- [X] T020 Create `cmd/test-analyzer/types.go` with MethodSignature struct (Name, Parameters, Returns, Receiver)
- [X] T021 Create `cmd/test-analyzer/types.go` with Parameter and Return type definitions
- [X] T022 Create `cmd/test-analyzer/types.go` with AnalysisReport struct (PackagesAnalyzed, FilesAnalyzed, FunctionsAnalyzed, IssuesFound, IssuesByType, IssuesBySeverity, IssuesByPackage, FixesApplied, FixesFailed, ExecutionTime, GeneratedAt, Packages)
- [X] T023 Create `cmd/test-analyzer/types.go` with PackageAnalysis struct (Package, Files, Issues, Summary, AnalyzedAt)
- [X] T024 Create `cmd/test-analyzer/types.go` with FileAnalysis struct (File, Functions, Issues, AnalyzedAt)
- [X] T025 Create `cmd/test-analyzer/types.go` with AnalysisSummary struct (TotalFiles, TotalFunctions, TotalIssues, IssuesByType, IssuesBySeverity)
- [X] T026 Create `cmd/test-analyzer/types.go` with ReportFormat enum (FormatJSON, FormatHTML, FormatMarkdown, FormatPlain)

## Phase 3.3: AST Parsing Utilities
- [X] T027 Create `cmd/test-analyzer/internal/ast/parser.go` with ParseFile function to parse Go test files using go/parser
- [X] T028 Create `cmd/test-analyzer/internal/ast/walker.go` with ASTWalker to traverse AST nodes and extract test functions
- [X] T029 Create `cmd/test-analyzer/internal/ast/extractor.go` with ExtractTestFunctions function to extract Test*, Benchmark*, Fuzz* functions from AST
- [X] T030 Create `cmd/test-analyzer/internal/ast/analyzer.go` with AnalyzeFunction function to analyze function AST for patterns

## Phase 3.4: Pattern Detectors (from analyzer-interface.md)
- [X] T031 Create `cmd/test-analyzer/internal/patterns/detector.go` with PatternDetector interface
- [X] T032 Create `cmd/test-analyzer/internal/patterns/infinite_loop.go` with DetectInfiniteLoops function to detect `for { }` patterns without exit conditions, including `ConcurrentTestRunner` patterns with timer-based infinite loops (`for { select { ... } }`)
- [X] T033 Create `cmd/test-analyzer/internal/patterns/timeout.go` with DetectMissingTimeouts function to check for context.WithTimeout, t.Deadline, or test-level timeouts
- [X] T034 Create `cmd/test-analyzer/internal/patterns/iterations.go` with DetectLargeIterations function to detect hardcoded iteration counts exceeding thresholds (100+ simple, 20+ complex)
- [X] T035 Create `cmd/test-analyzer/internal/patterns/complexity.go` with DetectOperationComplexity function to identify complex operations (network, I/O, DB calls) within loops
- [X] T036 Create `cmd/test-analyzer/internal/patterns/implementation.go` with DetectActualImplementationUsage function to detect real provider instantiations, factory calls, API clients, file/DB operations
- [X] T037 Create `cmd/test-analyzer/internal/patterns/mocks.go` with DetectMissingMocks function to analyze interface usage and compare against available mocks in test_utils.go, internal/mock/, providers/mock/
- [X] T038 Create `cmd/test-analyzer/internal/patterns/sleep.go` with DetectSleepDelays function to detect time.Sleep calls accumulating to significant duration (>100ms)
- [X] T039 Create `cmd/test-analyzer/internal/patterns/benchmark.go` with DetectBenchmarkHelperUsage function to detect benchmark helper calls during regular tests
- [X] T040 Create `cmd/test-analyzer/internal/patterns/test_type.go` with DetectTestType function to determine Unit/Integration/Load based on file naming, function naming, and test characteristics

## Phase 3.5: Analyzer Implementation (from analyzer-interface.md)
- [X] T041 Create `cmd/test-analyzer/analyzer.go` with Analyzer interface (AnalyzePackage, AnalyzeFile, DetectIssues methods)
- [X] T042 Create `cmd/test-analyzer/analyzer.go` with NewAnalyzer constructor function accepting PatternDetector and AST utilities
- [X] T043 Implement AnalyzePackage method in `cmd/test-analyzer/analyzer.go` to analyze all *_test.go files in a package
- [X] T044 Implement AnalyzeFile method in `cmd/test-analyzer/analyzer.go` to parse and analyze a single test file
- [X] T045 Implement DetectIssues method in `cmd/test-analyzer/analyzer.go` to coordinate all pattern detectors and aggregate results

## Phase 3.6: Mock Generation (from fixer-interface.md)
- [X] T046 Create `cmd/test-analyzer/internal/mocks/generator.go` with MockGenerator interface (GenerateMock, GenerateMockTemplate, VerifyInterfaceCompatibility)
- [X] T047 Create `cmd/test-analyzer/internal/mocks/interface_analyzer.go` with AnalyzeInterface function to extract interface definition and method signatures using go/types
- [X] T048 Create `cmd/test-analyzer/internal/mocks/pattern_extractor.go` with ExtractMockPattern function to analyze existing test_utils.go files and extract AdvancedMock pattern structure
- [X] T049 Create `cmd/test-analyzer/internal/mocks/code_generator.go` with GenerateMockCode function using text/template to generate AdvancedMock{Component} struct following established pattern
- [X] T050 Create `cmd/test-analyzer/internal/mocks/template_generator.go` with GenerateTemplate function to create mock templates with TODOs and placeholders for complex cases
- [X] T051 Create `cmd/test-analyzer/internal/mocks/validator.go` with VerifyInterfaceCompatibility function using reflect to verify mock implements same interface as real implementation

## Phase 3.7: Fix Application (from fixer-interface.md)
- [X] T052 Create `cmd/test-analyzer/fixer.go` with Fixer interface (ApplyFix, ValidateFix, RollbackFix methods)
- [X] T053 Create `cmd/test-analyzer/fixer.go` with NewFixer constructor function accepting MockGenerator, CodeModifier, and Validator
- [X] T054 Implement ApplyFix method in `cmd/test-analyzer/fixer.go` to determine fix type, generate code changes, create backup, and apply changes
- [X] T055 Create `cmd/test-analyzer/internal/validation/fix_validator.go` with ValidateFix function implementing dual validation (interface compatibility + test execution)
- [X] T056 Create `cmd/test-analyzer/internal/validation/test_runner.go` with RunTests function to execute go test on affected package and measure execution time
- [X] T057 Create `cmd/test-analyzer/internal/validation/interface_checker.go` with CheckInterfaceCompatibility function to verify mock interface matches actual interface using reflect
- [X] T058 Implement ValidateFix method in `cmd/test-analyzer/fixer.go` to coordinate dual validation and return ValidationResult
- [X] T059 Implement RollbackFix method in `cmd/test-analyzer/fixer.go` to restore file from backup and update fix status
- [X] T060 Create `cmd/test-analyzer/internal/code/modifier.go` with CodeModifier interface (CreateBackup, ApplyCodeChange, FormatCode)
- [X] T061 Implement CreateBackup method in `cmd/test-analyzer/internal/code/modifier.go` to create timestamped backup files
- [X] T062 Implement ApplyCodeChange method in `cmd/test-analyzer/internal/code/modifier.go` to replace code at specified line numbers and format using go/format
- [X] T063 Implement FormatCode method in `cmd/test-analyzer/internal/code/modifier.go` to format Go code using go/format

## Phase 3.8: Fix Type Implementations
- [X] T064 Create `cmd/test-analyzer/internal/fixes/timeout.go` with AddTimeoutFix function to add context.WithTimeout to test functions
- [X] T065 Create `cmd/test-analyzer/internal/fixes/iterations.go` with ReduceIterationsFix function to reduce excessive iteration counts
- [X] T066 Create `cmd/test-analyzer/internal/fixes/sleep.go` with OptimizeSleepFix function to reduce or remove sleep durations
- [X] T067 Create `cmd/test-analyzer/internal/fixes/loop.go` with AddLoopExitFix function to add proper exit conditions to infinite loops
- [X] T068 Create `cmd/test-analyzer/internal/fixes/mock_replacement.go` with ReplaceWithMockFix function to replace actual implementations with mocks in unit tests
- [X] T069 Create `cmd/test-analyzer/internal/fixes/mock_creation.go` with CreateMockFix function to generate and insert missing mock implementations
- [X] T070 Create `cmd/test-analyzer/internal/fixes/test_update.go` with UpdateTestFileFix function to update test files to use newly created mocks

## Phase 3.9: Reporting (from reporter-interface.md)
- [X] T071 Create `cmd/test-analyzer/reporter.go` with Reporter interface (GenerateReport, GenerateSummary, GeneratePackageReport)
- [X] T072 Create `cmd/test-analyzer/reporter.go` with NewReporter constructor function
- [X] T073 Create `cmd/test-analyzer/internal/report/json.go` with GenerateJSONReport function to generate JSON format report
- [X] T074 Create `cmd/test-analyzer/internal/report/html.go` with GenerateHTMLReport function to generate interactive HTML report with charts and color-coded severity
- [X] T075 Create `cmd/test-analyzer/internal/report/markdown.go` with GenerateMarkdownReport function to generate markdown format with code blocks and tables
- [X] T076 Create `cmd/test-analyzer/internal/report/plain.go` with GeneratePlainReport function to generate terminal-friendly plain text output with color support
- [X] T077 Implement GenerateReport method in `cmd/test-analyzer/reporter.go` to route to appropriate format generator
- [X] T078 Implement GenerateSummary method in `cmd/test-analyzer/reporter.go` to generate human-readable summary text
- [X] T079 Implement GeneratePackageReport method in `cmd/test-analyzer/reporter.go` to generate package-specific reports

## Phase 3.10: CLI Implementation (from cli-interface.md)
- [X] T080 Create `cmd/test-analyzer/main.go` with main function and command-line argument parsing
- [X] T081 Create `cmd/test-analyzer/internal/cli/flags.go` with flag definitions for all CLI flags (--dry-run, --output, --auto-fix, --fix-types, thresholds, filters, etc.)
- [X] T082 Create `cmd/test-analyzer/internal/cli/parser.go` with ParseFlags function to parse command-line arguments and flags
- [X] T083 Create `cmd/test-analyzer/internal/cli/runner.go` with RunAnalysis function to coordinate analyzer, fixer, and reporter based on CLI flags
- [X] T084 Implement package discovery in `cmd/test-analyzer/main.go` to find all packages in pkg/ directory or specified packages
- [X] T085 Implement output handling in `cmd/test-analyzer/main.go` to write reports to stdout or file based on --output and --output-file flags
- [X] T086 Implement exit code handling in `cmd/test-analyzer/main.go` to return appropriate exit codes (0=success, 1=analysis errors, 2=fix errors, 3=validation errors, 4=invalid args)
- [X] T087 Create `cmd/test-analyzer/internal/cli/help.go` with PrintHelp function to display usage information and flag descriptions
- [X] T088 Create `cmd/test-analyzer/internal/cli/validator.go` with ValidateFlags function to check for conflicting flag combinations (e.g., --dry-run with --auto-fix, --skip-validation with --auto-fix) and return validation errors

## Phase 3.11: Testing - Unit Tests
- [X] T089 [P] Create `cmd/test-analyzer/types_test.go` with unit tests for all data model types and validation rules
- [X] T090 [P] Create `cmd/test-analyzer/internal/ast/parser_test.go` with unit tests for AST parsing functions
- [X] T091 [P] Create `cmd/test-analyzer/internal/ast/walker_test.go` with unit tests for ASTWalker traversal
- [X] T092 [P] Create `cmd/test-analyzer/internal/ast/extractor_test.go` with unit tests for test function extraction
- [X] T093 [P] Create `cmd/test-analyzer/internal/patterns/infinite_loop_test.go` with unit tests for infinite loop detection, including `ConcurrentTestRunner` patterns with timer-based infinite loops
- [X] T094 [P] Create `cmd/test-analyzer/internal/patterns/timeout_test.go` with unit tests for missing timeout detection
- [X] T095 [P] Create `cmd/test-analyzer/internal/patterns/iterations_test.go` with unit tests for large iteration detection
- [X] T096 [P] Create `cmd/test-analyzer/internal/patterns/complexity_test.go` with unit tests for operation complexity detection
- [X] T097 [P] Create `cmd/test-analyzer/internal/patterns/implementation_test.go` with unit tests for actual implementation usage detection
- [X] T098 [P] Create `cmd/test-analyzer/internal/patterns/mocks_test.go` with unit tests for missing mock detection
- [X] T099 [P] Create `cmd/test-analyzer/internal/patterns/sleep_test.go` with unit tests for sleep delay detection
- [X] T100 [P] Create `cmd/test-analyzer/internal/patterns/benchmark_test.go` with unit tests for benchmark helper usage detection
- [X] T101 [P] Create `cmd/test-analyzer/internal/patterns/test_type_test.go` with unit tests for test type detection logic
- [X] T102 [P] Create `cmd/test-analyzer/analyzer_test.go` with unit tests for Analyzer interface implementation
- [X] T103 [P] Create `cmd/test-analyzer/internal/mocks/generator_test.go` with unit tests for mock generation
- [X] T104 [P] Create `cmd/test-analyzer/internal/mocks/interface_analyzer_test.go` with unit tests for interface analysis
- [X] T105 [P] Create `cmd/test-analyzer/internal/mocks/code_generator_test.go` with unit tests for mock code generation
- [X] T106 [P] Create `cmd/test-analyzer/internal/mocks/validator_test.go` with unit tests for interface compatibility verification
- [X] T107 [P] Create `cmd/test-analyzer/fixer_test.go` with unit tests for Fixer interface implementation
- [X] T108 [P] Create `cmd/test-analyzer/internal/validation/fix_validator_test.go` with unit tests for fix validation
- [X] T109 [P] Create `cmd/test-analyzer/internal/validation/test_runner_test.go` with unit tests for test execution
- [X] T110 [P] Create `cmd/test-analyzer/internal/code/modifier_test.go` with unit tests for code modification functions
- [X] T111 [P] Create `cmd/test-analyzer/internal/fixes/timeout_test.go` with unit tests for timeout fix application
- [X] T112 [P] Create `cmd/test-analyzer/internal/fixes/iterations_test.go` with unit tests for iteration reduction fix
- [X] T113 [P] Create `cmd/test-analyzer/internal/fixes/mock_replacement_test.go` with unit tests for mock replacement fix
- [X] T114 [P] Create `cmd/test-analyzer/reporter_test.go` with unit tests for Reporter interface implementation
- [X] T115 [P] Create `cmd/test-analyzer/internal/report/json_test.go` with unit tests for JSON report generation
- [X] T116 [P] Create `cmd/test-analyzer/internal/report/html_test.go` with unit tests for HTML report generation
- [X] T117 [P] Create `cmd/test-analyzer/internal/report/markdown_test.go` with unit tests for Markdown report generation
- [X] T118 [P] Create `cmd/test-analyzer/internal/report/plain_test.go` with unit tests for Plain text report generation
- [X] T119 [P] Create `cmd/test-analyzer/internal/cli/flags_test.go` with unit tests for flag parsing
- [X] T120 [P] Create `cmd/test-analyzer/internal/cli/parser_test.go` with unit tests for command-line argument parsing
- [X] T121 [P] Create `cmd/test-analyzer/internal/cli/validator_test.go` with unit tests for flag validation

## Phase 3.12: Testing - Integration Tests
- [X] T122 Create `tests/integration/test-analyzer/analyzer_test.go` with integration tests for Analyzer analyzing real test files from pkg/ directory
- [X] T123 Create `tests/integration/test-analyzer/fixer_test.go` with integration tests for Fixer applying fixes to real test files and validating
- [X] T124 Create `tests/integration/test-analyzer/mock_generation_test.go` with integration tests for MockGenerator creating mocks for real interfaces
- [X] T125 Create `tests/integration/test-analyzer/end_to_end_test.go` with end-to-end test scenarios from quickstart.md (dry run, specific package analysis, report generation, auto-fix with validation)
- [X] T126 Create `tests/integration/test-analyzer/package_analysis_test.go` with integration tests analyzing each of the 14 framework packages (pkg/agents, pkg/llms, pkg/memory, etc.)

## Phase 3.13: Documentation and Polish
- [X] T127 Create `cmd/test-analyzer/README.md` with tool documentation, usage examples, and architecture overview
- [X] T128 Update main repository README.md to document the test-analyzer tool
- [X] T129 Create `docs/tools/test-analyzer.md` with comprehensive documentation including all flags, examples, and troubleshooting
- [X] T130 [P] Add code comments and documentation to all public functions and types following Go documentation conventions
- [X] T131 [P] Run gofmt and golint on all source files to ensure code quality
- [X] T132 [P] Create example test files in `tests/integration/test-analyzer/fixtures/` with various performance issues for testing

## Dependencies
- T001-T002 (Setup) before all other tasks
- T003-T005 (Config/Errors) before core implementation
- T006-T026 (Data Models) before T027-T045 (AST and Analyzer) - MUST be sequential (all modify types.go)
- T027-T030 (AST Utilities) before T031-T040 (Pattern Detectors)
- T031-T040 (Pattern Detectors) before T041-T045 (Analyzer Implementation)
- T041-T045 (Analyzer) before T052-T063 (Fixer)
- T046-T051 (Mock Generation) before T052-T063 (Fixer)
- T052-T063 (Fixer) before T064-T070 (Fix Type Implementations)
- T071-T079 (Reporter) can be parallel with Fixer (different files)
- T080-T087 (CLI) depends on Analyzer, Fixer, and Reporter
- T081-T082 (CLI flags/parser) before T088 (CLI validator)
- T089-T121 (Unit Tests) can be parallel after implementation
- T122-T126 (Integration Tests) depend on all implementation tasks
- T127-T132 (Documentation) can be parallel with testing

## Parallel Execution Examples

### Example 1: Data Model Tasks (T006-T026)
All data model tasks MUST be sequential as they all modify the same file (`types.go`):
```bash
# T006-T026: All type definitions in cmd/test-analyzer/types.go
# These MUST be executed sequentially to avoid merge conflicts
```

### Example 2: Pattern Detector Tests (T093-T101)
All pattern detector test files are independent and can run in parallel:
```bash
Task: "Create cmd/test-analyzer/internal/patterns/infinite_loop_test.go"
Task: "Create cmd/test-analyzer/internal/patterns/timeout_test.go"
Task: "Create cmd/test-analyzer/internal/patterns/iterations_test.go"
Task: "Create cmd/test-analyzer/internal/patterns/complexity_test.go"
Task: "Create cmd/test-analyzer/internal/patterns/implementation_test.go"
Task: "Create cmd/test-analyzer/internal/patterns/mocks_test.go"
Task: "Create cmd/test-analyzer/internal/patterns/sleep_test.go"
Task: "Create cmd/test-analyzer/internal/patterns/benchmark_test.go"
Task: "Create cmd/test-analyzer/internal/patterns/test_type_test.go"
```

### Example 3: Report Format Tests (T114-T117)
All report format test files are independent:
```bash
Task: "Create cmd/test-analyzer/internal/report/json_test.go"
Task: "Create cmd/test-analyzer/internal/report/html_test.go"
Task: "Create cmd/test-analyzer/internal/report/markdown_test.go"
Task: "Create cmd/test-analyzer/internal/report/plain_test.go"
```

### Example 4: Unit Tests (T089-T121)
Most unit test files are independent and can run in parallel:
```bash
# Launch multiple test file creation tasks in parallel
Task: "Create cmd/test-analyzer/types_test.go"
Task: "Create cmd/test-analyzer/internal/ast/parser_test.go"
Task: "Create cmd/test-analyzer/internal/patterns/infinite_loop_test.go"
Task: "Create cmd/test-analyzer/internal/mocks/generator_test.go"
Task: "Create cmd/test-analyzer/fixer_test.go"
Task: "Create cmd/test-analyzer/reporter_test.go"
```

## Notes
- [P] tasks = different files, no dependencies
- Data model tasks (T006-T026) all modify `types.go` - MUST be sequential (no [P] markers)
- Verify tests fail before implementing (TDD approach)
- Commit after each major component (Setup, Data Models, AST, Patterns, Analyzer, Fixer, Reporter, CLI)
- All tasks specify exact file paths
- Follow Go naming conventions and package structure
- Ensure all error handling follows Op/Err/Code pattern
- All code must be formatted with gofmt

## Task Generation Rules Applied
1. **From Data Model**: 14 entities → 21 type definition tasks (T006-T026)
2. **From Contracts**: 4 interface files → implementation tasks for Analyzer, Fixer, Reporter, MockGenerator
3. **From Research**: AST parsing, pattern detection, mock generation, validation → specific implementation tasks
4. **From CLI Interface**: All flags and commands → CLI implementation tasks
5. **From Quickstart**: Test scenarios → integration test tasks

## Validation Checklist
### Constitutional Compliance
- [x] Tool structure follows Go CLI conventions (cmd/test-analyzer/)
- [x] Error handling with Op/Err/Code pattern (T003)
- [x] Functional options for configuration (T004)
- [x] Comprehensive testing requirements (T088-T124)

### Task Quality
- [x] All contracts have corresponding implementation tasks
- [x] All entities have model tasks
- [x] All pattern detectors have implementation and test tasks
- [x] All fix types have implementation tasks
- [x] All report formats have implementation and test tasks
- [x] CLI interface fully covered
- [x] Integration tests cover all scenarios from quickstart.md
- [x] Each task specifies exact file path
- [x] Parallel tasks truly independent (different files)

---
*Based on Constitution v1.0.0 - See `.specify/memory/constitution.md`*
*Total Tasks: 132*

