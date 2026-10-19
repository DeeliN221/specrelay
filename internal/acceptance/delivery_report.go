// Package acceptance renders and validates reviewer-facing delivery reports.
package acceptance

import (
	"bytes"
	"fmt"
	"github.com/DeeliN221/specrelay/internal/i18n"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)

var (
	weakReviewerText = regexp.MustCompile(`(?i)^\s*(ok|pass(?:ed)?|success|successful|verified|done|todo|tbd|n/?a|测试通过|验证通过|命令成功|运行成功|功能正常|没有问题|无问题|见日志|见截图|最终演示|占位)\s*[.!。]?\s*$`)
	keyValueOnlyLine = regexp.MustCompile(`^[A-Za-z0-9_.-]+=[^\s]+$`)
	commandOnlyText  = regexp.MustCompile(`(?i)^\s*(go\s+test|pytest|pnpm|npm|npx|cargo\s+test|git\s+|curl\s+|make\s+test)(?:\s+.*)?$`)
)

var reviewerInternalTerms = []string{
	"run_id",
	"content_hash",
	"contract_hash",
	"required_evidence",
	"submission_evidence",
	"delivery_base_head",
	"exit code",
	"runtime",
	"artifact",
	"acceptance",
	"content hash",
	"运行时",
	"验收合同",
	"门禁",
	"检查点",
	"测试结果",
}

// DeliveryObservation records what final independent QA actually saw for one user scenario.
type DeliveryObservation struct {
	ScenarioID string
	Observed   string
}

// ValidateUserFacingText rejects placeholders and implementation-only prose that cannot guide a reviewer.
func ValidateUserFacingText(label, text string) error {
	trimmed := strings.TrimSpace(text)
	if utf8.RuneCountInString(trimmed) < 6 {
		return i18n.Errorf("%s must describe specific user-visible behavior", label)
	}
	if strings.ContainsAny(trimmed, "\r\n") {
		return i18n.Errorf("%s must be a clear, readable statement", label)
	}
	if weakReviewerText.MatchString(trimmed) || commandOnlyText.MatchString(trimmed) {
		return i18n.Errorf("%s must not contain only test results, commands, or placeholders: %q", label, text)
	}
	lower := strings.ToLower(trimmed)
	for _, term := range reviewerInternalTerms {
		if strings.Contains(lower, term) {
			return i18n.Errorf("%s must not substitute internal fields for user-visible results: %s", label, term)
		}
	}
	return nil
}

// validateDeliveryReportContract requires benefits, review steps, evidence, and paired repair comparisons.
func validateDeliveryReportContract(report DeliveryReport, evidenceByID map[string]Evidence) error {
	if len(report.UserBenefits) == 0 {
		return i18n.Errorf("delivery_report.user_benefits must describe at least one user benefit or improvement")
	}
	for i, benefit := range report.UserBenefits {
		if err := ValidateUserFacingText(i18n.Sprintf("delivery_report.user_benefits[%d]", i), benefit); err != nil {
			return err
		}
	}
	for i, prerequisite := range report.Prerequisites {
		if err := ValidateUserFacingText(i18n.Sprintf("delivery_report.prerequisites[%d]", i), prerequisite); err != nil {
			return err
		}
	}
	for i, limit := range report.KnownLimits {
		if err := ValidateUserFacingText(i18n.Sprintf("delivery_report.known_limits[%d]", i), limit); err != nil {
			return err
		}
	}
	if len(report.Scenarios) == 0 {
		return i18n.Errorf("delivery_report.scenarios must include at least one user acceptance scenario")
	}
	seenScenario := map[string]bool{}
	hasDemo := false
	for i, scenario := range report.Scenarios {
		prefix := i18n.Sprintf("delivery_report.scenarios[%d]", i)
		if strings.TrimSpace(scenario.ID) == "" || strings.TrimSpace(scenario.Title) == "" {
			return i18n.Errorf("%s.id and title must not be empty", prefix)
		}
		if seenScenario[scenario.ID] {
			return i18n.Errorf("%s.id is duplicated: %q", prefix, scenario.ID)
		}
		seenScenario[scenario.ID] = true
		if err := ValidateUserFacingText(prefix+".user_value", scenario.UserValue); err != nil {
			return err
		}
		if len(scenario.Steps) == 0 {
			return i18n.Errorf("%s.steps must include at least one step a reviewer can perform", prefix)
		}
		for j, step := range scenario.Steps {
			if err := ValidateUserFacingText(i18n.Sprintf("%s.steps[%d].action", prefix, j), step.Action); err != nil {
				return err
			}
			if err := ValidateUserFacingText(i18n.Sprintf("%s.steps[%d].expected", prefix, j), step.Expected); err != nil {
				return err
			}
		}
		evidenceIDs, err := validateDeliveryEvidenceIDs(prefix+".evidence_ids", scenario.EvidenceIDs, evidenceByID)
		if err != nil {
			return err
		}
		for id := range evidenceIDs {
			if err := ValidateUserFacingText(prefix+".evidence_purpose", evidenceByID[id].Purpose); err != nil {
				return err
			}
			if evidenceByID[id].Kind == "demo_video" {
				hasDemo = true
			}
		}
		if scenario.Comparison != nil {
			if err := validateDeliveryComparison(prefix+".comparison", *scenario.Comparison, evidenceIDs, evidenceByID); err != nil {
				return err
			}
		}
	}
	if !hasDemo {
		return i18n.Errorf("at least one delivery_report scenario must directly reference demo_video")
	}
	return nil
}

