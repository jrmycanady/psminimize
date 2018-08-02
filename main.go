package main

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/google/uuid"
)

const (
	CHARComment byte = 35
	ESCChar1    byte = 92
	ESCChar2    byte = 96
	LT          byte = 60
	GT          byte = 62
)

var (
	varShortNames = []byte{65, 66, 67, 68, 69, 70, 71, 72, 73, 74, 75, 76, 77, 78, 79, 80, 81, 82, 83, 84, 85, 86, 87, 88, 89, 90, 97, 98, 99, 100, 101, 102, 103, 104, 105, 106, 107, 108, 109, 110, 111, 112, 113, 114, 115, 116, 117, 118, 119, 120, 121}
)

// PSVariable represents a variable found in the PowerShell file.
type PSVariable struct {
	OriginalName string
	UniqueName   string
	ShortName    string
	Count        int
}

// PSVariables represents a slice of PSVariable structs that can be
// sorted.
type PSVariables []PSVariable

func (p PSVariables) Len() int           { return len(p) }
func (p PSVariables) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }
func (p PSVariables) Less(i, j int) bool { return p[i].Count > p[j].Count }

// assignUniqueRandomNames assigns a unique random name to every variable.
func (p PSVariables) assignUniqueRandomNames() {
	for i := range p {
		id, err := uuid.NewRandom()
		panicOnErr(err)
		p[i].UniqueName = fmt.Sprintf("$~~%s", strings.ToUpper(id.String()))
	}

	sort.Sort(PSVariablesNameMod(p))
}

// replaceVariablesWithUnique replaces all the variables with their unique
// name.
func (p PSVariables) replaceVariablesWithUnique(lines []string) {
	sort.Sort(PSVariablesNameMod(p))
	for i := range lines {
		lines[i] = strings.ToUpper(lines[i])
		for j := 0; j < len(p); j++ {
			// fmt.Println(lines[i])
			lines[i] = strings.Replace(lines[i], p[j].OriginalName, p[j].UniqueName, -1)
		}
	}
}

// replaceUniqueWithShort replaces all unique variables with the short version.
func (p PSVariables) replaceUniqueWithShort(lines []string) {
	for i := range lines {
		for j := range p {
			lines[i] = strings.Replace(lines[i], p[j].UniqueName, p[j].ShortName, -1)
		}
	}
}

type PSVariablesNameMod PSVariables

func (p PSVariablesNameMod) Len() int      { return len(p) }
func (p PSVariablesNameMod) Swap(i, j int) { p[i], p[j] = p[j], p[i] }
func (p PSVariablesNameMod) Less(i, j int) bool {
	return len(p[i].OriginalName) > len(p[j].OriginalName)
}

// panicOnErr checks if e is nil and if not panics.
func panicOnErr(e error) {
	if e != nil {
		panic(e)
	}
}

func main() {
	var minimizedLines = make([]string, 0, 20)
	var originalLines = make([]string, 0, 0)

	// Reading the file into the original array and duplicate for minimized.
	f, err := os.Open("sample.ps1")
	panicOnErr(err)
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		originalLines = append(originalLines, scanner.Text())
	}
	minimizedLines = make([]string, len(originalLines), len(originalLines))
	copy(minimizedLines, originalLines)

	stripAllComments(minimizedLines)

	shortenAllVariableNames(minimizedLines)

	printComparison(originalLines, minimizedLines)

	saveToFile(minimizedLines)
}

func saveToFile(lines []string) {
	f, err := os.Create("./test.ps1")
	panicOnErr(err)

	for i := range lines {
		_, err := f.WriteString(lines[i] + "\n")
		panicOnErr(err)
	}
}

func printComparison(original []string, minimized []string) {
	for i := 0; i < len(original); i++ {
		if len(minimized) > i {
			fmt.Printf("%-120s | %s\n", original[i], minimized[i])
		} else {
			fmt.Printf("%-120s |\n", original[i])
		}

	}
}

