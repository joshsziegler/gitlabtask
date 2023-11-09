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

// ListAllUsers for the provided Gitlab Server.
func ListAllUsers(git *gitlab.Client) {
	users, err := GetAllUsers(git)
	if err != nil {
		log.Printf("Users: %s\n", err)
	} else {
		fmt.Printf("Users (%d):\n", len(users))
		for _, u := range users {
			fmt.Printf("    - %s\n", u.Name)
		}
	}
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

	webServer, err := NewServer(git, projectID)
	if err != nil {
		log.Fatalf("could not create web server: %w", err)
	}
	webServer.Listen()
}
