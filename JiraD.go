package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strings"
)

type HeaderInfo struct {
	issueKeyIdx int
	summaryIdx  int
	statusIdx   int
	blockedIdx  []int
	blockerIdx  []int
}

type IssueInfo struct {
	issueKey    string
	summary     string
	status      string
	blockedKeys []string
	blockerKeys []string
}

type Options struct {
	inFilename           string
	outFilename          string
	supplementalFilename string
	hideSummary          bool
	hideOrphans          bool
	hideKeys             map[string]struct{}
	showKeys             map[string]struct{}
	highlightKeys        map[string]struct{}
	highlightColor       string
	wrapWidth            int
}

func main() {
	options := loadOptions()
	inFile, err := os.Open(options.inFilename)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "can't read input file (%s): %v\n", options.inFilename, err)
		os.Exit(1)
	}
	outFile, err := os.Create(options.outFilename)
	if err != nil {
		_ = inFile.Close()
		_, _ = fmt.Fprintf(os.Stderr, "can't create output file (%s): %v\n", options.outFilename, err)
		os.Exit(1)
	}

	err = process(inFile, outFile, options)
	_ = inFile.Close()
	_ = outFile.Close()
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "processing failed: %v\n", err)
		os.Exit(1)
	}
}

func loadOptions() Options {
	inFilename := flag.String("in", "tickets.csv", "the file to process")
	outFilename := flag.String("out", "tickets.txt", "the file to create")
	supplementalFilename := flag.String("supplemental", "", "supplemental file to process")
	hideSummary := flag.Bool("hideSummary", false, "don't show ticket summaries")
	hideOrphans := flag.Bool("hideOrphans", true, "don't show tickets without relationships")
	hideKeys := flag.String("hideKeys", "", "don't show these tickets (comma delimited)")
	showKeys := flag.String("showKeys", "", "always show these tickets (comma delimited)")
	highlightKeys := flag.String("highlightKeys", "", "highlight these tickets (comma delimited)")
	highlightColor := flag.String("highlightColor", "paleGreen", "color for highlightKeys")
	wrapWidth := flag.Int("wrapWidth", 150, "Point at which to start wrapping text")
	flag.Parse()

	var options Options
	options.inFilename = *inFilename
	options.outFilename = *outFilename
	options.supplementalFilename = *supplementalFilename
	options.hideSummary = *hideSummary
	options.hideOrphans = *hideOrphans
	options.hideKeys = parseKeys(*hideKeys)
	options.showKeys = parseKeys(*showKeys)
	options.highlightKeys = parseKeys(*highlightKeys)
	options.highlightColor = *highlightColor
	options.wrapWidth = *wrapWidth

	return options
}

func process(inFile *os.File, outFile *os.File, options Options) error {
	issues := make(map[string]IssueInfo)

	err := processSupplementalFile(options, &issues)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Problem processing supplemental: %v. Continuing.", err)
	}

	err = processFile(inFile, options, &issues)
	if err != nil {
		return fmt.Errorf("input failure: %v", err)
	}

	err = writeOutput(&issues, outFile, options)
	if err != nil {
		return fmt.Errorf("output failure: %v", err)
	}

	return nil
}

func processSupplementalFile(options Options, issues *map[string]IssueInfo) error {
	if len(options.supplementalFilename) > 0 {
		supplementalFile, err := os.Open(options.supplementalFilename)
		if err != nil {
			return fmt.Errorf("couldn't open: %v", err)
		}
		err = processFile(supplementalFile, options, issues)
		if err != nil {
			return fmt.Errorf("processing problem: %v", err)
		}
		_ = supplementalFile.Close()
	}
	return nil
}

func processFile(file *os.File, options Options, issues *map[string]IssueInfo) error {
	input := bufio.NewScanner(file)
	headerInfo, err := readHeader(input)
	if err != nil {
		return fmt.Errorf("header failure: %v", err)
	}
	readIssues(input, &headerInfo, options, issues)
	return nil
}

func readHeader(input *bufio.Scanner) (HeaderInfo, error) {
	var headerInfo HeaderInfo
	headerInfo.issueKeyIdx = -1
	headerInfo.summaryIdx = -1
	headerInfo.statusIdx = -1

	input.Scan()
	columns := strings.Split(input.Text(), ",")
	for i, col := range columns {
		switch col {
		case "Issue key":
			headerInfo.issueKeyIdx = i

		case "Summary":
			headerInfo.summaryIdx = i

		case "Status":
			headerInfo.statusIdx = i

		case "Inward issue link (Blocks)":
			headerInfo.blockerIdx = append(headerInfo.blockerIdx, i)

		case "Outward issue link (Blocks)":
			headerInfo.blockedIdx = append(headerInfo.blockedIdx, i)
		}
	}
	if headerInfo.issueKeyIdx == -1 {
		return headerInfo, fmt.Errorf("'Issue key' not found\n")
	}

	return headerInfo, nil
}