// stripAllComments strips any comments form all lines in the slice and
// stores the result back into place.
func stripAllComments(lines []string) {
	var multi bool
	for i := range lines {
		lines[i], multi = stripComments(lines[i], multi)
	}
}

// stripComments removes any comments from the line and returns the line
// with the comments stripped. If a multi line comment was started but not
// finished the bool return value will be true.
func stripComments(line string, multi bool) (string, bool) {
	minLine := make([]byte, 0, len(line))

	// Processing obvious ignorable lines returning an empty line in
	// its place.
	switch {
	case len(line) <= 0:
		return "", multi
	case line[0] == CHARComment:
		return "", multi
	}

	// Processing each character looking for a start or stop of a comment
	// depending on the value of mutli.
	for i := 0; i < len(line); i++ {
		if multi {
			// A multi line comment has continued so looking for the end
			// comment character. If found looking forward one (if possible)
			// and if it's > closing the comment. In all cases no characters
			// are saved as we are in a multi line comment.
			if line[i] == CHARComment {
				if len(line) >= i && line[i+1] == GT {
					i++
					multi = false
				}
			}
		} else {
			if line[i] == CHARComment {
				// Ruling out escaped comment character.
				if len(line) >= i && (line[i-1] == ESCChar1 || line[i-1] == ESCChar2) {
					minLine = append(minLine, line[i])
					continue
				}

				// Ruling out that it's not a start of a multiline.
				if len(line) >= i && (line[i-1] == LT) {
					multi = true
					// remove the start character found
					minLine = minLine[:len(minLine)-1]
					continue
				}

				// Appears to be a comment so ignore from here on.
				break
			} else {
				// Appears to be a valid character so add.
				minLine = append(minLine, line[i])
			}
		}
	}

	return string(minLine), multi
}

// shortenAllVariableNames shortens all the variable names to the minimum
// characters possible.
func shortenAllVariableNames(lines []string) {
	// Retrieving all variables and their counts.
	psVars := getVariables(lines)
	psVars.shortenVariables(lines)
}

// getVariables retrieves all the variables found in lines along with the
// count.
func getVariables(lines []string) PSVariables {
	var psVars PSVariables
	var psVarMap = make(map[string]int)
	var psVarReg = regexp.MustCompile("[`]?\\$[A-Z0-9a-z_]*")

	for i := range lines {
		r := psVarReg.FindAllStringSubmatch(lines[i], -1)
		if r == nil {
			continue
		}

		for j := range r {
			for m := range r[j] {
				varName := strings.ToUpper(r[j][m])
				psVarMap[varName]++
			}
		}
	}
	for k, v := range psVarMap {
		// Skipping any reserved.
		_, ok := reservedPSVariables[k]
		if ok {
			continue
		}

		// Skipping any starting with an escape character.
		if string(k[0]) == "`" {
			continue
		}

		psVars = append(psVars, PSVariable{OriginalName: k, Count: v})
	}

	return psVars
}

// Sort sorts the PSVariable by count.
func (p PSVariables) Sort() {
	sort.Sort(p)
}

// generateShortNames generates short names for all variables making sure
// the more used variables have the shortest name.
func (p PSVariables) generateShortNames() {

	var count int
	for i := 0; i < len(p); i++ {
		s := "$" + string(varShortNames[i-(51*count)])
		if count > 0 {
			s = s + strconv.Itoa(count-1)
		}

		p[i].ShortName = s

		if ((i + 1) % 51) == 0 {
			count++
		}
	}
}

// shortenVariables shorts all variables found in lines.
func (p PSVariables) shortenVariables(lines []string) {
	p.Sort()
	p.assignUniqueRandomNames()
	p.replaceVariablesWithUnique(lines)
	p.generateShortNames()
	p.replaceUniqueWithShort(lines)
}

// getNextShortname returns the next shortname to use. Use 0 for the first call.
func getNextShortName(lastName byte) byte {
	if lastName == 0 {
		return 65
	}

	// Get the next character.
	lastName++

	// Skip special characters
	if lastName == 91 {
		return 97
	}

	if lastName > 172 {

	}

	return lastName

}
