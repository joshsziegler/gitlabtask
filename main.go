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
	users, err := GetAllUsers(git)
	if err != nil {
		log.Printf("Users: %s\n", err)
	} else {
		fmt.Printf("Users (%d):\n", len(users))
		for _, u := range users {
			fmt.Printf("    - %s\n", u.Name)
		}
	}

	// List all open issues in the project /////////////////////////////////////
	issues, err := GetAllOpenIssues(git, projectID)
	if err != nil {
		log.Printf("Issues: %s\n", err)
	} else {
		fmt.Printf("Issues (%d):\n", len(issues))
		for _, i := range issues {
			fmt.Printf("    - %d %s\n", i.IID, i.Title) // IID is the Project-specific ID
			fmt.Printf("        - %+v\n", i.Labels)
		}
	}
}
