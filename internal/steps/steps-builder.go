package steps

import "github.com/google/uuid"

type StepBuilder struct {
	step *Step
}

func New() *StepBuilder {
	return &StepBuilder{
		step: &Step{
			Id:          uuid.New().String(),
			Description: "",
			ActionName:  "none",
			Action:      &None{},
			Parallel:    false,
			Steps:       make([]Step, 0),
			OnFailure:   "",
			DependsOn:   make([]string, 0),
			CanResume:   true,
			Labels:      make(map[string]string),
		},
	}
}

func (sb *StepBuilder) Build() Step {
	return *sb.step
}

func (sb *StepBuilder) Action(action Action) *StepBuilder {
	sb.step.Action = action
	sb.step.ActionName = action.Name() // FIXME: Try to get rid of ActionName

	return sb
}

func (sb *StepBuilder) Description(description string) *StepBuilder {
	sb.step.Description = description

	return sb
}

func (sb *StepBuilder) Id(id string) *StepBuilder {
	sb.step.Id = id

	return sb
}

func (sb *StepBuilder) Label(key string, value string) *StepBuilder {
	sb.step.Labels[key] = value

	return sb
}

func (sb *StepBuilder) Labels(labels map[string]string) *StepBuilder {
	for key, value := range labels {
		sb.Label(key, value)
	}

	return sb
}

func (sb *StepBuilder) DoNothingOnFailure() *StepBuilder {
	sb.step.OnFailure = ""

	return sb
}

func (sb *StepBuilder) ExitOnFailure() *StepBuilder {
	sb.step.OnFailure = "exit"

	return sb
}

func (sb *StepBuilder) RetryOnFailure() *StepBuilder {
	sb.step.OnFailure = "retry"

	return sb
}

func (sb *StepBuilder) Parallel() *StepBuilder {
	sb.step.Parallel = true

	return sb
}

func (sb *StepBuilder) Sequential() *StepBuilder {
	sb.step.Parallel = false

	return sb
}

func (sb *StepBuilder) DisableResume() *StepBuilder {
	sb.step.CanResume = false

	return sb
}

func (sb *StepBuilder) EnableResume() *StepBuilder {
	sb.step.CanResume = true

	return sb
}

func (sb *StepBuilder) AddSteps(steps ...Step) *StepBuilder {
	sb.step.Steps = append(sb.step.Steps, steps...)

	return sb
}

func (sb *StepBuilder) AddSequentialSteps(steps ...Step) *StepBuilder {
	for _, step := range steps {
		if len(sb.step.Steps) > 0 {
			// If there's existing steps, get the ID of the last one and add it as dependency to the current one
			step.DependsOn = append(step.DependsOn, sb.step.Steps[len(sb.step.Steps)-1].Id)
		}

		sb.AddSteps(step)
	}

	return sb
}

func (sb *StepBuilder) AddDependencies(dependencies ...string) *StepBuilder {
	sb.step.DependsOn = append(sb.step.DependsOn, dependencies...)

	return sb
}

func (sb *StepBuilder) AddDependenciesSteps(dependencies ...Step) *StepBuilder {
	for _, dependency := range dependencies {
		sb.AddDependencies(dependency.Id)
	}

	return sb
}
