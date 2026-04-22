defmodule Quetzal.Engine do
  use GenServer

  # FIXME: This is probably invalid
  @type status :: :new | :ready | :scheduled | {:blocked_by, []} | :running | {:awaiting_children, []} | :done | :failed
  # new: initial state after adding
  # blocked_by: waiting for dependencies to finish
  # constrained: step is ready but cannot start due to one or more constraints
  # ready: step can be scheduled. All dependencies have finished, and there's no constraints blocking it. Multiple steps can be marked schedulable, but it's only safe to schedule one step per iteration
  # scheduled: a request to start the step has been made but it haven't yet started
  # running: step is runnning
  # awaiting_children: the action of a step has finished, but is now waiting for its children to finish
  # done: step is done
  # failed: step has failed

  def start_link(_opts \\ []) do
    GenServer.start_link(__MODULE__, %{}, name: __MODULE__)
  end

  def init(_state) do
    IO.puts("engine: hello")
    {
      :ok,
      %{
        plans: %{},
      }
    }
  end

  def schedule_plan(plan) do
    # GenServer.call(__MODULE__, %{schedule_plan: plan})
    # fixme: do something more, e.g. handle constraints too
    # IO.inspect(plan)
    GenServer.call(__MODULE__, %{schedule_plan: plan})
  end

  def schedule_action(action) do
    GenServer.call(__MODULE__, %{schedule_action: action})
  end

  def handle_call(%{schedule_plan: plan}, _from, state) do
    %{"plan" => plan_inner, "constraints" => constraints} = plan

    plan_id = UUID.uuid4()

    # TODO: Have one GenServer for each plan, and move this to init
    steps = Quetzal.Steps.flatten(plan_inner)

    ids = Map.keys(steps)
    pids = Enum.reduce(ids, %{}, fn id, acc -> Map.put(acc, id, nil) end)
    step_states = Enum.reduce(ids, %{}, fn id, acc -> Map.put(acc, id, :new) end)

    plan_with_metadata = Map.merge(plan, %{
      id: plan_id,
      scheduled: DateTime.utc_now,
      completed: nil,
      description: plan_inner["description"],
      steps: steps,
      pids: pids,
      step_states: step_states,
    })

    new_state = %{ state |
      plans: Map.put(state.plans, plan_id, plan_with_metadata),
      # steps: steps,
      # pids: pids,
      # step_states: step_states,
    }

    IO.inspect(new_state)
    GenServer.cast(__MODULE__, :tick)
    IO.puts("tick sent")

    {:reply, {:ok, plan_id}, new_state}
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

    new_plans_state = Map.new(state[:plans], fn {plan_id, plan} ->

      if plan.completed == nil do
        GenServer.cast(__MODULE__, %{tick_plan: plan_id})
      end

      {plan_id, %{plan | step_states: update_step_states(plan) }}
    end)

    new_state = %{ state | plans: new_plans_state }

    GenServer.cast(__MODULE__, :broadcast_state)

    {:noreply, new_state}
  end

  def handle_cast(%{tick_plan: plan_id}, state) do
    plan = get_in(state, [:plans, plan_id])

    new_state = put_in(state, [:plans, plan_id, :step_states], update_step_states(plan))

    runnable_step = get_runnable_step(get_in(new_state, [:plans, plan_id]))

    new_state = case runnable_step do
      {:ok, step_id} ->
        step_description = get_in(new_state, [:plans, plan_id, :steps, step_id, "description"])
        IO.puts "Scheduling step: #{step_id}: #{step_description}"
        GenServer.cast(__MODULE__, %{run_step: step_id, plan_id: plan_id})
        put_in(new_state, [:plans, plan_id, :step_states, step_id], :scheduled)

      {:error, reason} ->
        IO.puts "Can't schedule more steps: #{reason}"
        new_state
    end

    step_states = get_in(new_state, [:plans, plan_id, :step_states])
    plan_completed = Enum.all?(step_states, fn {_, step_state} -> step_state == :done end)

    IO.inspect plan_completed
    IO.inspect(step_states)

    new_state = case plan_completed do
      false ->
        new_state
      true ->
        # fixme: take completion timestamp of last step to finish
        put_in(new_state, [:plans, plan_id, :completed], DateTime.utc_now)
    end

    GenServer.cast(__MODULE__, :broadcast_state)

    {:noreply, new_state}
  end

  def get_step_state(state, step) do
    Map.get(state[:step_states], step)
  end

  def update_step_states(plan) do
    step_states = plan[:step_states]

    updated_step_states = step_states
    |> Map.new(fn {id, step_state} ->
      new_step_state = case step_state do
        :new ->
          case plan[:steps][id]["dependencies"] do
            [] -> :ready
            dependencies -> {:blocked_by, dependencies}
          end

        {:blocked_by, []} ->
          :ready

        {:blocked_by, step_ids} ->
          new_step_ids = Enum.filter(step_ids, fn id ->
            # keep all steps that are not done
            get_step_state(plan, id) != :done
            # FIXME: also remove steps that are awaiting_children if this step is in the list of children
            # FIXME: ^ children are actually not scheduled yet
            # TODO: FIX THIS NOW
          end)

          case new_step_ids do
            [] -> :ready
            _ -> {:blocked_by, new_step_ids}
          end

        :running ->
          :running # TODO: check pid exists and is running, if not set to :failed. Potential for race conditions

        :awaiting_children ->
          children = get_in(plan, [:steps, id]).children
          {:awaiting_children, children}

        {:awaiting_children, [child_ids]} ->
          new_child_ids = Enum.filter(child_ids, fn id ->
            get_step_state(plan, id) != :done
          end)

          case new_child_ids do
            [] -> :ready
            _ -> {:awaiting_children, new_child_ids}
          end

        # catch all for states that can't transition automatically
        _ ->
          step_state

      end

      {id, new_step_state}
    end)
  end

  def get_steps_with_state(state, step_state) do
    get_steps_with_states(state, [step_state])
  end

  def get_steps_with_states(state, step_states) do
    Map.filter(state[:step_states], fn {id, step_state_} -> step_state_ in step_states end)
    |> Map.keys
  end


  def get_runnable_step(state) do
    # add labels as arg, find steps running (or rather not started and not done) matching each label

    steps_ready = get_steps_with_state(state, :ready)

    case get_steps_with_state(state, :ready) do
      [ step | _ ] -> {:ok, step}
      [] -> {:error, "no steps marked ready"}
    end
  end


  def get_step(state, plan_id, step_id), do: get_in(state, [:plans, plan_id, :steps, step_id])
  def get_step_state(state, plan_id, step_id), do: get_in(state, [:plans, plan_id, :step_states, step_id])
  def set_step_state(state, plan_id, step_id, step_state), do: put_in(state, [:plans, plan_id, :step_states, step_id], step_state)
  def get_step_pid(state, plan_id, step_id), do: get_in(state, [:plans, plan_id, :pids, step_id])
  def set_step_pid(state, plan_id, step_id, step_state), do: put_in(state, [:plans, plan_id, :pids, step_id], step_state)

  def handle_cast(%{plan_id: plan_id, run_step: step_id}, state) do
    case get_step_state(state, plan_id, step_id) do
      :scheduled ->
        step = get_step(state, plan_id, step_id)

        pid = spawn(fn -> Quetzal.Engine.Runner.run_step(plan_id, step) end)

        new_state = state
        |> set_step_state(plan_id, step_id, :running)
        |> set_step_pid(plan_id, step_id, pid)

        GenServer.cast(__MODULE__, :tick)
        {:noreply, new_state}

      current_state ->
        {:stop, "Bug or inconsistent state. run_step was called on a step that wasn't scheduled, but instead had state=#{current_state}", state}
    end
  end

  def mark_step_done(plan_id, step_id) do
    GenServer.call(__MODULE__, %{plan_id: plan_id, transition_step: step_id, to: :awaiting_children})
  end

  def mark_step_failed(plan_id, step_id) do
    GenServer.call(__MODULE__, %{plan_id: plan_id, transition_step: step_id, to: :failed})
  end

  def handle_call(%{plan_id: plan_id, transition_step: step_id, to: step_state}, _from, state) do
    new_state = set_step_state(state, plan_id, step_id, step_state)

    GenServer.cast(__MODULE__, :tick)
    {:reply, :ok, new_state}
  end

  # def handle_call(%{transition_step: step_id, to: step_state}, _from, state) do
  #   new_state = put_in(state, [:step_states, step_id], step_state)
  #   GenServer.cast(__MODULE__, :tick)
  #   {:reply, :ok, new_state}
  # end

  def handle_cast(:broadcast_state, state) do
    QuetzalWeb.Endpoint.broadcast("state", "state_updated", state)
    {:noreply, state}
  end
end
