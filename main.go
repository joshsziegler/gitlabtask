package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/xanzy/go-gitlab"
)

// contains checks if a string is present in a slice
func contains(haystack []string, needle string) bool {
	for _, v := range haystack {
		if v == needle {
			return true
		}
	}
	return false
}

// GetEnv variable or use the default value if it is not set (i.e. empty).
func GetEnv(varName, defaultValue string) string {
	val := strings.TrimSpace(os.Getenv(varName))
	if val == "" {
		val = defaultValue
	}
	return val
}

// GetEnv variable or use the default value if it is not set (i.e. empty).
func GetEnvInt(varName string, defaultValue int) int {
	str := strings.TrimSpace(os.Getenv(varName))
	if str == "" {
		return defaultValue
	}
	val, err := strconv.Atoi(str)
	if err != nil {
		log.Fatalf("cannot convert environment variable to an integer: %s", err.Error())
	}
	return val
}

// GetAllUsers from the Gitlab instance.
func GetAllUsers(git *gitlab.Client) ([]*gitlab.User, error) {
	users, resp, err := git.Users.ListUsers(&gitlab.ListUsersOptions{})
	if err != nil {
		return nil, fmt.Errorf("error retrieving users from Gitlab: %s", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected response from Gitlab: %d %v", resp.StatusCode, resp.Response)
	}
	return users, nil
}

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

func main() {
	// Allow the project ID to be set via environment variable, but require a GITLAB_API_KEY
	projectID := GetEnvInt("GITLAB_PROJ_ID", 111)
	key := os.Getenv("GITLAB_API_KEY")
	if key == "" {
		log.Fatal("GITLAB_API_KEY environment variable is required")
	}
	git, err := gitlab.NewClient(key, gitlab.WithBaseURL("https://gitlab.office.analyticsgateway.com/api/v4"))
	if err != nil {
		log.Fatalf("Error creating client: %v", err)
	}

	// List All Users //////////////////////////////////////////////////////////
	// users, err := GetAllUsers(git)
	// if err != nil {
	// 	log.Printf("Users: %s\n", err)
	// } else {
	// 	fmt.Printf("Users (%d):\n", len(users))
	// 	for _, u := range users {
	// 		fmt.Printf("    - %s\n", u.Name)
	// 	}
	// }

	// List all open issues in the project /////////////////////////////////////
	issues, err := GetAllOpenIssues(git, projectID)
	if err != nil {
		log.Fatalf("# Issues: %s\n\n", err)
	}

	labelOrder := []string{
		"HELP!",
		"customer communication",
		"New", // -- this is a category means it has no other labels in this list
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
	// All tickets that are not in one of the labels should still be shown as "New" 
	for _, i := range issues {
		issueGroups["New"] = append(issueGroups["New"], i)
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
			if contains(i.Labels, "Type::Bug"){
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
