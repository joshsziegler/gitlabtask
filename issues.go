package main

import (
	"fmt"
	"log"
	"net/http"
	"strings"

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

// ListAllOpenIssues as Markdown from the specified project.
func ListAllOpenIssues(git *gitlab.Client, projectID int) {
	issues, err := GetAllOpenIssues(git, projectID)
	if err != nil {
		log.Fatalf("# Issues: %s\n\n", err)
	}

	labelOrder := []string{
		"HELP!",
		"customer communication",
		"Unsorted", // -- this is a category means it has no other labels in this list
		"T::23-05",
		"T::23-06",
		"T::23-07",
		"T::23-08",
		"T::23-09",
		"T::23-10",
		"T::23-11",
		"T::23-12",
		"T::24-01",
		"T::24-02",
		"T::24-03",
		"T::24-04",
		"T::24-05",
		"T::24-06",
		"T::24-07",
		"T::24-08",
		"T::24-09",
		"T::24-10",
		"T::24-11",
		"T::24-12",
		"T::Future",
		"STIG:CAT-2",
		"STIG:CAT-3",
	}
	links := []string{}
	issueGroups, err := GroupIssues(issues, labelOrder)
	if err != nil {
		log.Fatalf("error grouping issues: %w", err)
	}

	// Done sorting through issues. Print everything using Markdown
	for _, label := range labelOrder {
		fmt.Printf("## %s\n", label)
		for _, i := range issueGroups[label] {
			assignee := ""
			if i.Assignee != nil {
				parts := strings.Split(i.Assignee.Name, " ")
				for _, p := range parts {
					assignee = fmt.Sprintf("%s%s", assignee, string(p[0]))
				}
				assignee = fmt.Sprintf(" — **%s**", assignee)
			}
			// Add a bold BUG prefix if it's labeled as one
			prefix := ""
			if contains(i.Labels, "Type::Bug") {
				prefix = "**BUG**"
			}
			fmt.Printf("- [%d][%d] %s %s%s\n", i.IID, i.IID, prefix, i.Title, assignee)
			// Use Markdown formatting to save the ID-to-URL for the bottom of the doc
			links = append(links, fmt.Sprintf("[%d]: %s", i.IID, i.WebURL))
		}
	}

	fmt.Printf("\n\n\n# Links\n\n")
	for _, link := range links {
		fmt.Println(link)
	}
}
