package main

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/xanzy/go-gitlab"
)

func NewServer(git *gitlab.Client, projectID int) (*Server, error) {
	if git == nil {
		return nil, errors.New("gitlab client is required")
	}
	if projectID == 0 {
		return nil, errors.New("project ID is required")
	}
	return &Server{
		git:       git,
		projectID: projectID,
	}, nil
}

type Server struct {
	git       *gitlab.Client
	projectID int
}

func (s *Server) Listen() error {
	// Setup routes
	http.HandleFunc("/", s.handleIndex)
	http.HandleFunc("/list", s.handleIssueList)
	http.HandleFunc("/msw", s.handleIssueByMustShouldWant)

	return http.ListenAndServe(":8080", nil)
}

func writeHeader(w http.ResponseWriter) {
	header := `
	<html>
	<head>
		<style>
			*, html, body {
				font-family: "Times", "Times New Roman", "NimbusRoman", serif;
				color: #1d1d1d;
				font-size: 20px;
				line-height: 30px;
			}

			h1,h2,h3,h4,h5,h6 {
			  margin: 1em 0 0.5em;
			}

			h1,h2 {
				font-size: 31.25px;
				line-height: 48px;
				border-bottom: 1px solid #8a8a8a;
			}

			p,ul,ol {
			  margin-bottom: 0.5em;
			}

			a {
				text-decoration: none;
			}

			article {
				max-width: 900px;
				margin-right: auto;
				margin-left: auto;
				margin-bottom: 4rem;
			}

			.b { font-weight: bold; }
			.bug { color: #c20000 !important; }

			.px-1 { padding-left: 1rem; padding-right: 1rem;}
			.ta-end {text-align: end;}

			td.truncate { /* Truncate td to 700px using ellipsis */
				display: block;
				width: 700px;
				overflow: hidden;
				text-overflow: ellipsis;
				white-space: nowrap;
			}
		</style>
	</head>
	<body>
	`
	fmt.Fprintln(w, header)
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	writeHeader(w)
	fmt.Fprint(w, "<ul>")
	fmt.Fprint(w, `<li><a href="/list">List</a></li>`)
	fmt.Fprint(w, "</ul>")

	fmt.Fprintln(w, "</article>")
	fmt.Fprintln(w, "</body>")
	fmt.Fprintln(w, "</html>")
}

func (s *Server) handleIssueList(w http.ResponseWriter, r *http.Request) {
	writeHeader(w)
	issues, err := GetAllOpenIssues(s.git, s.projectID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	labelOrder := []string{
		"HELP!",
		"Customer Communication",
		"New", // -- this is a category means it has no other labels in this list
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
	issueGroups, err := GroupIssues(issues, labelOrder)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error grouping issues: %w", err), http.StatusInternalServerError)
		return
	}

	// Done sorting through issues. Print everything using Markdown
	fmt.Fprintln(w, "<article>")
	for _, label := range labelOrder {
		fmt.Fprintf(w, "<h2>%s</h2>\n<ul>", label) // TODO(JZ): Add link to the list view, sorted by creation date (oldest to newest) with only this label https://gitlab.office.analyticsgateway.com/it/scale/analytics-hub/-/issues/?sort=created_asc&state=opened&label_name%5B%5D=customer%20communication&first_page_size=20
		for _, i := range issueGroups[label] {
			assignee := ""
			if i.Assignee != nil {
				parts := strings.Split(i.Assignee.Name, " ")
				for _, p := range parts {
					assignee = fmt.Sprintf("%s%s", assignee, string(p[0]))
				}
				assignee = fmt.Sprintf("— <span class=\"b\">%s<span>", assignee)
			}
			// Add a bold BUG prefix if it's labeled as one
			prefix := ""
			if contains(i.Labels, "Type::Bug") {
				prefix = "<span class=\"b bug\">BUG</span>"
			}
			fmt.Fprintf(w, "<li><a href=\"%s\">%d — %s %s %s</a></li>\n", i.WebURL, i.IID, prefix, i.Title, assignee)
		}
		fmt.Fprint(w, "</ul>")
	}
	fmt.Fprintln(w, "</article>")
	fmt.Fprintln(w, "</body>\n</html>")
}

// Show all tickets sorted by deadlines then Must, Should, and Want.
func (s *Server) handleIssueByMustShouldWant(w http.ResponseWriter, r *http.Request) {
	writeHeader(w)
	issues, err := GetAllOpenIssues(s.git, s.projectID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	labelOrder := []string{
		"HELP!",
		"M::Must",
		"M::Should",
		"M::Want",
		"Unsorted", // -- this is a category means it has no other labels in this list
		"STOPHERE", // Hack to stop the list at this point
		"Customer Communication",
		"STIG:CAT-2",
		"STIG:CAT-3",
	}
	issueGroups, err := GroupIssues(issues, labelOrder)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error grouping issues: %w", err), http.StatusInternalServerError)
		return
	}

	// Done sorting through issues. Print everything
	fmt.Fprintln(w, "<article>")
	fmt.Fprintln(w, "<table>")
	for _, label := range labelOrder {
		if label == "STOPHERE" {
			break
		}
		fmt.Fprintf(w, `<tr><td colspan="3"><h2>%s</h2></td>`, label) // TODO(JZ): Add link to the list view, sorted by creation date (oldest to newest) with only this label https://gitlab.office.analyticsgateway.com/it/scale/analytics-hub/-/issues/?sort=created_asc&state=opened&label_name%5B%5D=customer%20communication&first_page_size=20
		for _, i := range issueGroups[label] {
			fmt.Fprint(w, `<tr>`)

			assignee := ""
			if i.Assignee != nil {
				parts := strings.Split(i.Assignee.Name, " ")
				for _, p := range parts {
					assignee = fmt.Sprintf("%s%s", assignee, string(p[0]))
				}
				assignee = fmt.Sprintf("<span class=\"b\">%s<span>", assignee)
			}
			fmt.Fprintf(w, `<td class="">%s</td>`, assignee)

			// Make ID red and bold if this ticket is labeled as a BUG
			bug := contains(i.Labels, "Type::Bug")
			if bug {
				fmt.Fprintf(w, `<td class="px-1 ta-end"><a class="bug b" href="%s">%d</td>`, i.WebURL, i.IID)
			} else {
				fmt.Fprintf(w, `<td class="px-1 ta-end"><a href="%s">%d</td>`, i.WebURL, i.IID)
			}
			fmt.Fprintf(w, `<td class="truncate">%s</a></td>`, i.Title)

			fmt.Fprint(w, "</tr>")
		}
	}
	fmt.Fprintln(w, "</table>")
	fmt.Fprintln(w, "</table>")
	fmt.Fprintln(w, "</body>\n</html>")
}
