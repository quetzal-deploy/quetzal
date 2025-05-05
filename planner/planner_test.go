package planner

import (
	"context"
	"fmt"
	"reflect"
	"slices"
	"testing"

	"github.com/kr/pretty"

	"github.com/DBCDK/morph/common"
	"github.com/DBCDK/morph/events"
	"github.com/DBCDK/morph/internal/constraints"
	"github.com/DBCDK/morph/nix"
	"github.com/DBCDK/morph/steps"
)

func TestSolverGetNumeralId(t *testing.T) {
	ids0 := make([]string, 0)

	stringId1 := "donkey"
	expectedIds1 := append(make([]string, 0), stringId1)
	ids1, nextId1 := solverGetNumericalId(ids0, stringId1)

	stringId2 := "zebra"
	expectedIds2 := append(expectedIds1, stringId2)
	ids2, nextId2 := solverGetNumericalId(ids1, stringId2)

	if nextId1 != 1 {
		t.Fatalf("Expected next numerical id = 1, got id = %d", nextId1)
	}

	if len(ids1) != 1 {
		t.Fatalf("Expected returned list of ids to have length = 1, got length = %d", len(ids1))
	}

	if !slices.Equal(expectedIds1, ids1) {
		t.Fatalf("Expected returned list of ids to be %v, got %v", expectedIds1, ids1)
	}

	if nextId2 != 2 {
		t.Fatalf("Expected next numerical id = 2, got id = %d", nextId2)
	}

	if len(ids2) != 2 {
		t.Fatalf("Expected returned list of ids to have length = 2, got length = %d", len(ids2))
	}

	if !slices.Equal(expectedIds2, ids2) {
		t.Fatalf("Expected returned list of ids to be %v, got %v", expectedIds2, ids2)
	}

	ids2_, nextId1_ := solverGetNumericalId(ids2, stringId1)

	if nextId1 != nextId1_ {
		t.Fatalf("Expected to get previous numerical id = %d when querying for %s again, but got id = %d", nextId1, stringId1, nextId1_)
	}

	if !slices.Equal(ids2, ids2_) {
		t.Fatalf("Expected to get previous list of ids = %v when querying for %s again, but got ids = %v", ids2, stringId1, ids2_)
	}
}

func TestWeightsOfOnes(t *testing.T) {

	count0 := 0
	result0 := weightsOfOnes(count0)
	expected0 := make([]int, 0)

	count5 := 5
	result5 := weightsOfOnes(count5)
	expected5 := []int{1, 1, 1, 1, 1}

	if !slices.Equal(result0, expected0) {
		t.Fatalf("Expected array to have %d ones, got %v", count0, result0)
	}

	if !slices.Equal(result5, expected5) {
		t.Fatalf("Expected array to have %d ones, got %v", count5, result5)
	}
}

func TestCanStartStepCardinality1(t *testing.T) {
	hosts := make(map[string]nix.Host)
	opts := common.MorphOptions{}

	label1Key := "type"
	label1Value := "web"

	constraints_ := make([]constraints.Constraint, 0)
	constraints_ = append(constraints_, constraints.NewConstraint(constraints.LabelSelector{Label: label1Key, Value: label1Value}, 1))

	planner := NewPlanner(events.NewManager(), hosts, &opts, constraints_)

	step1 := steps.New().Id("host:1").Label(label1Key, label1Value).Build()
	step2 := steps.New().Id("host:2").Label(label1Key, label1Value).Build()
	step3 := steps.New().Id("host:3").Label(label1Key, label1Value).Build()

	planner.QueueSteps(step1, step2, step3)

	if ok, _ := planner.CanStartStep(step1); !ok {
		t.Fatalf("Expected to be able to start step = %s", step1.Id)
	}

	planner.StepStatus.Update(step1.Id, Running)

	if ok, _ := planner.CanStartStep(step2); ok {
		t.Fatalf("Should not be able to start step = %s with constraints = %v", step2.Id, planner.Constraints)
	}
}

