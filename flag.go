package astikit

import (
	"os"
	"strings"
)

// FlagCmd retrieves the command from the input Args
func FlagCmd() (o string) {
	if len(os.Args) >= 2 && os.Args[1][0] != '-' {
		o = os.Args[1]
		os.Args = append([]string{os.Args[0]}, os.Args[2:]...)
	}
	return
}

// FlagStrings represents a flag that can be set several times and
// stores unique string values
type FlagStrings map[string]bool

// NewFlagStrings creates a new FlagStrings
func NewFlagStrings() FlagStrings {
	return FlagStrings(make(map[string]bool))
}

// String implements the flag.Value interface
func (f FlagStrings) String() string {
	var s []string
	for k := range f {
		s = append(s, k)
	}
	return strings.Join(s, ",")
}

// Set implements the flag.Value interface
func (f FlagStrings) Set(i string) error {
	f[i] = true
	return nil
}