// validateDeliveryEvidenceIDs binds a reviewer scenario to declared runtime evidence.
func validateDeliveryEvidenceIDs(label string, ids []string, evidenceByID map[string]Evidence) (map[string]bool, error) {
	if len(ids) == 0 {
		return nil, i18n.Errorf("%s must reference at least one directly reviewable evidence item", label)
	}
	seen := map[string]bool{}
	for i, id := range ids {
		if _, ok := evidenceByID[id]; !ok {
			return nil, i18n.Errorf("%s[%d] references unknown required_evidence: %q", label, i, id)
		}
		if seen[id] {
			return nil, i18n.Errorf("%s[%d] is duplicated: %q", label, i, id)
		}
		seen[id] = true
	}
	return seen, nil
}

// validateDeliveryComparison requires distinct, understandable before and after proof for a repair claim.
func validateDeliveryComparison(label string, comparison DeliveryComparison, scenarioEvidence map[string]bool, evidenceByID map[string]Evidence) error {
	if err := ValidateUserFacingText(label+".before", comparison.Before); err != nil {
		return err
	}
	if err := ValidateUserFacingText(label+".after", comparison.After); err != nil {
		return err
	}
	if normalizedReviewerClaim(comparison.Before) == normalizedReviewerClaim(comparison.After) {
		return i18n.Errorf("%s before and after must describe different user-visible behavior", label)
	}
	if comparison.BeforeEvidenceID == "" || comparison.AfterEvidenceID == "" {
		return i18n.Errorf("%s must provide both before_evidence_id and after_evidence_id", label)
	}
	if comparison.BeforeEvidenceID == comparison.AfterEvidenceID {
		return i18n.Errorf("%s before and after evidence must be different files", label)
	}
	for field, id := range map[string]string{
		"before_evidence_id": comparison.BeforeEvidenceID,
		"after_evidence_id":  comparison.AfterEvidenceID,
	} {
		if _, ok := evidenceByID[id]; !ok {
			return i18n.Errorf("%s.%s references unknown required_evidence: %q", label, field, id)
		}
		if !scenarioEvidence[id] {
			return i18n.Errorf("%s.%s must also be included in the scenario evidence_ids: %q", label, field, id)
		}
	}
	return nil
}

