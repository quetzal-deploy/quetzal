package constraints

import (
	"testing"

	"github.com/kr/pretty"
)

func TestSelect(t *testing.T) {
	locationFuzzy := NewConstraint(LabelSelector{
		Label: "location",
		Value: "*",
	}, 1)
	locationSpecific := NewConstraint(LabelSelector{
		Label: "location",
		Value: "dc1",
	}, 2)

	typeHostFirst := NewConstraint(LabelSelector{
		Label: "type",
		Value: "host",
	}, 1)
	typeHostSecond := NewConstraint(LabelSelector{
		Label: "type",
		Value: "host",
	}, 2)

	archSpecific := NewConstraint(LabelSelector{
		Label: "arch",
		Value: "amd64",
	}, 2)
	archFuzzy := NewConstraint(LabelSelector{
		Label: "arch",
		Value: "*",
	}, 1)

	constraints := []Constraint{
		locationFuzzy,
		locationSpecific,
		typeHostFirst,
		typeHostSecond,
		archSpecific,
		archFuzzy,
	}

	result, err := Select(constraints, "location", "dc1")
	if err != nil {
		t.Fatal(err.Error())
	}

	if result != locationSpecific {
		t.Fatalf("Wrong constraint returned: Expected %v, got %v", locationSpecific, result)
	}

	result, err = Select(constraints, "location", "dc2")
	if err != nil {
		t.Fatal(err.Error())
	}

	if result != locationFuzzy {
		t.Fatalf("Wrong constraint returned: Expected %v, got %v", locationFuzzy, result)
	}

	result, err = Select(constraints, "type", "host")
	if err != nil {
		t.Fatal(err.Error())
	}

	if result != typeHostFirst {
		t.Fatalf("Wrong constraint returned: Expected %v, got %v", typeHostFirst, result)
	}

	result, err = Select(constraints, "arch", "amd64")
	if err != nil {
		t.Fatal(err.Error())
	}

	if result != archSpecific {
		t.Fatalf("Wrong constraint returned: Expected %v, got %v", archSpecific, result)
	}

	result, err = Select(constraints, "arch", "x86")
	if err != nil {
		t.Fatal(err.Error())
	}

	if result != archFuzzy {
		t.Fatalf("Wrong constraint returned: Expected %v, got %v", archFuzzy, result)
	}
}

func TestMatch(t *testing.T) {

	lsA := LabelSelector{Label: "type", Value: "a"}
	lsB := LabelSelector{Label: "type", Value: "b"}

	set1 := []Constraint{
		NewConstraint(lsA, 1),
	}

	set2 := []Constraint{
		NewConstraint(lsA, 2),
	}

	result := Merge(set1, set2)

	if len(result) != 1 {
		t.Fatalf("Expected %d constraints, got %d", 1, len(result))
	}

	if result[0].MaxUnavailable != set2[0].MaxUnavailable {
		t.Fatalf("Expected MaxUnavailable = %d, got %d", set2[0].MaxUnavailable, result[0].MaxUnavailable)
	}

	set3 := []Constraint{
		NewConstraint(lsB, 3),
	}

	result = Merge(set3, result)

	pretty.Println(result)

	if len(result) != 2 {
		t.Fatalf("Expected %d constraints, got %d", 2, len(result))
	}

	if result[1].MaxUnavailable != set3[0].MaxUnavailable {
		t.Fatalf("Expected MaxUnavailable = %d, got %d", set3[0].MaxUnavailable, result[1].MaxUnavailable)
	}

	set4 := []Constraint{
		NewConstraint(lsB, 10),
	}

	result = Merge(result, set4)

	pretty.Println(result)

	if len(result) != 2 {
		t.Fatalf("Expected %d constraints, got %d", 2, len(result))
	}

	if result[0].MaxUnavailable != set4[0].MaxUnavailable {
		t.Fatalf("Expected MaxUnavailable = %d, got %d", set4[0].MaxUnavailable, result[0].MaxUnavailable)
	}

	//FIXME: Not a feasible way of testing this. Make a constraints.Select function to use instead.

	//t.Fatalf(pretty.Sprint(result))
}
