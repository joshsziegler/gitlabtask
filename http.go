package main

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

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
	http.HandleFunc("/issue/labels/update", s.UpdateIssueLabels)

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
			}

			p,ul,ol {
			  margin-bottom: 0.5em;
			}

			a {
				text-decoration: none;
			}

			article {
				margin-right: auto;
				margin-left: 200px;
				margin-bottom: 4rem;
				padding: 0 1rem;
			}

			.weight-b { font-weight: bold; }
			.weight-n { font-weight: normal; }
			.style-i { font-style: italic; }
			.style-n { font-style: normal; }
			.text-right { text-align: right; }
			.text-color-slate { color: #757575; }
			.pl-1 { padding-left: 0.25rem; }
			.pl-2 { padding-left: 0.5rem; }
			.pl-4 { padding-left: 1rem; }
			.px-1 { padding-left: 0.25rem; padding-right: 0.25rem; }
			.px-2 { padding-left: 0.5rem; padding-right: 0.5rem; }
			.px-4 { padding-left: 1rem; padding-right: 1rem; }
			.ta-end {text-align: end; }
			.inline { display: inline; }
			.inline-block {display: inline-block; }
			.block {display: block; }

			td.truncate { /* Truncate td to 700px using ellipsis */
				display: block;
				width: 700px;
				overflow: hidden;
				text-overflow: ellipsis;
				white-space: nowrap;
			}

			.bug { color: #c20000 !important; }
			.underline {
				border-bottom: 1px solid #8a8a8a;
			}
			.main-nav {
				position: fixed;
				top: 0;
				left: 0;
				width: 200px;
				height: 100%;

				background-color: rgb(37,38,41);
				color: rgb(255,255,255);
			}
			.main-nav > h1,
			.main-nav > h2,
			.main-nav > h3,
			.main-nav > h4,
			.main-nav > h5 {
				color: rgb(215,215,215);
			}
			.main-nav > ul {
				margin: 0.5rem;
				padding: 0 0 0 0.25rem;
			}
			.main-nav > ul > li {
				list-style: none;
			}
			.main-nav > ul > li > a {
				color: rgba(191,191,191,90);
			}
			.main-nav > ul > li > a:hover {
				color: rgba(191,191,191,100);
			}
		</style>
	</head>
	<body>
		<nav class="main-nav">
			<h2 class="">View</h2>
			<ul>
				<li class=""><a href="/list">List</a></li>
				<li class=""><a href="/msu">Must/Should/Want</a></li>
			</ul>
		</nav>
	`
	fmt.Fprintln(w, header)
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/msw", http.StatusSeeOther)
}

func (s *Server) UpdateIssueLabels(w http.ResponseWriter, r *http.Request) {
	issueIDStr := r.PostFormValue("issueID")
	if issueIDStr == "" {
		fmt.Fprintln(w, `<h2>Requires IssueID</h2>`)
		return
	}
	issueID, err := strconv.Atoi(issueIDStr)
	if err != nil {
		fmt.Fprintf(w, `<h2>Invalid IssueID: %s</h2>`, issueIDStr)
		return
	}
	var addLabel *string
	addLabelStr := r.PostFormValue("addLabel")
	if addLabelStr != "" {
		addLabel = &addLabelStr
	}

	var delLabel *string
	delLabelStr := r.PostFormValue("delLabel")
	if delLabelStr != "" {
		delLabel = &delLabelStr
	}
	_, err = UpdateIssueLabels(s.git, s.projectID, issueID, addLabel, delLabel)
	if err != nil {
		fmt.Fprintf(w, "<h2>Error updating issue's labels: %s</h2>", err.Error())
		return
	}
	http.Redirect(w, r, "/msw", http.StatusSeeOther)
}

func (s *Server) handleIssueList(w http.ResponseWriter, r *http.Request) {
	labelOrder := []string{
		"HELP!",
		"Customer Communication",
		"Unsorted", // -- this is a category means it has no other labels in this list
		// "Design",
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
	s.printTemplateV2(w, labelOrder, false)
}

// Show all tickets sorted by deadlines then Must, Should, and Want.
func (s *Server) handleIssueByMustShouldWant(w http.ResponseWriter, r *http.Request) {
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
	s.printTemplateV2(w, labelOrder, true)
}

func (s *Server) printTemplateV1(w http.ResponseWriter, labelOrder []string) {
	writeHeader(w)

	issues, err := GetAllOpenIssues(s.git, s.projectID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	issueGroups, err := GroupIssues(issues, labelOrder)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error grouping issues: %w", err), http.StatusInternalServerError)
		return
	}

	// Done sorting through issues. Print everything using Markdown
	fmt.Fprintln(w, "<article>")
	for _, label := range labelOrder {
		fmt.Fprintf(w, `<h2 class="underline">%s</h2><ul>`, label) // TODO(JZ): Add link to the list view, sorted by creation date (oldest to newest) with only this label https://gitlab.office.analyticsgateway.com/it/scale/analytics-hub/-/issues/?sort=created_asc&state=opened&label_name%5B%5D=customer%20communication&first_page_size=20
		for _, i := range issueGroups[label] {
			assignee := ""
			if i.Assignee != nil {
				parts := strings.Split(i.Assignee.Name, " ")
				for _, p := range parts {
					assignee = fmt.Sprintf("%s%s", assignee, string(p[0]))
				}
				assignee = fmt.Sprintf("— <span class=\"weight-b\">%s<span>", assignee)
			}
			// Add a bold BUG prefix if it's labeled as one
			prefix := ""
			if contains(i.Labels, "Type::Bug") {
				prefix = "<span class=\"weight-b bug\">BUG</span>"
			}
			fmt.Fprintf(w, "<li><a href=\"%s\">%d — %s %s %s</a></li>\n", i.WebURL, i.IID, prefix, i.Title, assignee)
		}
		fmt.Fprint(w, "</ul>")
	}
	fmt.Fprintln(w, "</article>")
	fmt.Fprintln(w, "</body>\n</html>")
}

func (s *Server) printTemplateV2(w http.ResponseWriter, labelOrder []string, showPlannedMonth bool) {
	writeHeader(w)

	issues, err := GetAllOpenIssues(s.git, s.projectID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	issueGroups, err := GroupIssues(issues, labelOrder)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error grouping issues: %w", err), http.StatusInternalServerError)
		return
	}

	// Done sorting through issues. Print everything
	numCol := 5
	if showPlannedMonth {
		numCol = 6
	}
	fmt.Fprintln(w, "<article>")
	fmt.Fprintln(w, "<table>")
	for _, label := range labelOrder {
		if label == "STOPHERE" {
			break
		}
		numIssues := len(issueGroups[label])
		// TODO(JZ): Add link to the list view, sorted by creation date (oldest to newest) with only this label
		// https://gitlab.office.analyticsgateway.com/it/scale/analytics-hub/-/issues/?sort=created_asc&state=opened&label_name%5B%5D=customer%20communication&first_page_size=20
		fmt.Fprintf(w, `<tr>`)
		fmt.Fprintf(w, `<td colspan="%d"><h2 class="underline">%s <span class="weight-n style-i pl-1">%d</span></h2></td>`, numCol, label, numIssues)
		//fmt.Fprintf(w, `<td class="">Age</td>`)
		//fmt.Fprintf(w, `<td class="">Due</td>`)
		//fmt.Fprintf(w, `<td class="">Scheduled</td>`)
		fmt.Fprint(w, `</tr>`)
		if numIssues < 1 {
			fmt.Fprintf(w, `<tr><td colspan="%d"></td></tr>`, numCol)
		}

		for _, i := range issueGroups[label] {
			fmt.Fprint(w, `<tr>`)

			assignee := ""
			if i.Assignee != nil {
				parts := strings.Split(i.Assignee.Name, " ")
				for _, p := range parts {
					assignee = fmt.Sprintf("%s%s", assignee, string(p[0]))
				}
				assignee = fmt.Sprintf(`<span class="weight-b">%s<span>`, assignee)
			}
			fmt.Fprintf(w, `<td class="">%s</td>`, assignee)

			// Make ID red and bold if this ticket is labeled as a BUG
			bug := contains(i.Labels, "Type::Bug")
			titleClasses := ""
			if bug {
				titleClasses += "bug weight-b" // bug makes it red, b makes it bold
			}
			fmt.Fprintf(w, `<td class="px-4 ta-end"><span class="%s">%d</span></td>`, titleClasses, i.IID)
			fmt.Fprintf(w, `<td class="truncate"><a href="%s">%s</a></td>`, i.WebURL, i.Title)

			// Show the ticket's age in days
			daysSinceCreation := int64(time.Now().Sub(*i.CreatedAt).Hours() / 24)
			fmt.Fprintf(w, `<td class="style-i text-color-slate text-right">%d</td>`, daysSinceCreation)

			// Show the ticket's due date
			fmt.Fprintf(w, `<td><span class="weight-b px-4">%s</span></td>`, i.DueDate)

			// Show the ticket's current time label (e.g. T::23-11, T::24-01)
			if showPlannedMonth {
				for _, label := range i.Labels {
					if strings.HasPrefix(label, "T::") {
						fmt.Fprintf(w, `<td class="text-color-slate">%s</td>`, label[3:])
					}
				}
			}
			fmt.Fprint(w, "</tr>")
		}
	}
	fmt.Fprintln(w, "</table>")
	fmt.Fprintln(w, "</article>")
	fmt.Fprintln(w, "</body>\n</html>")
}
