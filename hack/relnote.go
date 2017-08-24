package main

import (
	"bufio"
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/google/go-github/github"
	"golang.org/x/net/context"
	"golang.org/x/oauth2"
)

type Item struct {
	number          int
	title           string
	summary         string
	actionsRequired string
	isDocUpdate     bool
	isMetaUpdate    bool
	isImprovement   bool
	isFeature       bool
	isBugFix        bool
	isProposal      bool
	isRefactoring   bool
}

func Info(msg string) {
	println(msg)
}

func Header(title string) {
	fmt.Printf("\n## %s\n\n", title)
}

func PanicIfError(err error) {
	if err != nil {
		panic(err)
	}
}

func capture(cmdName string, cmdArgs []string) (string, error) {
	fmt.Printf("running %s %v\n", cmdName, cmdArgs)
	cmd := exec.Command(cmdName, cmdArgs...)

	stdoutBuffer := bytes.Buffer{}

	{
		stdoutReader, err := cmd.StdoutPipe()
		if err != nil {
			return "", fmt.Errorf("failed to pipe stdout: %v", err)
		}

		stdoutScanner := bufio.NewScanner(stdoutReader)
		go func() {
			for stdoutScanner.Scan() {
				stdoutBuffer.WriteString(stdoutScanner.Text())
			}
		}()
	}

	stderrBuffer := bytes.Buffer{}
	{
		stderrReader, err := cmd.StderrPipe()
		if err != nil {
			return "", fmt.Errorf("failed to pipe stderr: %v", err)
		}

		stderrScanner := bufio.NewScanner(stderrReader)
		go func() {
			for stderrScanner.Scan() {
				stderrBuffer.WriteString(stderrScanner.Text())
			}
		}()
	}

	err := cmd.Start()
	if err != nil {
		return "", fmt.Errorf("failed to start command: %v: %s", err, stderrBuffer.String())
	}

	err = cmd.Wait()
	if err != nil {
		return "", fmt.Errorf("failed to wait command: %v: %s", err, stderrBuffer.String())
	}

	return stdoutBuffer.String(), nil
}

func filesChangedInCommit(refName string) []string {
	output, err := capture("bash", []string{"-c", fmt.Sprintf("git log -m -1 --name-only --pretty=format: %s | awk -v RS=  '{ print; exit }'", refName)})
	if err != nil {
		panic(err)
	}
	files := strings.Split(output, "\n")
	return files
}

func onlyDocsAreChanged(files []string) bool {
	all := true
	for _, file := range files {
		all = all && (strings.HasPrefix(file, "Documentation/") || strings.HasPrefix(file, "docs/"))
	}
	return all
}

func onlyTopLevelFilesAreChanged(files []string) bool {
	all := true
	for _, file := range files {
		all = all && len(strings.Split(file, "/")) == 1
	}
	return all
}

func containsAny(str string, substrs []string) bool {
	for _, sub := range substrs {
		if strings.Contains(str, sub) {
			return true
		}
	}
	return false
}

type Labels []github.Label

func (labels Labels) Contains(name string) bool {
	found := false
	for _, label := range labels {
		if label.GetName() == name {
			found = true
		}
	}
	return found
}

var errorlog *log.Logger

func init() {
	errorlog = log.New(os.Stderr, "", 0)
}

func exitWithErrorMessage(msg string) {
	errorlog.Println(msg)
	os.Exit(1)
}

func indent(orig string, num int) string {
	lines := strings.Split(orig, "\n")
	space := ""
	buf := bytes.Buffer{}
	for i := 0; i < num; i++ {
		space = space + " "
	}
	for _, line := range lines {
		buf.WriteString(fmt.Sprintf("%s%s\n", space, line))
	}
	return buf.String()
}