func TestCanStartStepCardinality2(t *testing.T) {
	hosts := make(map[string]nix.Host)
	opts := common.MorphOptions{}

	label1Key := "type"
	label1Value := "web"

	constraints_ := make([]constraints.Constraint, 0)
	constraints_ = append(constraints_, constraints.NewConstraint(constraints.LabelSelector{Label: label1Key, Value: label1Value}, 2))
	constraints_ = append(constraints_, constraints.NewConstraint(constraints.LabelSelector{Label: "location", Value: "*"}, 1))
	//constraints_ = append(constraints_, constraints.NewConstraint(constraints.LabelSelector{Label: "location", Value: "dc1"}, 1))
	//constraints_ = append(constraints_, constraints.NewConstraint(constraints.LabelSelector{Label: "location", Value: "dc2"}, 1))

	planner := NewPlanner(events.NewManager(), hosts, &opts, constraints_)

	step1 := steps.New().Id("host:1").Label(label1Key, label1Value).Label("location", "dc1").Build()
	step2 := steps.New().Id("host:2").Label(label1Key, label1Value).Label("location", "dc1").Build()
	step3 := steps.New().Id("host:3").Label(label1Key, label1Value).Label("location", "dc2").Build()
	step4 := steps.New().Id("host:4").Label(label1Key, label1Value).Label("location", "dc2").Build()

	planner.QueueSteps(step1, step2, step3, step4)

	//FIXME: gør det en forskel at denne test ikke tester en plan med substeps, men multiple steps loaded one by one som hver sin plan?

	// Emulate starting step1. Should succeed.
	if ok, _ := planner.CanStartStep(step1); !ok {
		t.Fatalf("Expected to be able to start step = %s", step1.Id)
	}

	// step1 is now started/running
	planner.StepStatus.Update(step1.Id, Running)

	// Emulate starting step2. Should fail since it has same location as step1.
	if ok, _ := planner.CanStartStep(step2); ok {
		t.Fatalf("Should not be able to start step = %s with constraints_ = %v", step2.Id, planner.Constraints)
	}

	// Emulate starting step3. Should succeed.
	if ok, _ := planner.CanStartStep(step3); !ok {
		t.Fatalf("Expected to be able to start step = %s with constraints_ = %v", step3.Id, planner.Constraints)
	}

	// reset step1
	planner.StepStatus.Update(step1.Id, Queued)

	// Emulate starting step3. Should succeed.
	if ok, _ := planner.CanStartStep(step3); !ok {
		t.Fatalf("Expected to be able to start step = %s with constraints_ = %v", step3.Id, planner.Constraints)
	}

	// step2 is now started/running
	planner.StepStatus.Update(step3.Id, Running)

	// Emulate starting step4. Should fail since it has same location as step1.
	if ok, _ := planner.CanStartStep(step4); ok {
		t.Fatalf("Should not be able to start step = %s with constraints_ = %v", step4.Id, planner.Constraints)
	}

	// Emulate starting step2. Should succeed.
	if ok, _ := planner.CanStartStep(step2); !ok {
		t.Fatalf("Expected to be able to start step = %s with constraints_ = %v", step3.Id, planner.Constraints)
	}

	// step2 is now started/running
	planner.StepStatus.Update(step3.Id, Done)

	// Emulate starting step4. Should succeed.
	if ok, _ := planner.CanStartStep(step4); !ok {
		t.Fatalf("Expected to be able to start step = %s with constraints_ = %v", step4.Id, planner.Constraints)
	}

	planner.StepStatus.Update(step4.Id, Done)

	// Emulate starting step1. Should succeed.
	if ok, _ := planner.CanStartStep(step1); !ok {
		t.Fatalf("Expected to be able to start step = %s with constraints_ = %v", step1.Id, planner.Constraints)
	}

	planner.StepStatus.Update(step1.Id, Running)

	// Emulate starting step2. Should fail since it has same location as step1.
	if ok, _ := planner.CanStartStep(step2); ok {
		t.Fatalf("Should not be able to start step = %s with constraints_ = %v", step2.Id, planner.Constraints)
	}

	planner.StepStatus.Update(step1.Id, Done)

	// Emulate starting step2. Should succeed.
	if ok, _ := planner.CanStartStep(step2); !ok {
		t.Fatalf("Expected to be able to start step = %s with constraints_ = %v", step2.Id, planner.Constraints)
	}

	planner.StepStatus.Update(step2.Id, Done)
}

