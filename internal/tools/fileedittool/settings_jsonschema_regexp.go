package fileedittool

import (
	"github.com/dlclark/regexp2"
	"github.com/santhosh-tekuri/jsonschema/v6"
)

// SchemaStore permissionRule.pattern uses lookahead; Go regexp (RE2) cannot compile it.
// ECMAScript mode matches JSON Schema / AJV behavior closely enough for Edit-time checks.

type dlclarkRegexp regexp2.Regexp

func (re *dlclarkRegexp) MatchString(s string) bool {
	matched, err := (*regexp2.Regexp)(re).MatchString(s)
	return err == nil && matched
}

func (re *dlclarkRegexp) String() string {
	return (*regexp2.Regexp)(re).String()
}

func settingsSchemaRegexpCompile(s string) (jsonschema.Regexp, error) {
	re, err := regexp2.Compile(s, regexp2.ECMAScript)
	if err != nil {
		return nil, err
	}
	return (*dlclarkRegexp)(re), nil
}