func generateNote(primaryMaintainer string, org string, repository string, releaseVersion string) {
	accessToken, found := os.LookupEnv("GITHUB_ACCESS_TOKEN")
	if !found {
		exitWithErrorMessage("GITHUB_ACCESS_TOKEN must be set")
	}
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: accessToken},
	)
	tc := oauth2.NewClient(ctx, ts)

	client := github.NewClient(tc)

	milestoneOpt := &github.MilestoneListOptions{
		ListOptions: github.ListOptions{PerPage: 10},
	}

	allMilestones := []*github.Milestone{}
	for {
		milestones, resp, err := client.Issues.ListMilestones(ctx, org, repository, milestoneOpt)
		PanicIfError(err)
		allMilestones = append(allMilestones, milestones...)
		if resp.NextPage == 0 {
			break
		}
		milestoneOpt.Page = resp.NextPage
	}

	milestoneNumber := -1
	for _, m := range allMilestones {
		if m.GetTitle() == releaseVersion {
			milestoneNumber = m.GetNumber()
		}
	}
	if milestoneNumber == -1 {
		exitWithErrorMessage(fmt.Sprintf("Milestone titled \"%s\" not found", releaseVersion))
	}

	opt := &github.IssueListByRepoOptions{
		ListOptions: github.ListOptions{PerPage: 10},
		State:       "closed",
		Sort:        "created",
		Direction:   "asc",
		Milestone:   fmt.Sprintf("%d", milestoneNumber),
	}

	items := []Item{}

	// list all organizations for user "mumoshu"
	var allIssues []*github.Issue
	for {
		issues, resp, err := client.Issues.ListByRepo(ctx, org, repository, opt)
		PanicIfError(err)
		for _, issue := range issues {
			if issue.PullRequestLinks == nil {
				fmt.Printf("skipping issue #%d %s\n", issue.GetNumber(), issue.GetTitle())
				continue
			}
			pr, _, err := client.PullRequests.Get(ctx, org, repository, issue.GetNumber())
			PanicIfError(err)
			if !pr.GetMerged() {
				continue
			}
			hash := pr.GetMergeCommitSHA()

			login := issue.User.GetLogin()
			num := issue.GetNumber()
			title := issue.GetTitle()
			summary := ""
			if login != primaryMaintainer {
				summary = fmt.Sprintf("#%d: %s(Thanks to @%s)", num, title, login)
			} else {
				summary = fmt.Sprintf("#%d: %s", num, title)
			}

			labels := Labels(issue.Labels)

			isRefactoring := labels.Contains("refactoring")

			fmt.Printf("analyzing #%d %s...\n", num, title)
			fmt.Printf("labels=%v\n", labels)
			changedFiles := filesChangedInCommit(hash)

			isDocUpdate := onlyDocsAreChanged(changedFiles)
			if isDocUpdate {
				fmt.Printf("%s is doc update\n", title)
			}

			isMetaUpdate := onlyTopLevelFilesAreChanged(changedFiles)
			if isMetaUpdate {
				fmt.Printf("%s is meta update\n", title)
			}

			isBugFix := labels.Contains("bug") ||
				(!isRefactoring && !isDocUpdate && !isMetaUpdate && (strings.Contains(title, "fix") || strings.Contains(title, "Fix")))

			isProposal := labels.Contains("proposal") ||
				(!isRefactoring && !isDocUpdate && !isMetaUpdate && !isBugFix && (strings.Contains(title, "proposal") || strings.Contains(title, "Proposal")))

			isImprovement := labels.Contains("improvement") ||
				(!isRefactoring && !isDocUpdate && !isMetaUpdate && !isBugFix && !isProposal && containsAny(title, []string{"improve", "Improve", "update", "Update", "bump", "Bump", "Rename", "rename"}))

			isFeature := labels.Contains("feature") ||
				(!isRefactoring && !isDocUpdate && !isMetaUpdate && !isBugFix && !isProposal && !isImprovement)

			actionsRequired := ""
			noteShouldBeAdded := false
			for _, label := range issue.Labels {
				if label.GetName() == "release-note" {
					noteShouldBeAdded = true
				}
			}
			if noteShouldBeAdded {
				body := issue.GetBody()
				splits := strings.Split(body, "**Release note**:")
				if len(splits) != 2 {
					panic(fmt.Errorf("failed to extract release note from PR body: unexpected format of PR body: it should include \"**Release note**:\" followed by note: issue=%s body=%s", title, body))
				}
				fmt.Printf("actions required(raw)=\"%s\"\n", splits[1])
				actionsRequired = strings.TrimSpace(splits[1])
				fmt.Printf("actions required(trimmed)=\"%s\"\n", actionsRequired)

				if !strings.HasPrefix(actionsRequired, "* ") {
					actionsRequired = fmt.Sprintf("* %s", actionsRequired)
				}
			}

			item := Item{
				number:          num,
				title:           title,
				summary:         summary,
				actionsRequired: actionsRequired,
				isMetaUpdate:    isMetaUpdate,
				isDocUpdate:     isDocUpdate,
				isImprovement:   isImprovement,
				isFeature:       isFeature,
				isBugFix:        isBugFix,
				isProposal:      isProposal,
				isRefactoring:   isRefactoring,
			}
			items = append(items, item)
			//Info(summary)
		}
		allIssues = append(allIssues, issues...)
		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}

	Info("# Changelog since v")

	Header("Component versions")

	println("Kubernetes: v")
	println("Etcd: v")

	Header("Actions required")
	for _, item := range items {
		if item.actionsRequired != "" {
			fmt.Printf("* #%d: %s\n%s\n", item.number, item.title, indent(item.actionsRequired, 2))
		}
	}

	Header("Features")
	for _, item := range items {
		if item.isFeature {
			Info("* " + item.summary)
		}
	}

	Header("Improvements")
	for _, item := range items {
		if item.isImprovement {
			Info("* " + item.summary)
		}
	}

	Header("Bug fixes")
	for _, item := range items {
		if item.isBugFix {
			Info("* " + item.summary)
		}
	}

	Header("Documentation")
	for _, item := range items {
		if item.isDocUpdate {
			Info("* " + item.summary)
		}
	}

	Header("Refactorings")
	for _, item := range items {
		if item.isRefactoring {
			Info("* " + item.summary)
		}
	}

	Header("Other changes")
	for _, item := range items {
		if !item.isDocUpdate && !item.isFeature && !item.isImprovement && !item.isBugFix && !item.isRefactoring {
			Info("* " + item.summary)
		}
	}
}

func main() {
	releaseVersion, found := os.LookupEnv("VERSION")
	if !found {
		exitWithErrorMessage("VERSION must be set")
	}
	generateNote("mumoshu", "kubernetes-incubator", "kube-aws", releaseVersion)
}
