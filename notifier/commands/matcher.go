package commands

import (
	"errors"
	"regexp"
)

var ErrNoMatch = errors.New("text doesn't getHandler")

// This matches a user message returning the argument list
// Provided is a regexp matcher, but this could also be a NLP one.
type Matcher interface {
	Match(text string) (map[string]string, error)
}

type regexpMatcher struct {
	regexp *regexp.Regexp
}

func RegexpMatcher(regexp *regexp.Regexp) Matcher {
	return &regexpMatcher{
		regexp: regexp,
	}
}

func (matcher *regexpMatcher) Match(text string) (map[string]string, error) {
	matched := matcher.regexp.FindStringSubmatch(text)

	if matched == nil {
		return nil, ErrNoMatch
	}

	if len(matched) != len(matcher.regexp.SubexpNames()) {
		return nil, errors.New("invalid subexpression count")
	}

	params := make(map[string]string)
	// We skip the first one as that's the whole message text
	subexpNames := matcher.regexp.SubexpNames()[1:]
	subexpMatches := matched[1:]
	for i, key := range subexpNames {
		params[key] = subexpMatches[i]
	}

	return params, nil
}
