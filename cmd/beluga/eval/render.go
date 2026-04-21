package eval

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"sort"
	"text/tabwriter"
)

// RenderText writes a human-readable summary of the report to w: a
// per-row table ordered by sample index and an aggregate footer. Used
// for stdout in interactive and CI logs. Output is deterministic —
// metric columns are emitted in alphabetical order so golden-file
// tests round-trip cleanly.
func RenderText(w io.Writer, rep *Report) error {
	if rep == nil {
		return fmt.Errorf("nil report")
	}
	if rep.DryRun {
		_, err := fmt.Fprintf(w, "dry-run: %d skipped, %d samples queued (run_id=%s)\n",
			rep.Skipped, len(rep.Samples), rep.RunID)
		return err
	}

	metricNames := sortedMetricNames(rep)

	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	header := "IDX\tROW_ID\tINPUT\tOUTPUT\tEXPECTED"
	for _, name := range metricNames {
		header += "\t" + name
	}
	header += "\tERROR"
	fmt.Fprintln(tw, header)

	for _, s := range rep.Samples {
		row := fmt.Sprintf("%d\t%s\t%s\t%s\t%s",
			s.Index, s.RowID, truncate(s.Input, 40), truncate(s.Output, 40), truncate(s.Expected, 40))
		for _, name := range metricNames {
			if v, ok := s.Scores[name]; ok {
				row += fmt.Sprintf("\t%.2f", v)
			} else {
				row += "\t-"
			}
		}
		row += "\t" + truncate(s.Error, 60)
		fmt.Fprintln(tw, row)
	}
	if err := tw.Flush(); err != nil {
		return err
	}

	fmt.Fprintln(w)
	fmt.Fprintf(w, "run_id:  %s\n", rep.RunID)
	fmt.Fprintf(w, "dataset: %s (%s)\n", rep.DatasetName, rep.DatasetPath)
	fmt.Fprintf(w, "samples: %d", len(rep.Samples))
	if rep.Skipped > 0 {
		fmt.Fprintf(w, " (+%d skipped)", rep.Skipped)
	}
	fmt.Fprintln(w)
	fmt.Fprintf(w, "duration: %s\n", rep.Duration)

	if len(metricNames) > 0 {
		fmt.Fprintln(w, "aggregate:")
		for _, name := range metricNames {
			fmt.Fprintf(w, "  %s: %.4f\n", name, rep.Aggregate[name])
		}
	}
	if len(rep.Errors) > 0 {
		fmt.Fprintf(w, "errors: %d\n", len(rep.Errors))
	}
	return nil
}

