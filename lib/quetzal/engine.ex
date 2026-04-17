defmodule Quetzal.Engine do
  use GenServer

  # FIXME: This is probably invalid
  @type status :: :new | :ready | :scheduled | {:blocked_by, []} | :running | {:awaiting_children, []} | :done | :failed

  def start_link(_opts \\ []) do
    GenServer.start_link(__MODULE__, %{}, name: __MODULE__)
  end

  def init(_state) do
    IO.puts("engine: hello")
    {
      :ok,
      %{
        steps: {},
        pids: {},
        step_states: {},
      }
    }
  end

  def schedule_plan(plan) do
    # GenServer.call(__MODULE__, %{schedule_plan: plan})
    # fixme: do something more, e.g. handle constraints too
    GenServer.call(__MODULE__, %{schedule_plan: Map.get(plan, "plan")})
  end

  def schedule_action(action) do
    GenServer.call(__MODULE__, %{schedule_action: action})
  end

  def handle_call(%{schedule_plan: plan}, _from, state) do
    # TODO: Have one GenServer for each plan, and move this to init
    steps = Quetzal.Steps.flatten(plan)

    ids = Map.keys(steps)
    pids = Enum.reduce(ids, %{}, fn id, acc -> Map.put(acc, id, nil) end)
    step_states = Enum.reduce(ids, %{}, fn id, acc -> Map.put(acc, id, :new) end)

    new_state = %{ state |
      steps: steps,
      pids: pids,
      step_states: step_states,
    }

    IO.inspect(new_state)
    GenServer.cast(__MODULE__, :tick)
    IO.puts("tick sent")

    {:reply, :ok, new_state}
  end

  def get_state do
    GenServer.call(__MODULE__, :get_state)
  end

  def handle_call(:get_state, _from, state) do
    {:reply, state, state}
  end

  def handle_cast(:tick, state) do
    IO.puts("ticking")

    desired_states = []

    new_state = %{ state | step_states: update_step_states(state) }

    runnable_step = get_runnable_step(new_state)

    new_state = case runnable_step do
      {:ok, step_id} ->
        step_description = get_in(state, [:steps, step_id, "description"])
        IO.puts "Scheduling step: #{step_id}: #{step_description}"
        GenServer.cast(__MODULE__, %{run_step: step_id})
        put_in(new_state, [:step_states, step_id], :scheduled)

      {:error, reason} ->
        IO.puts "Can't schedule more steps: #{reason}"
        new_state
    end

    {:noreply, new_state}
  end

  def get_step_state(state, step) do
    Map.get(state[:step_states], step)
  end

  def update_step_states(state) do
    step_states = state[:step_states]

    updated_step_states = step_states
    |> Map.new(fn {id, step_state} ->
      new_step_state = case step_state do
        :new ->
          case state[:steps][id]["dependencies"] do
            [] -> :ready
            dependencies -> {:blocked_by, dependencies}
          end

        {:blocked_by, []} ->
          :ready

        {:blocked_by, step_ids} ->
          new_step_ids = Enum.filter(step_ids, fn id ->
            get_step_state(state, id) != :done
          end)

          case new_step_ids do
            [] -> :ready
            _ -> {:blocked_by, new_step_ids}
          end

        :running ->
          :running # TODO: check pid exists and is running, if not set to :failed. Potential for race conditions

        {:awaiting_children, [child_ids]} ->
          new_child_ids = Enum.filter(child_ids, fn id ->
            get_step_state(state, id) != :done
          end)

          case new_child_ids do
            [] -> :ready
            _ -> {:awaiting_children, new_child_ids}
          end

        # catch all for states that can't be updated automatically
        _ ->
          step_state

      end

      {id, new_step_state}
    end)
  end

  def get_steps_with_state(state, step_state) do
    Map.filter(state[:step_states], fn {id, step_state_} -> step_state == step_state_ end)
    |> Map.keys
  end

  def get_runnable_step(state) do
    case get_steps_with_state(state, :ready) do
      [ step | _ ] -> {:ok, step}
      [] -> {:error, "no steps marked ready"}
    end
  end

  def get_step(state, step_id), do: get_in(state, [:steps, step_id])
  def get_step_state(state, step_id), do: get_in(state, [:step_states, step_id])
  def set_step_state(state, step_id, step_state), do: put_in(state, [:step_states, step_id], step_state)
  def get_step_pid(state, step_id), do: get_in(state, [:pids, step_id])
  def set_step_pid(state, step_id, step_state), do: put_in(state, [:pids, step_id], step_state)

  def handle_cast(%{run_step: step_id}, state) do
    case get_step_state(state, step_id) do
      :scheduled ->
        step = get_in(state, [:steps, step_id])

        pid = spawn(fn -> Quetzal.Engine.Runner.run_step(step) end)

        new_state = state
        |> set_step_state(step_id, :running)
        |> set_step_pid(step_id, pid)

        GenServer.cast(__MODULE__, :tick)
        {:noreply, new_state}

      current_state ->
        {:stop, "Bug or inconsistent state. run_step was called on a step that wasn't scheduled, but instead had state=#{current_state}", state}
    end
  end

  def mark_step_done(step_id) do
    GenServer.call(__MODULE__, %{transition_step: step_id, to: :done})
  end

  def mark_step_failed(step_id) do
    GenServer.call(__MODULE__, %{transition_step: step_id, to: :failed})
  end

  def handle_call(%{transition_step: step_id, to: step_state}, _from, state) do
    new_state = put_in(state, [:step_states, step_id], step_state)
    GenServer.cast(__MODULE__, :tick)
    {:reply, :ok, new_state}
  end
end
