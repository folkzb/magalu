package core

import (
	"slices"
	"strings"
)

type Scope string
type Scopes []Scope
type ScopesString string

func (s *Scopes) Add(scopes ...Scope) {
	for _, scope := range scopes {
		if slices.Contains(*s, scope) {
			continue
		}
		*s = append(*s, scope)
	}
}

func (s *Scopes) Remove(toBeRemoved ...Scope) {
	if s == nil {
		return
	}
	result := make(Scopes, 0, len(*s)) // Capacity can't be lower because we can't know if there are repeated scopes to be removed...
	for _, existingScope := range *s {
		if slices.Contains(toBeRemoved, existingScope) {
			continue
		}
		result = append(result, existingScope)
	}
	*s = result
}

func (s Scopes) AsScopesString() ScopesString {
	// Can't use 'strings.Join' because of 'Scopes' type
	var result ScopesString
	for i, scope := range s {
		if i > 0 {
			result += " "
		}
		result += ScopesString(scope)
	}
	return result
}

func (s ScopesString) AsScopes() Scopes {
	strSlice := strings.Split(string(s), " ")
	result := make(Scopes, len(strSlice))
	for i, scopeStr := range strSlice {
		result[i] = Scope(scopeStr)
	}
	return result
}
