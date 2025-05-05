package cliparser

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/DBCDK/morph/internal/constraints"
)

func ParseConstraints(constraintFlags []string) (result []constraints.Constraint, err error) {
	for _, c := range constraintFlags {
		if len(c) == 0 {
			continue
		}

		// arguments look like this: labelKey=labelValue:constraintType:constraintValue, e.g. location=dc1:maxUnavailable=2
		parts := strings.SplitN(c, ":", 2)
		labelHalf := parts[0]
		constraintHalf := parts[1]
		labelParts := strings.SplitN(labelHalf, "=", 2)

		labelKey := labelParts[0]
		labelValue := labelParts[1]
		labelSelector := constraints.LabelSelector{Label: labelKey, Value: labelValue}

		constraintParts := strings.SplitN(constraintHalf, "=", 2)

		constraintType := constraintParts[0]
		constraintValue := constraintParts[1]

		switch strings.ToLower(constraintType) {
		case "maxunavailable":
			maxUnavailable, err := strconv.Atoi(constraintValue)
			if err != nil {
				return result, fmt.Errorf("Invalid value in constraint - not an integer: " + constraintValue)
			}
			result = append(result, constraints.NewConstraint(labelSelector, maxUnavailable))

		default:
			return result, fmt.Errorf("Unknown constraint type: " + constraintType)
		}
	}

	return result, nil
}