// RenderDeliveryReport creates the canonical human review entry from the sealed contract and final QA observations.
func RenderDeliveryReport(contract Contract, changeName string, observations []DeliveryObservation) ([]byte, error) {
	if err := ValidateSubmissionEvidenceContractForChange(contract, changeName); err != nil {
		return nil, err
	}
	observedByScenario := map[string]string{}
	scenarios := map[string]bool{}
	for _, scenario := range contract.DeliveryReport.Scenarios {
		scenarios[scenario.ID] = true
	}
	for _, item := range observations {
		if !scenarios[item.ScenarioID] {
			return nil, i18n.Errorf("final QA references an unknown delivery scenario: %s", item.ScenarioID)
		}
		if _, duplicate := observedByScenario[item.ScenarioID]; duplicate {
			return nil, i18n.Errorf("delivery report contains a duplicate observed scenario: %s", item.ScenarioID)
		}
		if err := ValidateUserFacingText("delivery report observed result", item.Observed); err != nil {
			return nil, err
		}
		observedByScenario[item.ScenarioID] = strings.TrimSpace(item.Observed)
	}
	report := *contract.DeliveryReport
	archiveByID := submissionArchivePaths(contract.SubmissionEvidence)
	evidenceIndex := evidenceByID(contract.RequiredEvidence)
	var out strings.Builder
	fmt.Fprintf(&out, "# %s Delivery report\n\n", changeName)
	out.WriteString("## User benefits\n\n")
	for _, benefit := range report.UserBenefits {
		fmt.Fprintf(&out, "- %s\n", strings.TrimSpace(benefit))
	}
	out.WriteString("\n## How to verify\n\n### Prerequisites\n\n")
	if len(report.Prerequisites) == 0 {
		out.WriteString("- No special setup is required; use an account and data with normal access to this feature.\n")
	} else {
		for _, prerequisite := range report.Prerequisites {
			fmt.Fprintf(&out, "- %s\n", strings.TrimSpace(prerequisite))
		}
	}
	for index, scenario := range report.Scenarios {
		observed, ok := observedByScenario[scenario.ID]
		if !ok {
			return nil, i18n.Errorf("final QA is missing an actual observation for delivery scenario %q", scenario.ID)
		}
		fmt.Fprintf(&out, "\n### %d. %s\n\n", index+1, strings.TrimSpace(scenario.Title))
		fmt.Fprintf(&out, "- User value: %s\n", strings.TrimSpace(scenario.UserValue))
		for stepIndex, step := range scenario.Steps {
			fmt.Fprintf(&out, "- Step %d: %s\n", stepIndex+1, strings.TrimSpace(step.Action))
			fmt.Fprintf(&out, "  - Expected: %s\n", strings.TrimSpace(step.Expected))
		}
		fmt.Fprintf(&out, "- Observed: %s\n", observed)
		fmt.Fprintf(&out, "- Evidence: %s\n", deliveryEvidenceLinks(scenario.EvidenceIDs, archiveByID, evidenceIndex))
	}
	out.WriteString("\n## Before and after\n")
	hasComparison := false
	for _, scenario := range report.Scenarios {
		if scenario.Comparison == nil {
			continue
		}
		hasComparison = true
		fmt.Fprintf(&out, "\n### %s\n\n", strings.TrimSpace(scenario.Title))
		fmt.Fprintf(&out, "- Before: %s (%s)\n", strings.TrimSpace(scenario.Comparison.Before), deliveryEvidenceLink(scenario.Comparison.BeforeEvidenceID, "View before evidence", archiveByID))
		fmt.Fprintf(&out, "- After: %s (%s)\n", strings.TrimSpace(scenario.Comparison.After), deliveryEvidenceLink(scenario.Comparison.AfterEvidenceID, "View after evidence", archiveByID))
	}
	if !hasComparison {
		out.WriteString("\nThis is a new capability; verify the user scenarios and visible outcomes above.\n")
	}
	out.WriteString("\n## Reviewable evidence\n\n")
	for _, item := range contract.SubmissionEvidence {
		purpose := strings.TrimSpace(evidenceIndex[item.EvidenceID].Purpose)
		fmt.Fprintf(&out, "- %s\n", deliveryEvidenceLink(item.EvidenceID, purpose, archiveByID))
	}
	out.WriteString("\n## Known limitations\n\n")
	if len(report.KnownLimits) == 0 {
		out.WriteString("- No additional limitations.\n")
	} else {
		for _, limit := range report.KnownLimits {
			fmt.Fprintf(&out, "- %s\n", strings.TrimSpace(limit))
		}
	}
	return []byte(out.String()), nil
}

// normalizedReviewerClaim removes cosmetic differences before comparing before/after descriptions.
func normalizedReviewerClaim(text string) string {
	var out strings.Builder
	for _, r := range strings.ToLower(strings.TrimSpace(text)) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			out.WriteRune(r)
		}
	}
	return out.String()
}

