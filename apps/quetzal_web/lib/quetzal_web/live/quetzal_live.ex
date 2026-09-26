defmodule QuetzalWeb.Dashboard do
  use QuetzalWeb, :live_view

  @topic "state"

  # brug phoenix channels til at opdatere state fra genserveren: https://hexdocs.pm/phoenix/channels.html

  def render(assigns) do
    ~H"""
    <Layouts.app flash={@flash}>
      <QuetzalWeb.RepositoryComponent.list repositories={@repositories} />
      <QuetzalWeb.PlanComponent.list plans={@plans} current_plan_id={nil} />
    </Layouts.app>
    """
  end

  def mount(params, _session, socket) do
    QuetzalWeb.Endpoint.subscribe("state")
    QuetzalWeb.Endpoint.subscribe("repositories")

    state = Quetzal.Engine.get_state
    repositories = Quetzal.Git.RepoManager.get_repositories
    # {:ok, assign(socket, foo(state))}
    {
      :ok,
      socket
      |> assign(foo(state))
      |> assign(:repositories, repositories)
    }
  end

  # todo: rename this state to something reflecting it's the state from Quetzal.Engine
  def handle_info(%{topic: "state", event: event, payload: payload}, socket) do
    IO.puts("#{__MODULE__}: topic: state")
    IO.inspect("topic: state")
    IO.inspect(event)
    # IO.inspect(payload)
    {
      :noreply,
      assign(
        socket, foo(payload)
      )
    }
  end

  def handle_info(%{topic: "repositories", event: _event, payload: repositories}, socket) do
    IO.puts("#{__MODULE__}: topic: repositories")
    {
      :noreply,
      socket
      |> assign(:repositories, repositories)
    }
  end

  defp foo(state) do
    %{
      plans: state.plans,
      # steps: state.steps,
      # step_states: state.step_states,
      # step_states_simple: Map.new(state.step_states, fn {id, step_state} ->
      #   case step_state do
      #     {x, _} ->
      #       {id, x}

      #     x ->
      #       {id, x}
      #   end
      # end),
    }
  end

  def steps_with_dep(steps, parent_step_id) do
    steps
    |> Enum.filter(fn {step_id, step} ->
      %{"dependencies" => dependencies} = step
      case parent_step_id do
        nil ->
          dependencies == []

        _ ->
          parent_step_id in dependencies
      end
    end)
    |> Enum.map(fn {id, _} -> id end)
  end

  def handle_event("schedule_plan", %{"repository" => repository, "deployment" => deployment, "plan_id" => plan_id} = params, socket) do
    IO.inspect(params)
    Quetzal.Engine.schedule_plan(repository, deployment, plan_id)
    {:noreply, socket}
  end
end
