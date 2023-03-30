package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/xanzy/go-gitlab"
)

func main() {
	projectID := 111 // TODO: Allow to set at the CLI?
	key := os.Getenv("GITLAB_API_KEY")
	if key == "" {
		log.Fatal("GITLAB_API_KEY environment variable is required")
	}
	git, err := gitlab.NewClient(key, gitlab.WithBaseURL("https://gitlab.office.analyticsgateway.com/api/v4"))
	if err != nil {
		log.Fatalf("Error creating client: %v", err)
	}
	users, resp, err := git.Users.ListUsers(&gitlab.ListUsersOptions{})
	if err != nil {
		log.Fatalf("Error: %s\n", err.Error())
	}
	if resp.StatusCode != http.StatusOK {
		log.Fatalf("Unesexped response from Gitlab: %d %v", resp.StatusCode, resp.Response)
	}
	fmt.Printf("Users (%d):\n", len(users))
	for _, u := range users {
		fmt.Printf("    - %s\n", u.Name)
	}

	///////////////////////////////////////////////////////////////////////////

	// List issues I have created and are still open
	// opt := &gitlab.ListIssuesOptions{
	// 	ListOptions: gitlab.ListOptions{
	// 		PerPage: 100, // Gitlab's Max per page is 100
	// 		Page:    1,
	// 	},
	// 	State: gitlab.String("opened"),
	// }
	// issues, resp, err := git.Issues.ListIssues(opt)
	// if err != nil {
	// 	log.Fatalf("Error: %s\n", err.Error())
	// }
	// if resp.StatusCode != http.StatusOK {
	// 	log.Fatalf("Unesexped response from Gitlab: %d %v", resp.StatusCode, resp.Response)
	// }
	// fmt.Println("Issues:")
	// for _, i := range issues {
	// 	fmt.Printf("  - %d %s\n", i.ID, i.Title)
	// }

	///////////////////////////////////////////////////////////////////////////
	// List issues in AnalyticsGateway repo that are still open
	opt := &gitlab.ListProjectIssuesOptions{
		ListOptions: gitlab.ListOptions{
			PerPage: 100, // Gitlab's Max per page is 100
			Page:    1,
		},
		State: gitlab.String("opened"),
	}
	var allIssues []*gitlab.Issue
	for {
		issues, resp, err := git.Issues.ListProjectIssues(projectID, opt)
		if err != nil {
			log.Fatalf("Error: %s\n", err.Error())
		}
		if resp.StatusCode != http.StatusOK {
			log.Fatalf("Unesexped response from Gitlab: %d %v", resp.StatusCode, resp.Response)
		}
		allIssues = append(allIssues, issues...)

		// Get next page or exit if this was the last
		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}

	fmt.Printf("Issues (%d):\n", len(allIssues))
	for _, i := range allIssues {
		fmt.Printf("    - %d %s\n", i.IID, i.Title) // IID is the Project-specific ID
	}
}