// ValidateDeliveryReportFileForChange ensures the tracked report contains the promised user guidance and evidence links.
func ValidateDeliveryReportFileForChange(projectRoot string, contract Contract, changeName string) error {
	relative := filepath.ToSlash(filepath.Join("tests", "evidence", "proposals", changeName, "DELIVERY.md"))
	if err := validateSubmissionArchiveFile(projectRoot, relative); err != nil {
		return err
	}
	data, err := os.ReadFile(filepath.Join(projectRoot, filepath.FromSlash(relative)))
	if err != nil {
		return err
	}
	if !utf8.Valid(data) || len(bytes.TrimSpace(data)) < 256 {
		return i18n.Errorf("%s must be a complete, readable UTF-8 delivery report", relative)
	}
	text := string(data)
	// Accept existing translated reports while generating English by default.
	for _, headings := range [][2]string{
		{"## User benefits", "## 这次给用户带来了什么"},
		{"## How to verify", "## 如何验收"},
		{"## Before and after", "## 修复前后对比"},
		{"## Reviewable evidence", "## 可直接查看的证据"},
		{"## Known limitations", "## 已知限制"},
	} {
		if !strings.Contains(text, headings[0]) && !strings.Contains(text, headings[1]) {
			return i18n.Errorf("%s is missing a review section: %s", relative, headings[0])
		}
	}
	report := *contract.DeliveryReport
	for _, benefit := range report.UserBenefits {
		if !strings.Contains(text, strings.TrimSpace(benefit)) {
			return i18n.Errorf("%s is missing a user benefit: %s", relative, benefit)
		}
	}
	for _, scenario := range report.Scenarios {
		for label, expected := range map[string]string{
			"title":      scenario.Title,
			"user_value": scenario.UserValue,
		} {
			if !strings.Contains(text, strings.TrimSpace(expected)) {
				return i18n.Errorf("%s is missing scenario %s: %s", relative, label, expected)
			}
		}
		for _, step := range scenario.Steps {
			if !strings.Contains(text, strings.TrimSpace(step.Action)) || !strings.Contains(text, strings.TrimSpace(step.Expected)) {
				return i18n.Errorf("%s does not include all actions and visible outcomes for scenario %q", relative, scenario.Title)
			}
		}
	}
	for _, item := range contract.SubmissionEvidence {
		link := deliveryEvidenceRelativePath(item.ArchivePath)
		if !strings.Contains(text, link) && !strings.Contains(text, deliveryURLPath(link)) {
			return i18n.Errorf("%s does not link evidence %q", relative, item.EvidenceID)
		}
	}
	observed := regexp.MustCompile(`(?m)^- (?:Observed: |实测：)(.+)$`).FindAllStringSubmatch(text, -1)
	if len(observed) != len(report.Scenarios) {
		return i18n.Errorf("%s must include one actual observation per user acceptance scenario", relative)
	}
	for i, match := range observed {
		if err := ValidateUserFacingText(i18n.Sprintf("%s observed result[%d]", relative, i), match[1]); err != nil {
			return err
		}
	}
	return nil
}

// submissionArchivePaths indexes committed evidence by contract ID.
func submissionArchivePaths(items []SubmissionEvidence) map[string]string {
	paths := make(map[string]string, len(items))
	for _, item := range items {
		paths[item.EvidenceID] = item.ArchivePath
	}
	return paths
}

// deliveryEvidenceLinks renders multiple directly clickable evidence files.
func deliveryEvidenceLinks(ids []string, archiveByID map[string]string, evidenceByID map[string]Evidence) string {
	links := make([]string, 0, len(ids))
	for _, id := range ids {
		links = append(links, deliveryEvidenceLink(id, evidenceByID[id].Purpose, archiveByID))
	}
	return strings.Join(links, ", ")
}

// deliveryEvidenceLink renders one artifact using a reviewer-facing label instead of an internal ID.
func deliveryEvidenceLink(id, label string, archiveByID map[string]string) string {
	relative := deliveryEvidenceRelativePath(archiveByID[id])
	return i18n.Sprintf("[%s](%s)", strings.TrimSpace(label), deliveryURLPath(relative))
}

// deliveryEvidenceRelativePath removes the stable proposal package prefix from an archive path.
func deliveryEvidenceRelativePath(archivePath string) string {
	parts := strings.Split(filepath.ToSlash(archivePath), "/")
	if len(parts) < 5 {
		return filepath.Base(filepath.FromSlash(archivePath))
	}
	return strings.Join(parts[4:], "/")
}

// deliveryURLPath escapes each path segment without obscuring the evidence filename.
func deliveryURLPath(relative string) string {
	parts := strings.Split(filepath.ToSlash(relative), "/")
	for i, part := range parts {
		parts[i] = url.PathEscape(part)
	}
	return strings.Join(parts, "/")
}

// evidenceByID indexes required evidence for report rendering.
func evidenceByID(items []Evidence) map[string]Evidence {
	index := make(map[string]Evidence, len(items))
	for _, item := range items {
		index[item.ID] = item
	}
	return index
}