// RenderJSON writes the report as indented JSON to w. This is the
// always-on CI artefact: every `beluga eval` run writes eval-report.json
// regardless of any --format toggle (brief §Q3, devops-expert §Q3).
func RenderJSON(w io.Writer, rep *Report) error {
	if rep == nil {
		return fmt.Errorf("nil report")
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(rep)
}

// junitTestSuites is the dorny/test-reporter-compatible wire shape.
// A single <testsuite> wraps every sample as a <testcase>; per-row
// errors become <failure> children so PR annotations point at the
// failing dataset row. Non-exact_match metrics show up as <properties>
// on the testcase for UI hover-over consumption.
type junitTestSuites struct {
	XMLName  xml.Name `xml:"testsuites"`
	Name     string   `xml:"name,attr"`
	Tests    int      `xml:"tests,attr"`
	Failures int      `xml:"failures,attr"`
	Errors   int      `xml:"errors,attr"`
	Time     float64  `xml:"time,attr"`
	Suites   []junitTestSuite
}

type junitTestSuite struct {
	XMLName   xml.Name `xml:"testsuite"`
	Name      string   `xml:"name,attr"`
	Tests     int      `xml:"tests,attr"`
	Failures  int      `xml:"failures,attr"`
	Errors    int      `xml:"errors,attr"`
	Time      float64  `xml:"time,attr"`
	Testcases []junitTestCase
}

type junitTestCase struct {
	XMLName    xml.Name         `xml:"testcase"`
	Classname  string           `xml:"classname,attr"`
	Name       string           `xml:"name,attr"`
	Time       float64          `xml:"time,attr"`
	Failure    *junitFailure    `xml:"failure,omitempty"`
	Properties *junitProperties `xml:"properties,omitempty"`
}

type junitFailure struct {
	Message string `xml:"message,attr"`
	Type    string `xml:"type,attr"`
	Body    string `xml:",chardata"`
}

type junitProperties struct {
	Items []junitProperty `xml:"property"`
}

type junitProperty struct {
	Name  string `xml:"name,attr"`
	Value string `xml:"value,attr"`
}

// RenderJUnit writes a JUnit XML report to w compatible with
// dorny/test-reporter. It is called only when cfg.WantsJUnit() is
// true. The MVP emits a single suite named after the dataset; per-row
// failures are derived from cfg-less heuristics — a sample is "failed"
// when it recorded an exec error OR exact_match=0 when exact_match is
// the configured metric. Metric scores are attached as properties so
// they survive the trip through dorny without custom UI work.
func RenderJUnit(w io.Writer, rep *Report) error {
	if rep == nil {
		return fmt.Errorf("nil report")
	}

	suite := junitTestSuite{
		Name:      rep.DatasetName,
		Tests:     len(rep.Samples),
		Time:      rep.Duration.Seconds(),
		Testcases: make([]junitTestCase, 0, len(rep.Samples)),
	}

	for _, s := range rep.Samples {
		tc := junitTestCase{
			Classname: rep.DatasetName,
			Name:      junitTestcaseName(s),
		}
		switch {
		case s.Error != "":
			tc.Failure = &junitFailure{
				Message: s.Error,
				Type:    "exec_error",
				Body:    s.Error,
			}
			suite.Errors++
		case isFailedByExactMatch(s):
			tc.Failure = &junitFailure{
				Message: fmt.Sprintf("exact_match=%.2f (expected=%q, got=%q)",
					s.Scores["exact_match"], s.Expected, s.Output),
				Type: "exact_match_fail",
				Body: s.Output,
			}
			suite.Failures++
		}
		if len(s.Scores) > 0 {
			tc.Properties = &junitProperties{Items: scoresAsProperties(s.Scores)}
		}
		suite.Testcases = append(suite.Testcases, tc)
	}

	suites := junitTestSuites{
		Name:     rep.DatasetName,
		Tests:    suite.Tests,
		Failures: suite.Failures,
		Errors:   suite.Errors,
		Time:     suite.Time,
		Suites:   []junitTestSuite{suite},
	}

	if _, err := io.WriteString(w, xml.Header); err != nil {
		return err
	}
	enc := xml.NewEncoder(w)
	enc.Indent("", "  ")
	if err := enc.Encode(suites); err != nil {
		return err
	}
	_, err := io.WriteString(w, "\n")
	return err
}

// junitTestcaseName derives a stable per-row identifier. RowID is
// preferred for join-with-JSON; Index is the fallback when RowID is
// empty (e.g., a future path where samples lack row IDs).
func junitTestcaseName(s SampleReport) string {
	if s.RowID != "" {
		return s.RowID
	}
	return fmt.Sprintf("sample_%04d", s.Index)
}

// isFailedByExactMatch returns true when exact_match is the only
// correctness metric present and the sample scored 0. Other metrics
// (latency, LLM-judge) are not treated as pass/fail here because their
// thresholds are context-specific and surface via the JSON report.
func isFailedByExactMatch(s SampleReport) bool {
	v, ok := s.Scores["exact_match"]
	return ok && v == 0
}

// scoresAsProperties renders metric scores in deterministic order for
// golden-file stability.
func scoresAsProperties(scores map[string]float64) []junitProperty {
	names := make([]string, 0, len(scores))
	for name := range scores {
		names = append(names, name)
	}
	sort.Strings(names)
	out := make([]junitProperty, 0, len(names))
	for _, name := range names {
		out = append(out, junitProperty{
			Name:  name,
			Value: fmt.Sprintf("%.4f", scores[name]),
		})
	}
	return out
}

// sortedMetricNames returns the union of metrics present in the
// aggregate map (authoritative) plus any per-row scores, sorted
// alphabetically for deterministic column ordering across runs.
func sortedMetricNames(rep *Report) []string {
	seen := make(map[string]struct{}, len(rep.Aggregate))
	for name := range rep.Aggregate {
		seen[name] = struct{}{}
	}
	for _, s := range rep.Samples {
		for name := range s.Scores {
			seen[name] = struct{}{}
		}
	}
	out := make([]string, 0, len(seen))
	for name := range seen {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

// truncate returns s unchanged when it fits, else a three-dot
// truncated copy. Used for stdout-readable row rendering without
// destroying alignment when a row happens to carry a long prompt.
func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	if max <= 3 {
		return s[:max]
	}
	return s[:max-3] + "..."
}
