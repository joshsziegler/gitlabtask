package main

import (
	"fmt"
	"net/http"

	"github.com/xanzy/go-gitlab"
)

// GetAllOpenIssues from the Gitlab instance for the specific project.
func GetAllOpenIssues(git *gitlab.Client, projectID int) (issues []*gitlab.Issue, err error) {
	opt := &gitlab.ListProjectIssuesOptions{
		ListOptions: gitlab.ListOptions{
			PerPage: 100, // Gitlab's Max per page is 100
			Page:    1,
		},
		State: gitlab.String("opened"),
	}
	for { // Get each page of results and add to the full set of issues
		issueList, resp, err := git.Issues.ListProjectIssues(projectID, opt)
		if err != nil {
			return issues, fmt.Errorf("error retrieving issues from Gitlab: %s", err)
		}
		if resp.StatusCode != http.StatusOK {
			return issues, fmt.Errorf("unexpected response from Gitlab: %d %v", resp.StatusCode, resp.Response)
		}
		issues = append(issues, issueList...)

		// Get next page or exit if this was the last
		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}
	return
}

func GroupIssues(issues []*gitlab.Issue, labelOrder []string) (map[string][]*gitlab.Issue, error) {
	issueGroups := make(map[string][]*gitlab.Issue)
	// For each label, iterate through each ticket and put unclaimed ones in the first label's bucket that they match
	for _, label := range labelOrder {
		for j := len(issues) - 1; j >= 0; j-- {
			i := issues[j]
			if contains(i.Labels, label) {
				issueGroups[label] = append(issueGroups[label], i) // Add the issue to the right key
				issues = append(issues[:j], issues[j+1:]...)       // Delete the issue from the list
			}
		}
	}
	// All tickets that are not in one of the labels should still be shown as "New" unless it is in excluded label.
	for _, i := range issues {
		issueGroups["Unsorted"] = append(issueGroups["Unsorted"], i)
	}
	return issueGroups, nil
}

// UpdateIssueLables allows you to remove one label and/or add another to a ticket.
func UpdateIssueLabels(git *gitlab.Client, projectID int, issueID int, addLabel *string, delLabel *string) (*gitlab.Issue, error) {
	opt := &gitlab.UpdateIssueOptions{}
	if addLabel != nil {
		var labels gitlab.Labels = []string{*addLabel}
		opt.AddLabels = &labels
	}
	if delLabel != nil {
		var labels gitlab.Labels = []string{*delLabel}
		opt.RemoveLabels = &labels
	}
	issue, _, err := git.Issues.UpdateIssue(projectID, issueID, opt)
	return issue, err
}