func TestCanStartStepCardinality2copy(t *testing.T) {
	fmt.Println()
	fmt.Println("TestCanStartStepCardinality2copy")

	hosts := make(map[string]nix.Host)
	opts := common.MorphOptions{}

	label1Key := "type"
	label1Value := "web"

	constraints_ := make([]constraints.Constraint, 0)
	constraints_ = append(constraints_, constraints.NewConstraint(constraints.LabelSelector{Label: label1Key, Value: label1Value}, 2))
	//constraints_ = append(constraints_, constraints.NewConstraint(nix.LabelSelector{Label: "location", Value: "*"}, 1))
	constraints_ = append(constraints_, constraints.NewConstraint(constraints.LabelSelector{Label: "location", Value: "dc1"}, 1))
	constraints_ = append(constraints_, constraints.NewConstraint(constraints.LabelSelector{Label: "location", Value: "dc2"}, 1))

	planner := NewPlanner(events.NewManager(), hosts, &opts, constraints_)

	step1 := steps.New().Id("host:1").Label(label1Key, label1Value).Label("location", "dc1").Build()
	step2 := steps.New().Id("host:2").Label(label1Key, label1Value).Label("location", "dc1").Build()
	step3 := steps.New().Id("host:3").Label(label1Key, label1Value).Label("location", "dc2").Build()
	step4 := steps.New().Id("host:4").Label(label1Key, label1Value).Label("location", "dc2").Build()

	planner.QueueSteps(step1, step2, step3, step4)

	//FIXME: gør det en forskel at denne test ikke tester en plan med substeps, men multiple steps loaded one by one som hver sin plan?

	// Status:
	// - step1: Queued
	// - step2: Queued
	// - step3: Queued
	// - step4: Queued

	// Emulate starting step1. Should succeed.
	if ok, _ := planner.CanStartStep(step1); !ok {
		t.Fatalf("Expected to be able to start step = %s", step1.Id)
	}

	// mark step1 as started/running
	//planner.StepStatus.Update(step1.Id, Running)
	planner.UpdateStepStatus(step1.Id, Running)

	// Status:
	// - step1: Running
	// - step2: Queued
	// - step3: Queued
	// - step4: Queued

	// Emulate starting step2. Should fail:
	// - same location as step1
	if ok, _ := planner.CanStartStep(step2); ok {
		t.Fatalf("Should not be able to start step = %s with constraints_ = %v", step2.Id, planner.Constraints)
	}

	// Emulate starting step3. Should succeed.
	if ok, _ := planner.CanStartStep(step3); !ok {
		t.Fatalf("Expected to be able to start step = %s with constraints_ = %v", step3.Id, planner.Constraints)
	}

	// mark step3 as started/running
	//planner.StepStatus.Update(step3.Id, Running)
	planner.UpdateStepStatus(step3.Id, Running)

	// Status:
	// - step1: Running
	// - step2: Queued
	// - step3: Running
	// - step4: Queued

	// Emulate starting step4. Should fail, since
	// - type constraint: requires only 2 down
	// - location constraint: already 1 down in each location
	if ok, _ := planner.CanStartStep(step4); ok {
		t.Fatalf("Should not be able to start step = %s with constraints_ = %v", step3.Id, planner.Constraints)
	}

	// mark step1 as done
	//planner.StepStatus.Update(step1.Id, Done)
	planner.UpdateStepStatus(step1.Id, Done)

	// Status:
	// - step1: Done
	// - step2: Queued
	// - step3: Running
	// - step4: Queued

	// Emulate starting step4. Should still fail, since
	// - same location as step 3
	if ok, _ := planner.CanStartStep(step4); ok {
		t.Fatalf("Should not be able to start step = %s with constraints_ = %v", step4.Id, planner.Constraints)
	}

	// Emulate starting step2. Should succeed.
	if ok, _ := planner.CanStartStep(step2); !ok {
		t.Fatalf("Expected to be able to start step = %s with constraints_ = %v", step2.Id, planner.Constraints)
	}

	// step2 is now started/running
	//planner.StepStatus.Update(step2.Id, Running)
	planner.UpdateStepStatus(step2.Id, Running)

	// Status:
	// - step1: Done
	// - step2: Running
	// - step3: Running
	// - step4: Queued

	// Emulate starting step4. Should fail since it has same location as step3.
	if ok, _ := planner.CanStartStep(step4); ok {
		t.Fatalf("Should not be able to start step = %s with constraints_ = %v", step4.Id, planner.Constraints)
	}

	// Emulate starting step2. Should succeed.
	if ok, _ := planner.CanStartStep(step2); !ok {
		t.Fatalf("Expected to be able to start step = %s with constraints_ = %v", step2.Id, planner.Constraints)
	}

	// mark step3 as done
	//planner.StepStatus.Update(step3.Id, Done)
	planner.UpdateStepStatus(step3.Id, Done)

	// Status:
	// - step1: Done
	// - step2: Running
	// - step3: Done
	// - step4: Queued

	// Emulate starting step4. Should succeed.
	if ok, _ := planner.CanStartStep(step4); !ok {
		t.Fatalf("Expected to be able to start step = %s with constraints_ = %v", step4.Id, planner.Constraints)
	}

	//planner.StepStatus.Update(step4.Id, Running)
	planner.UpdateStepStatus(step4.Id, Running)

	// Status:
	// - step1: Done
	// - step2: Running
	// - step3: Done
	// - step4: Running

}

