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
	statusIdx   int
	blockedIdx  []int
	blockerIdx  []int
}

type IssueInfo struct {
	issueKey    string
	status      string
	blockedKeys []string
	blockerKeys []string
}

var inFilename = flag.String("in", "tickets.csv", "the file to process")
var outFilename = flag.String("out", "tickets.txt", "the file to create")

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

	err = process(inFile, outFile)
	inFile.Close()
	outFile.Close()
	if err != nil {
		fmt.Fprintf(os.Stderr, "processing failed: %v\n", err)
		os.Exit(1)
	}
}

func process(inFile *os.File, outFile *os.File) error {
	input := bufio.NewScanner(inFile)
	output := bufio.NewWriter(outFile)
	_, err := output.WriteString("@startuml\n")
	if err != nil {
		return fmt.Errorf("output failure: %v", err)
	}

	headerInfo := readHeader(input)
	issueInfo := readIssues(input, headerInfo)

	for _, issue := range issueInfo {
		_, err := output.WriteString(fmt.Sprintf("object %s {\n \"%s\" \n}\n", normalizeKey(issue.issueKey), issue.status))
		if err != nil {
			return fmt.Errorf("output failure: %v", err)
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

func readIssues(input *bufio.Scanner, headerInfo HeaderInfo) []IssueInfo {
	var issues []IssueInfo
	for input.Scan() {
		var issue IssueInfo
		columns := strings.Split(input.Text(), ",")
		issue.issueKey = columns[headerInfo.issueKeyIdx]
		issue.status = columns[headerInfo.statusIdx]
		for _, idx := range headerInfo.blockerIdx {
			key := columns[idx]
			if len(key) > 0 {
				issue.blockerKeys = append(issue.blockerKeys, key)
			}
		}
		for _, idx := range headerInfo.blockedIdx {
			key := columns[idx]
			if len(key) > 0 {
				issue.blockedKeys = append(issue.blockedKeys, key)
			}
		}
		issues = append(issues, issue)
	}
	return issues
}

func normalizeKey(key string) string {
	return strings.ReplaceAll(key, "-", "")
}
