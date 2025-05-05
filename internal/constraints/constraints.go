package constraints

import (
	"encoding/json"
	"fmt"
)

type Constraint struct {
	Selector       LabelSelector `json:"selector"`
	MaxUnavailable int           `json:"maxUnavailable"`
}

func NewConstraint(selector LabelSelector, maxUnavailable int) Constraint {
	return Constraint{
		Selector:       selector,
		MaxUnavailable: maxUnavailable,
	}
}

// Set default values for a Constraint when being unmarshalled
func (c *Constraint) UnmarshalJSON(data []byte) error {
	type ConstraintAlias Constraint
	return json.Unmarshal(data, (*ConstraintAlias)(c))
}

// FIXME: Make LabelSelector a map[string]string where all labels have to match
type LabelSelector struct {
	Label string
	Value string
}

func (ls LabelSelector) Match(label string, value string) bool {
	return ls.Label == label && (ls.Value == "*" || value == "*" || ls.Value == value)
}

func Select(constraints []Constraint, label string, value string) (result Constraint, err error) {
	fuzzyHit := false

	for _, constraint := range constraints {
		// ignore constraint if the label doesn't match
		if constraint.Selector.Label != label {
			continue
		}

		if constraint.Selector.Value == value {
			// this is an exact match -> return immediately
			result = constraint
			return result, nil

		} else if constraint.Selector.Value == "*" {
			// already had a fuzzy hit -> prioritise the first one found
			if fuzzyHit {
				continue
			}

			// register this as the first fuzzy hit
			fuzzyHit = true
			result = constraint
		} else {
			// no match -> do nothing
		}
	}

	if fuzzyHit {
		return result, err
	} else {
		return result, fmt.Errorf("no constraints with label '%s' found", label)
	}
}

// later sets take priority
// a constraint matching '*' will override any prior '*' or specific value
// a constraint matching 'value' will _not_ override a prior '*', only prior 'value'
func Merge(lowPriConstraints []Constraint, highPriConstraints []Constraint) []Constraint {
	//panic("FIXME: implement me")

	if len(lowPriConstraints) == 0 {
		return highPriConstraints
	} else if len(highPriConstraints) == 0 {
		return lowPriConstraints
	}

	result := highPriConstraints

	for _, lowPriConstraint := range lowPriConstraints {
		add := true

		for _, highPriConstraint := range highPriConstraints {
			if lowPriConstraint.Selector.Label != highPriConstraint.Selector.Label {
				continue
			}

			if lowPriConstraint.Selector.Value == highPriConstraint.Selector.Value {
				if highPriConstraint.Selector.Value == "*" && highPriConstraint.Selector.Value != "*" {
					add = true
				} else {
					add = false
				}
			}
		}

		if add {
			result = append(result, lowPriConstraint)
		}
	}

	return result
}