func TestCanStartStepCardinality2copyNested(t *testing.T) {
	fmt.Println()
	fmt.Println("TestCanStartStepCardinality2copyNested")

	hosts := make(map[string]nix.Host)
	opts := common.MorphOptions{}

	label1Key := "type"
	label1Value := "web"

	constraints_ := make([]constraints.Constraint, 0)
	constraints_ = append(constraints_, constraints.NewConstraint(constraints.LabelSelector{Label: label1Key, Value: label1Value}, 2))
	//constraints_ = append(constraints_, nix.NewConstraint(nix.LabelSelector{Label: "location", Value: "*"}, 1))
	constraints_ = append(constraints_, constraints.NewConstraint(constraints.LabelSelector{Label: "location", Value: "dc1"}, 1))
	constraints_ = append(constraints_, constraints.NewConstraint(constraints.LabelSelector{Label: "location", Value: "dc2"}, 1))

	planner := NewPlanner(events.NewManager(), hosts, &opts, constraints_)

	step1 := steps.New().Id("host:1").Label(label1Key, label1Value).Label("location", "dc1").Build()
	step2 := steps.New().Id("host:2").Label(label1Key, label1Value).Label("location", "dc1").Build()
	step3 := steps.New().Id("host:3").Label(label1Key, label1Value).Label("location", "dc2").Build()
	step4 := steps.New().Id("host:4").Label(label1Key, label1Value).Label("location", "dc2").Build()

	plan := steps.New().Id("root").AddSteps(step1, step2, step3, step4).Build()

	planner.QueueSteps(plan)

	planner.UpdateStepStatus(plan.Id, Running)
	planner.QueueSteps(step1, step2, step3, step4)

	// Status:
	// - step1: Queued
	// - step2: Queued
	// - step3: Queued
	// - step4: Queued

	// Emulate starting step1. Should succeed.
	if ok, _ := planner.CanStartStep(step1); !ok {
		t.Fatalf("Expected to be able to start step = %s", step1.Id)
	}

	// mark step1 as started/running
	//planner.StepStatus.Update(step1.Id, Running)
	planner.UpdateStepStatus(step1.Id, Running)

	// Status:
	// - step1: Running
	// - step2: Queued
	// - step3: Queued
	// - step4: Queued

	// Emulate starting step2. Should fail:
	// - same location as step1
	if ok, _ := planner.CanStartStep(step2); ok {
		t.Fatalf("Should not be able to start step = %s with constraints_ = %v", step2.Id, planner.Constraints)
	}

	// Emulate starting step3. Should succeed.
	if ok, _ := planner.CanStartStep(step3); !ok {
		t.Fatalf("Expected to be able to start step = %s with constraints_ = %v", step3.Id, planner.Constraints)
	}

	// mark step3 as started/running
	//planner.StepStatus.Update(step3.Id, Running)
	planner.UpdateStepStatus(step3.Id, Running)

	// Status:
	// - step1: Running
	// - step2: Queued
	// - step3: Running
	// - step4: Queued

	// Emulate starting step4. Should fail, since
	// - type constraint: requires only 2 down
	// - location constraint: already 1 down in each location
	if ok, _ := planner.CanStartStep(step4); ok {
		t.Fatalf("Should not be able to start step = %s with constraints_ = %v", step3.Id, planner.Constraints)
	}

	// mark step1 as done
	//planner.StepStatus.Update(step1.Id, Done)
	planner.UpdateStepStatus(step1.Id, Done)

	// Status:
	// - step1: Done
	// - step2: Queued
	// - step3: Running
	// - step4: Queued

	// Emulate starting step4. Should still fail, since
	// - same location as step 3
	if ok, _ := planner.CanStartStep(step4); ok {
		t.Fatalf("Should not be able to start step = %s with constraints_ = %v", step4.Id, planner.Constraints)
	}

	// Emulate starting step2. Should succeed.
	if ok, _ := planner.CanStartStep(step2); !ok {
		t.Fatalf("Expected to be able to start step = %s with constraints_ = %v", step2.Id, planner.Constraints)
	}

	// step2 is now started/running
	//planner.StepStatus.Update(step2.Id, Running)
	planner.UpdateStepStatus(step2.Id, Running)

	// Status:
	// - step1: Done
	// - step2: Running
	// - step3: Running
	// - step4: Queued

	// Emulate starting step4. Should fail since it has same location as step3.
	if ok, _ := planner.CanStartStep(step4); ok {
		t.Fatalf("Should not be able to start step = %s with constraints_ = %v", step4.Id, planner.Constraints)
	}

	// Emulate starting step2. Should succeed.
	if ok, _ := planner.CanStartStep(step2); !ok {
		t.Fatalf("Expected to be able to start step = %s with constraints_ = %v", step2.Id, planner.Constraints)
	}

	// mark step3 as done
	//planner.StepStatus.Update(step3.Id, Done)
	planner.UpdateStepStatus(step3.Id, Done)

	// Status:
	// - step1: Done
	// - step2: Running
	// - step3: Done
	// - step4: Queued

	// Emulate starting step4. Should succeed.
	if ok, _ := planner.CanStartStep(step4); !ok {
		t.Fatalf("Expected to be able to start step = %s with constraints_ = %v", step4.Id, planner.Constraints)
	}

	//planner.StepStatus.Update(step4.Id, Running)
	planner.UpdateStepStatus(step4.Id, Running)

	// Status:
	// - step1: Done
	// - step2: Running
	// - step3: Done
	// - step4: Running

	// t.Fatal("fail for debug")

	// compare problem strings for this vs the non-nested version
	// Why are tests for both going ok? Does that mean that these tests doesn't really emulate what the planner is doing?
	// If so, create a test that subscribes to events and makes sure the events comes in a meaningful order (might require a fake delay step)

}

