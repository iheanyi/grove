// Package github provides GitHub integration via the gh CLI
package github

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

// PRInfo contains pull request information
type PRInfo struct {
	Number int    `json:"number"`
	Title  string `json:"title"`
	URL    string `json:"url"`
	State  string `json:"state"`
}

// CIStatus represents CI status for a branch
type CIStatus struct {
	State      string // success, failure, pending, none
	Conclusion string // success, failure, canceled, skipped, etc.
	URL        string
}

// BranchInfo contains GitHub info for a branch
type BranchInfo struct {
	PR *PRInfo
	CI *CIStatus
}

// ghCLIAvailable checks if gh CLI is installed and authenticated
func ghCLIAvailable() bool {
	cmd := exec.Command("gh", "auth", "status")
	return cmd.Run() == nil
}

// GetBranchInfo fetches PR and CI info for a branch
func GetBranchInfo(branch string) *BranchInfo {
	if !ghCLIAvailable() {
		return nil
	}

	info := &BranchInfo{}

	// Get PR info
	info.PR = getPRForBranch(branch)

	// Get CI status
	info.CI = getCIStatus(branch)

	return info
}

// GetBranchInfoBatch fetches info for multiple branches efficiently
func GetBranchInfoBatch(branches []string) map[string]*BranchInfo {
	result := make(map[string]*BranchInfo)

	if !ghCLIAvailable() {
		return result
	}

	for _, branch := range branches {
		result[branch] = GetBranchInfo(branch)
	}

	return result
}

func getPRForBranch(branch string) *PRInfo {
	// Use gh pr list to find PR for this branch
	cmd := exec.Command("gh", "pr", "list",
		"--head", branch,
		"--json", "number,title,url,state",
		"--limit", "1")

	output, err := cmd.Output()
	if err != nil {
		return nil
	}

	var prs []PRInfo
	if err := json.Unmarshal(output, &prs); err != nil {
		return nil
	}

	if len(prs) == 0 {
		return nil
	}

	return &prs[0]
}

func getCIStatus(branch string) *CIStatus {
	// Get the latest commit SHA for the branch
	cmd := exec.Command("git", "rev-parse", branch)
	shaOutput, err := cmd.Output()
	if err != nil {
		return nil
	}
	sha := strings.TrimSpace(string(shaOutput))

	// Use gh api to get check runs for the commit
	cmd = exec.Command("gh", "api",
		"repos/{owner}/{repo}/commits/"+sha+"/check-runs",
		"--jq", ".check_runs | map({name, status, conclusion}) | first")

	output, err := cmd.Output()
	if err != nil {
		// Try status API instead (for older status checks)
		return getCIStatusFromStatus(sha)
	}

	var checkRun struct {
		Name       string `json:"name"`
		Status     string `json:"status"`
		Conclusion string `json:"conclusion"`
	}

	if err := json.Unmarshal(output, &checkRun); err != nil {
		return nil
	}

	status := &CIStatus{}
	if checkRun.Status == "completed" {
		status.State = checkRun.Conclusion
		status.Conclusion = checkRun.Conclusion
	} else if checkRun.Status != "" {
		status.State = "pending"
	}

	return status
}

func getCIStatusFromStatus(sha string) *CIStatus {
	// Fallback to combined status API
	cmd := exec.Command("gh", "api",
		"repos/{owner}/{repo}/commits/"+sha+"/status",
		"--jq", ".state")

	output, err := cmd.Output()
	if err != nil {
		return nil
	}

	state := strings.TrimSpace(string(output))
	if state == "" {
		return nil
	}

	return &CIStatus{
		State: state,
	}
}

// FormatCIStatus returns a colored status indicator
func FormatCIStatus(ci *CIStatus) string {
	if ci == nil {
		return ""
	}

	switch ci.State {
	case "success":
		return "✓"
	case "failure":
		return "✗"
	case "pending":
		return "◐"
	case "cancelled", "skipped": //nolint:misspell // GitHub API uses British spelling
		return "○"
	default:
		return ""
	}
}

// FormatPRInfo returns a short PR reference
func FormatPRInfo(pr *PRInfo) string {
	if pr == nil {
		return ""
	}
	return fmt.Sprintf("#%d", pr.Number)
}
