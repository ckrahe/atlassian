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

var inFilename = flag.String("in", "tickets.csv", "the file to process")
var outFilename = flag.String("out", "tickets.txt", "the file to create")
var supplementalFilename = flag.String("supplemental", "", "supplemental file to process")
var hideSummary = flag.Bool("hideSummary", false, "don't show ticket summaries")
var hideOrphans = flag.Bool("hideOrphans", true, "don't show tickets without relationships")
var hideKeys = flag.String("hideKeys", "", "don't show these tickets (comma delimited)")
var wrapWidth = flag.Int("wrapWidth", 150, "Point at which to start wrapping text")

func main() {
	flag.Parse()
	inFile, err := os.Open(*inFilename)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "can't read input file (%s): %v\n", *inFilename, err)
		os.Exit(1)
	}
	outFile, err := os.Create(*outFilename)
	if err != nil {
		_ = inFile.Close()
		_, _ = fmt.Fprintf(os.Stderr, "can't create output file (%s): %v\n", *outFilename, err)
		os.Exit(1)
	}
	keysToHide := make(map[string]struct{})
	if len(*hideKeys) > 0 {
		keysToHideList := strings.Split(*hideKeys, ",")
		for _, hideKey := range keysToHideList {
			keysToHide[hideKey] = struct{}{}
		}
	}

	err = process(inFile, outFile, &keysToHide)
	_ = inFile.Close()
	_ = outFile.Close()
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "processing failed: %v\n", err)
		os.Exit(1)
	}
}

func process(inFile *os.File, outFile *os.File, keysToHide *map[string]struct{}) error {
	issues := make(map[string]IssueInfo)

	err := processSupplementalFile(keysToHide, &issues)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Problem processing supplemental: %v. Continuing.", err)
	}

	err = processFile(inFile, keysToHide, &issues)
	if err != nil {
		return fmt.Errorf("input failure: %v", err)
	}

	err = writeOutput(&issues, outFile)
	if err != nil {
		return fmt.Errorf("output failure: %v", err)
	}

	return nil
}

func processSupplementalFile(keysToHide *map[string]struct{}, issues *map[string]IssueInfo) error {
	if len(*supplementalFilename) > 0 {
		supplementalFile, err := os.Open(*supplementalFilename)
		if err != nil {
			return fmt.Errorf("couldn't open: %v", err)
		}
		err = processFile(supplementalFile, keysToHide, issues)
		if err != nil {
			return fmt.Errorf("processing problem: %v", err)
		}
		_ = supplementalFile.Close()
	}
	return nil
}

func processFile(file *os.File, keysToHide *map[string]struct{}, issues *map[string]IssueInfo) error {
	input := bufio.NewScanner(file)
	headerInfo, err := readHeader(input)
	if err != nil {
		return fmt.Errorf("header failure: %v", err)
	}
	readIssues(input, &headerInfo, keysToHide, issues)
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

func readIssues(input *bufio.Scanner, headerInfo *HeaderInfo, keysToHide *map[string]struct{}, issues *map[string]IssueInfo) {
	for input.Scan() {
		columns := strings.Split(input.Text(), ",")
		if len(columns) > headerInfo.issueKeyIdx {
			issueKey := columns[headerInfo.issueKeyIdx]
			if len(issueKey) > 0 {
				_, hideIt := (*keysToHide)[issueKey]
				if !hideIt {
					var issue IssueInfo
					issue.issueKey = issueKey
					if headerInfo.summaryIdx != -1 && len(columns) > headerInfo.summaryIdx {
						issue.summary = columns[headerInfo.summaryIdx]
					}
					if headerInfo.statusIdx != -1 && len(columns) > headerInfo.statusIdx {
						issue.status = columns[headerInfo.statusIdx]
					}
					loadBlockers(headerInfo, &columns, keysToHide, &issue, issues)
					loadBlocked(headerInfo, &columns, keysToHide, &issue, issues)
					(*issues)[issue.issueKey] = issue
				}
			}
		}
	}
}

func loadBlockers(headerInfo *HeaderInfo, columns *[]string, keysToHide *map[string]struct{}, issue *IssueInfo, issues *map[string]IssueInfo) {
	for _, idx := range headerInfo.blockerIdx {
		if len(*columns) > idx {
			blockerKey := (*columns)[idx]
			if len(blockerKey) > 0 {
				_, hideBlocker := (*keysToHide)[blockerKey]
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

func loadBlocked(headerInfo *HeaderInfo, columns *[]string, keysToHide *map[string]struct{}, issue *IssueInfo, issues *map[string]IssueInfo) {
	for _, idx := range headerInfo.blockedIdx {
		if len(*columns) > idx {
			blockedKey := (*columns)[idx]
			if len(blockedKey) > 0 {
				_, hideBlocked := (*keysToHide)[blockedKey]
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

func writeOutput(issueInfo *map[string]IssueInfo, outFile *os.File) error {
	output := bufio.NewWriter(outFile)

	// write header
	_, err := output.WriteString("@startuml\n")
	if err != nil {
		return fmt.Errorf("output failure: %v", err)
	}
	_, _ = output.WriteString(fmt.Sprintf("skinparam wrapWidth %d\n", wrapWidth))

	// write each issue as an object
	for _, issue := range *issueInfo {
		if !*hideOrphans || len(issue.blockedKeys) > 0 || len(issue.blockerKeys) > 0 {
			effectiveStatus := "unknown"
			if len(issue.status) > 0 {
				effectiveStatus = issue.status
			}
			_, _ = output.WriteString(fmt.Sprintf("object %s {\n", normalizeKey(issue.issueKey)))
			_, _ = output.WriteString(fmt.Sprintf("  %s\n", strings.ToUpper(effectiveStatus)))
			if !*hideSummary && len(issue.summary) > 0 {
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
