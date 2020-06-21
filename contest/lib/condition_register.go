package contestlib

import (
	"fmt"
	"regexp"
	"strings"

	"golang.org/x/xerrors"
)

var (
	reAssignStringFormat = `^[a-zA-Z0-9_][a-zA-Z0-9_]*$`
	reAssignString       = regexp.MustCompile(reAssignStringFormat)
)

type ConditionRegister struct {
	Key    string
	Assign string
}

func (cr *ConditionRegister) String() string {
	return fmt.Sprintf("key=%q assign=%q", cr.Key, cr.Assign)
}

func (cr *ConditionRegister) IsValid([]byte) error {
	if s := strings.TrimSpace(cr.Key); len(s) < 1 {
		return xerrors.Errorf("empty key")
	} else {
		cr.Key = s
	}

	if s := strings.TrimSpace(cr.Assign); len(s) < 1 {
		return xerrors.Errorf("empty assign")
	} else {
		cr.Assign = s
	}

	if !IsValidLookupKey(cr.Key) {
		return xerrors.Errorf("invalid key format, should be `%s`", reLookupKeyFormat)
	}

	if !reAssignString.Match([]byte(cr.Assign)) {
		return xerrors.Errorf("invalid assign format, should be `%s`", reAssignStringFormat)
	}

	return nil
}
