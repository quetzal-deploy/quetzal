defmodule QuetzalWeb.Dashboard do
  use QuetzalWeb, :live_view

  @topic "state"

  # brug phoenix channels til at opdatere state fra genserveren: https://hexdocs.pm/phoenix/channels.html

  def render(assigns) do
    ~H"""
    <Layouts.app flash={@flash}>
      <QuetzalWeb.PlanComponent.list plans={@plans} current_plan_id={nil} />
    </Layouts.app>
    """
  end

  def mount(params, _session, socket) do
    QuetzalWeb.Endpoint.subscribe(@topic)
    state = Quetzal.Engine.get_state
    {:ok, assign(socket, foo(state))}
  end

  def handle_info(%{topic: topic, event: event, payload: payload}, socket) do
    IO.inspect(topic)
    IO.inspect(event)
    # IO.inspect(payload)
    {
      :noreply,
      assign(
        socket, foo(payload)
      )
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
end
