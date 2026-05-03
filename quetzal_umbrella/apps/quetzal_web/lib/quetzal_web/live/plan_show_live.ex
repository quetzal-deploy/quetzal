defmodule QuetzalWeb.PlanLive do
  use QuetzalWeb, :live_view

  @topic "state"

  def render(assigns) do
    ~H"""
    <Layouts.app flash={@flash}>
      engine paused: {@paused}
      <button :if={!@paused} phx-click="pause">pause</button>
      <button :if={@paused} phx-click="unpause">unpause</button>
      <button :if={@paused} phx-click="tick">tick</button>

      <div id="plans">
        <QuetzalWeb.PlanComponent.list plans={@plans} current_plan_id={@plan_id} />
        <div id="plan-display">
          <QuetzalWeb.PlanComponent.tree
            steps={@plans[@plan_id].steps}
            step_states={@plans[@plan_id].step_states}
            parent_id={@plans[@plan_id]["plan"]["id"]}
            foo=""
          />
        </div>
      </div>
    </Layouts.app>
    """
  end

  def mount(params, _session, socket) do
    plan_id = params["plan_id"]

    QuetzalWeb.Endpoint.subscribe(@topic)
    state = Quetzal.Engine.get_state

    plan = get_in(state, [:plans, plan_id])

    case plan do
      nil ->
        # FIXME: create custom 404 page here that still shows the list of existing plans
        raise QuetzalWeb.PlanLive.PlanNotFound

      plan ->
        {
          :ok,
          assign(
            socket,
            Map.merge(foo(state), %{
              plan_id: plan_id,
            })
          )
        }
    end
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

  def handle_event("tick", _params, socket) do
    IO.puts "Not implemented: Single tick"
    {:noreply, socket}
  end

  def handle_event("pause", _params, socket) do
    Quetzal.Engine.pause
    {:noreply, socket}
  end

  def handle_event("unpause", _params, socket) do
    Quetzal.Engine.unpause
    {:noreply, socket}
  end

  def handle_event("plan_pause", %{"plan_id" => plan_id}, socket) do
    Quetzal.Engine.plan_pause(plan_id)
    {:noreply, socket}
  end

  def handle_event("plan_unpause", %{"plan_id" => plan_id}, socket) do
    Quetzal.Engine.plan_unpause(plan_id)
    {:noreply, socket}
  end

  defp foo(state) do
    %{
      paused: state.paused,
      plans: state.plans,
      # steps: state.steps,
      # step_states: state.step_states,
    }
  end
end



defmodule QuetzalWeb.PlanLive.PlanNotFound do
  defexception message: "plan not found", plug_status: 404
end
