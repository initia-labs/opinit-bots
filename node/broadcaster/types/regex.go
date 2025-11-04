package types

import (
	"regexp"
	"strconv"

	"github.com/pkg/errors"
)

var AccountSeqRegex = regexp.MustCompile("account sequence mismatch, expected ([0-9]+), got ([0-9]+)")

func ParseAccountSequenceMismatch(str string) (uint64, uint64, error) {
	if strs := AccountSeqRegex.FindStringSubmatch(str); strs != nil {
		expected, parseErr := strconv.ParseUint(strs[1], 10, 64)
		if parseErr != nil {
			return 0, 0, parseErr
		}
		got, parseErr := strconv.ParseUint(strs[2], 10, 64)
		if parseErr != nil {
			return 0, 0, parseErr
		}
		return expected, got, nil
	}
	return 0, 0, errors.New("not matched")
}