func matchEventOrder(expectedEvents []events.Event, allEvents []events.Event) []events.Event {
	for _, event := range allEvents {
		// fmt.Printf("expectedEvents: %v\n", expectedEvents)
		if len(expectedEvents) == 0 {
			// OK
			return expectedEvents
		}

		fmt.Printf("expected: %#v, current: %#v\n", expectedEvents[0], event)

		if reflect.DeepEqual(event, expectedEvents[0]) {
			// remove first element
			expectedEvents = expectedEvents[1:]
		}
	}

	return expectedEvents
}

func TestRun(t *testing.T) {
	fmt.Println()
	fmt.Println("TestRun")

	hosts := make(map[string]nix.Host)
	opts := common.MorphOptions{}

	constraints_ := make([]constraints.Constraint, 0)
	constraints_ = append(constraints_, constraints.NewConstraint(constraints.LabelSelector{Label: "_", Value: "host"}, 100))
	constraints_ = append(constraints_, constraints.NewConstraint(constraints.LabelSelector{Label: "type", Value: "*"}, 1))
	constraints_ = append(constraints_, constraints.NewConstraint(constraints.LabelSelector{Label: "location", Value: "*"}, 1))
	//constraints_ =
	//= append(constraints_, nix.NewConstraint(nix.LabelSelector{Label: "location", Value: "dc3"}, 1))

	planner := NewPlanner(events.NewManager(), hosts, &opts, constraints_)

	step1 := steps.New().Id("host:1").
		Label("_", "host").
		Label("host", "1").
		Label("location", "dc1").
		Label("type", "web").
		Action(&steps.Delay{MilliSeconds: 10}).
		Build()

	step2 := steps.New().Id("host:2").
		Label("_", "host").
		Label("host", "2").
		Label("location", "dc1").
		Label("type", "db").
		Action(&steps.Delay{MilliSeconds: 10}).
		Build()

	step3 := steps.New().Id("host:3").
		Label("_", "host").
		Label("host", "3").
		Label("location", "dc2").
		Label("type", "web").
		Action(&steps.Delay{MilliSeconds: 10}).
		Build()

	step4 := steps.New().Id("host:4").
		Label("_", "host").
		Label("host", "4").
		Label("location", "dc2").
		Label("type", "db").
		Action(&steps.Delay{MilliSeconds: 10}).
		Build()

	step5 := steps.New().Id("host:5").
		Label("_", "host").
		Label("host", "5").
		Label("location", "dc3").
		Label("type", "web").
		Action(&steps.Delay{MilliSeconds: 10}).
		Build()

	step6 := steps.New().Id("host:6").
		Label("_", "host").
		Label("host", "6").
		Label("location", "dc3").
		Label("type", "db").
		Action(&steps.Delay{MilliSeconds: 10}).
		Build()

	plan := steps.New().Id("root").AddSteps(step1, step2, step3, step4, step5, step6).Parallel().Build()

	pretty.Println(plan)

	expectedEvents := []events.Event{
		events.StepUpdate{StepId: plan.Id, State: Queued},
		events.StepUpdate{StepId: plan.Id, State: Running},
		events.StepUpdate{StepId: step1.Id, State: Queued},
		events.StepUpdate{StepId: step2.Id, State: Queued},
		events.StepUpdate{StepId: step3.Id, State: Queued},
		events.StepUpdate{StepId: step4.Id, State: Queued},
		events.StepUpdate{StepId: step5.Id, State: Queued},
		events.StepUpdate{StepId: step6.Id, State: Queued},
		events.StepUpdate{StepId: step1.Id, State: Running},
		events.StepUpdate{StepId: step4.Id, State: Running},
		events.StepUpdate{StepId: step5.Id, State: Running},
		events.StepUpdate{StepId: step6.Id, State: Running},
	}

	planner.QueueStep(plan)

	err := planner.Run(context.TODO())

	if err != nil {
		t.Fatalf("Planner returned err: %v", err)
	}

	allEvents, _ := planner.EventManager.GetEvents("", 1000000)

	fmt.Println(len(allEvents))

	unmatchedEvents := matchEventOrder(expectedEvents, allEvents)

	if len(unmatchedEvents) > 0 {
		res := make([]string, 0)
		for _, e := range unmatchedEvents {
			res = append(res, fmt.Sprintf("%#v", e))
		}

		t.Fatalf("Events not matched to planner output: %v", res)
	}

	fmt.Printf("%v\n", unmatchedEvents)

	//t.Fatal("fail for debug")
}