func readIssues(input *bufio.Scanner, headerInfo *HeaderInfo, options Options, issues *map[string]IssueInfo) {
	for input.Scan() {
		columns := strings.Split(input.Text(), ",")
		if len(columns) > headerInfo.issueKeyIdx {
			issueKey := strings.TrimSpace(columns[headerInfo.issueKeyIdx])
			if len(issueKey) > 0 {
				_, hideIt := (options.hideKeys)[issueKey]
				_, showIt := (options.showKeys)[issueKey]
				if showIt || !hideIt {
					var issue IssueInfo
					issue.issueKey = issueKey
					if headerInfo.summaryIdx != -1 && len(columns) > headerInfo.summaryIdx {
						issue.summary = columns[headerInfo.summaryIdx]
					}
					if headerInfo.statusIdx != -1 && len(columns) > headerInfo.statusIdx {
						issue.status = columns[headerInfo.statusIdx]
					}
					loadBlockers(headerInfo, &columns, options, &issue, issues)
					loadBlocked(headerInfo, &columns, options, &issue, issues)
					(*issues)[issue.issueKey] = issue
				}
			}
		}
	}
}

func loadBlockers(headerInfo *HeaderInfo, columns *[]string, options Options, issue *IssueInfo, issues *map[string]IssueInfo) {
	for _, idx := range headerInfo.blockerIdx {
		if len(*columns) > idx {
			blockerKey := (*columns)[idx]
			if len(blockerKey) > 0 {
				_, hideBlocker := (options.hideKeys)[blockerKey]
				if !hideBlocker {
					issue.blockerKeys = append(issue.blockerKeys, blockerKey)
					_, ok := (*issues)[blockerKey]
					if !ok {
						var blocker IssueInfo
						blocker.issueKey = blockerKey
						blocker.blockedKeys = append(blocker.blockerKeys, issue.issueKey)
						(*issues)[blockerKey] = blocker
					}
				}
			}
		}
	}
}

func loadBlocked(headerInfo *HeaderInfo, columns *[]string, options Options, issue *IssueInfo, issues *map[string]IssueInfo) {
	for _, idx := range headerInfo.blockedIdx {
		if len(*columns) > idx {
			blockedKey := (*columns)[idx]
			if len(blockedKey) > 0 {
				_, hideBlocked := (options.hideKeys)[blockedKey]
				if !hideBlocked {
					issue.blockedKeys = append(issue.blockedKeys, blockedKey)
					_, ok := (*issues)[blockedKey]
					if !ok {
						var blocked IssueInfo
						blocked.issueKey = blockedKey
						blocked.blockerKeys = append(blocked.blockerKeys, issue.issueKey)
						(*issues)[blockedKey] = blocked
					}
				}
			}
		}
	}
}

func writeOutput(issueInfo *map[string]IssueInfo, outFile *os.File, options Options) error {
	output := bufio.NewWriter(outFile)

	// write header
	_, err := output.WriteString("@startuml\n")
	if err != nil {
		return fmt.Errorf("output failure: %v", err)
	}
	_, _ = output.WriteString(fmt.Sprintf("skinparam wrapWidth %d\n", options.wrapWidth))

	// write each issue as an object
	for _, issue := range *issueInfo {
		_, showIt := (options.showKeys)[issue.issueKey]
		if showIt || !options.hideOrphans || len(issue.blockedKeys) > 0 || len(issue.blockerKeys) > 0 {
			effectiveStatus := "unknown"
			if len(issue.status) > 0 {
				effectiveStatus = issue.status
			}
			_, _ = output.WriteString(fmt.Sprintf("object %s %s {\n", normalizeKey(issue.issueKey),
				getHighlight(issue.issueKey, options)))
			_, _ = output.WriteString(fmt.Sprintf("  %s\n", strings.ToUpper(effectiveStatus)))
			if !options.hideSummary && len(issue.summary) > 0 {
				_, _ = output.WriteString(fmt.Sprintf("  %s\n", issue.summary))
			}
			_, _ = output.WriteString("}\n")
		}
	}
	// write each relationship
	for _, issue := range *issueInfo {
		for _, blockedKey := range issue.blockedKeys {
			_, _ = output.WriteString(fmt.Sprintf("%s <|-- %s\n", normalizeKey(issue.issueKey), normalizeKey(blockedKey)))
		}
	}
	// write end
	_, _ = output.WriteString("@enduml\n")

	err = output.Flush()
	if err != nil {
		return fmt.Errorf("couldn't flush: %v\n", err)
	}
	return nil
}

func normalizeKey(key string) string {
	return strings.ReplaceAll(key, "-", "")
}

func parseKeys(keys string) map[string]struct{} {
	keyMap := make(map[string]struct{})

	if len(keys) > 0 {
		keyMap = make(map[string]struct{})
		keysList := strings.Split(keys, ",")
		for _, key := range keysList {
			keyMap[key] = struct{}{}
		}
	}

	return keyMap
}

func getHighlight(key string, options Options) string {
	var highlight string
	_, highlightIt := (options.highlightKeys)[key]
	if highlightIt {
		highlight = fmt.Sprintf("#%s", options.highlightColor)
	} else {
		highlight = ""
	}
	return highlight
}
