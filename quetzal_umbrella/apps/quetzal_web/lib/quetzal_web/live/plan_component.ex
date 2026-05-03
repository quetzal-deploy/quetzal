defmodule QuetzalWeb.PlanComponent do
  use QuetzalWeb, :verified_routes
  use Phoenix.Component

  def list(assigns) do
    ~H"""
    <div class="plan-list">
      <% plans = Enum.sort(Map.values(@plans), &DateTime.compare(&1.scheduled, &2.scheduled) != :lt) %>
      <.plan_menu :for={plan <- plans} plan={plan} current_plan_id={@current_plan_id} />
    </div>
    """
  end

  def plan_menu(assigns) do
    ~H"""
    <% stats = plan_stats(@plan) %>
    <.link href={~p"/plans/#{@plan.id}"} class={["plan", stats.state, @current_plan_id == @plan.id && "active"]}>
      <div class="state">
        <button :if={!@plan.paused} phx-click="plan_pause" phx-value-plan_id={@plan.id}>running</button>
        <button :if={@plan.paused} phx-click="plan_unpause" phx-value-plan_id={@plan.id}>paused</button>
      </div>
      <div class="timings">
        <div class="start">{display_time(@plan.scheduled)}</div>
        <div class="end">{display_time(@plan.completed)}</div>
      </div>
      <div class="description">{ @plan.description }</div>
      <div class="info">
        steps: {stats.steps.total} |
        running: {stats.steps.running} |
        done: {stats.steps.done}
      </div>
    </.link>
    """
  end

  def tree(assigns) do
    ~H"""
    <ul class="step-tree">
      <% step=@steps[@parent_id] %>
      <% step_state=simplify_step_state(@step_states[@parent_id]) %>
      <li class={["step", step_state]}>{step["description"]}</li>
      <li :for={step_id <- step.children}>
        <.tree steps={@steps} step_states={@step_states} parent_id={step_id} />
      </li>
    </ul>
    """
  end

  def steps_with_dep(steps, parent_step_id) do
    steps
    |> Enum.filter(fn {step_id, step} ->
      %{"dependencies" => dependencies} = step
      case parent_step_id do
        nil ->
          dependencies == []

        "" ->
          dependencies == []

        _ ->
          parent_step_id in dependencies
      end
    end)
    |> Enum.map(fn {id, _} -> id end)
  end

  def simplify_step_state(step_state) do
    case step_state do
      {x, _} -> x
      x -> x
    end
  end

  def plan_stats(plan) do
    steps = %{
      total: Enum.count(plan.step_states),
      done: Enum.count(plan.step_states, fn {_, step_state} -> step_state == :done end),
      running: Enum.count(plan.step_states, fn {_, step_state} -> step_state == :running end),
    }

    %{
      steps: steps,
      done: steps.total == steps.done,
      state: case steps.total == steps.done do
        true -> :done
        false -> :running # fix this to be more specific
      end
    }
  end

  def display_time(dt) do
    case dt do
      nil -> ""
      _ -> Calendar.strftime(dt, "%H:%M")
    end
  end
end
