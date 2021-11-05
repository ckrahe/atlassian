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
var hideSummary = flag.Bool("hideSummary", false, "don't show ticket summaries")
var hideOrphans = flag.Bool("hideOrphans", true, "don't show tickets without relationships")
var hideKeys = flag.String("hideKeys", "", "don't show these tickets (comma delimited)")
var wrapWidth = flag.Int("wrapWidth", 150, "Point at which to start wrapping text")

func main() {
	flag.Parse()
	inFile, err := os.Open(*inFilename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "can't read input file (%s): %v\n", *inFilename, err)
		os.Exit(1)
	}
	outFile, err := os.Create(*outFilename)
	if err != nil {
		inFile.Close()
		fmt.Fprintf(os.Stderr, "can't create output file (%s): %v\n", *outFilename, err)
		os.Exit(1)
	}
	keysToHide := make(map[string]struct{})
	if len(*hideKeys) > 0 {
		keysToHideList := strings.Split(*hideKeys, ",")
		for _, hideKey := range keysToHideList {
			keysToHide[hideKey] = struct{}{}
		}
	}

	err = process(inFile, outFile, *hideSummary, *hideOrphans, keysToHide, *wrapWidth)
	inFile.Close()
	outFile.Close()
	if err != nil {
		fmt.Fprintf(os.Stderr, "processing failed: %v\n", err)
		os.Exit(1)
	}
}

func process(inFile *os.File, outFile *os.File, hideSummary bool, hideOrphans bool, keysToHide map[string]struct{}, wrapWidth int) error {
	input := bufio.NewScanner(inFile)
	output := bufio.NewWriter(outFile)
	_, err := output.WriteString("@startuml\n")
	if err != nil {
		return fmt.Errorf("output failure: %v", err)
	}
	output.WriteString(fmt.Sprintf("skinparam wrapWidth %d\n", wrapWidth))

	headerInfo := readHeader(input)
	issueInfo := readIssues(input, headerInfo, keysToHide)

	for _, issue := range issueInfo {
		if !hideOrphans || len(issue.blockedKeys) > 0 || len(issue.blockerKeys) > 0 {
			effectiveStatus := "unknown"
			if len(issue.status) > 0 {
				effectiveStatus = issue.status
			}
			output.WriteString(fmt.Sprintf("object %s {\n", normalizeKey(issue.issueKey)))
			output.WriteString(fmt.Sprintf("  %s", effectiveStatus))
			if !hideSummary && len(issue.summary) > 0 {
				output.WriteString(fmt.Sprintf(" - %s", issue.summary))
			}
			output.WriteString("\n}\n")
		}
	}

	for _, issue := range issueInfo {
		for _, blockedKey := range issue.blockedKeys {
			_, err := output.WriteString(fmt.Sprintf("%s <|-- %s\n", normalizeKey(issue.issueKey), normalizeKey(blockedKey)))
			if err != nil {
				return fmt.Errorf("output failure: %v", err)
			}
		}
	}

	_, err = output.WriteString("@enduml\n")
	if err != nil {
		return fmt.Errorf("output failure: %v", err)
	}

	err = output.Flush()
	if err != nil {
		fmt.Fprintf(os.Stderr, "output may be incomplete: %v\n", err)
	}

	return nil
}

func readHeader(input *bufio.Scanner) HeaderInfo {
	var headerInfo HeaderInfo
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
	return headerInfo
}

func readIssues(input *bufio.Scanner, headerInfo HeaderInfo, keysToHide map[string]struct{}) map[string]IssueInfo {
	issues := make(map[string]IssueInfo)
	for input.Scan() {
		columns := strings.Split(input.Text(), ",")
		issueKey := columns[headerInfo.issueKeyIdx]
		_, hideIt := keysToHide[issueKey]
		if !hideIt {
			var issue IssueInfo
			issue.issueKey = issueKey
			issue.summary = columns[headerInfo.summaryIdx]
			issue.status = columns[headerInfo.statusIdx]
			for _, idx := range headerInfo.blockerIdx {
				blockerKey := columns[idx]
				if len(blockerKey) > 0 {
					_, hideBlocker := keysToHide[blockerKey]
					if !hideBlocker {
						issue.blockerKeys = append(issue.blockerKeys, blockerKey)
						_, ok := issues[blockerKey]
						if !ok {
							var blocker IssueInfo
							blocker.issueKey = blockerKey
							blocker.blockedKeys = append(blocker.blockerKeys, issue.issueKey)
							issues[blockerKey] = blocker
						}
					}
				}
			}
			for _, idx := range headerInfo.blockedIdx {
				blockedKey := columns[idx]
				if len(blockedKey) > 0 {
					_, hideBlocked := keysToHide[blockedKey]
					if !hideBlocked {
						issue.blockedKeys = append(issue.blockedKeys, blockedKey)
						_, ok := issues[blockedKey]
						if !ok {
							var blocked IssueInfo
							blocked.issueKey = blockedKey
							blocked.blockerKeys = append(blocked.blockerKeys, issue.issueKey)
							issues[blockedKey] = blocked
						}
					}
				}
			}
			issues[issue.issueKey] = issue
		}
	}
	return issues
}

func normalizeKey(key string) string {
	return strings.ReplaceAll(key, "-", "")
}
