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
        paused: false,
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

  def initialize_step_states(ids) do
    Enum.reduce(ids, %{}, fn id, acc -> Map.put(acc, id, :new) end)
  end

  def initialize_pids(ids) do
    Enum.reduce(ids, %{}, fn id, acc -> Map.put(acc, id, nil) end)
  end

  def initialize_plan(%{"plan" => plan_inner, "constraints" => constraints} = plan) do
    steps = Quetzal.Steps.flatten(plan_inner)
    ids = Map.keys(steps)
    pids = initialize_pids(ids)
    step_states = initialize_step_states(ids)

    Map.merge(plan, %{
      completed: nil,
      description: plan_inner["description"],
      steps: steps,
      pids: pids,
      step_states: step_states,
    })
    # only add ID if doesn't exist - useful if re-initializing the plan
    |> Map.put_new(:id, UUID.uuid4())
    |> Map.put_new(:paused, false)
    |> Map.put_new(:scheduled, DateTime.utc_now)
  end

  def handle_call(%{schedule_plan: plan}, _from, state) do
    %{"plan" => plan_inner, "constraints" => constraints} = plan

    plan_id = UUID.uuid4()

    # TODO: Have one GenServer for each plan, and move this to init
    steps = Quetzal.Steps.flatten(plan_inner)

    ids = Map.keys(steps)
    pids = initialize_pids(ids)
    step_states = initialize_step_states(ids)

    plan_with_metadata = initialize_plan(plan)
    # |> IO.inspect()

    new_state = %{ state |
      # plans: Map.put(state.plans, plan_id, plan_with_metadata),
      plans: Map.put(state.plans, plan_with_metadata.id, plan_with_metadata),
      # steps: steps,
      # pids: pids,
      # step_states: step_states,
    }

    GenServer.cast(__MODULE__, :broadcast_state)
    GenServer.cast(__MODULE__, {:tick, false})
    IO.puts("tick sent")

    {:reply, {:ok, plan_id}, new_state}
  end

  def get_state do
    GenServer.call(__MODULE__, :get_state)
  end

  def handle_call(:get_state, _from, state) do
    {:reply, state, state}
  end

  def tick do
    GenServer.cast(__MODULE__, {:tick, false})
  end

  def force_tick do
    GenServer.cast(__MODULE__, {:tick, true})
  end

  def tick_plan(plan_id) do
    GenServer.cast(__MODULE__, %{tick_plan: plan_id})
  end

  def handle_cast({:tick, forced}, %{paused: paused} = state) when not forced and paused  do
    IO.puts "Tick skipped: Engine is paused"
    {:noreply, state}
  end

  def handle_cast({:tick, forced}, %{paused: paused} = state) when forced or not paused do
    IO.puts("ticking")

    desired_states = []

    new_plans_state = Map.new(state[:plans], fn {plan_id, plan} ->

      if plan.completed == nil and plan.paused == false do
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

    # IO.inspect plan_completed
    # IO.inspect(step_states)

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
      step = get_in(plan, [:steps, id])

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
          #   # keep all steps that are not done
            get_step_state(plan, id) != :done
          #   # FIXME: also remove steps that are awaiting_children if this step is in the list of children
          #   # FIXME: ^ children are actually not scheduled yet
          #   # TODO: FIX THIS NOW
          #   # probably has to run in a second loop, to avoid running on partially stale data
          end)

          case new_step_ids do
            [] -> :ready
            _ -> {:blocked_by, new_step_ids}
          end

        :running ->
          :running # TODO: check pid exists and is running, if not set to :failed. Potential for race conditions

        :awaiting_children ->
          GenServer.cast(__MODULE__, %{tick_plan: plan.id})
          # IO.inspect "await chld <none>"
          children = get_in(plan, [:steps, id]).children
          {:awaiting_children, children}

        {:awaiting_children, []} ->
          # IO.inspect "await chld empty"
          # tick required to bubble up the done state
          # GenServer.cast(__MODULE__, :tick)
          GenServer.cast(__MODULE__, %{tick_plan: plan.id})
          :done

        {:awaiting_children, child_ids} when is_list(child_ids) ->
          # IO.inspect "await chld list"
          new_child_ids = Enum.filter(child_ids, fn id ->
            # IO.inspect id
            get_step_state(plan, id) != :done
          end)

          # new_child_ids = child_ids -- plan.steps_done

          case new_child_ids do
            [] -> :done
            _ -> {:awaiting_children, new_child_ids}
          end

          # case new_child_ids do
          #   [] -> :ready # FIXME: :ready must be wrong!!!
          #   _ -> {:awaiting_children, new_child_ids}
          # end

        # catch all for states that can't transition automatically
        _ ->
          step_state

      end

      {id, new_step_state}
    end)
  end

  # If tuple: Unwraps step states that are tuples with an argument, and returns only first part of the tuple.
  # If not tuple, returns identity
  def simplify_step_state(step_state) do
    case step_state do
      {step_state_, _ } -> step_state_
      _ -> step_state
    end
  end

  def get_steps_with_state(plan, step_state) do
    get_steps_with_states(plan, [step_state])
  end

  def get_steps_with_states(plan, step_states) do
    Map.filter(plan[:step_states], fn {id, step_state_} ->
      simplify_step_state(step_state_) in step_states
    end)
    |> Map.keys
  end

  # combines matching labels and constraints
  def constraints_x_labels(constraints, labels) do
    Enum.reduce(constraints, [], fn constraint, acc ->
      %{"selector" => %{"label" => c_label, "value" => c_value}} = constraint

      acc ++ Enum.reduce(labels, [], fn {label, value}, acc2 ->
        res = cond do
          label == c_label and (value == c_value or c_value == "*") ->
            [ %{label: label, value: value, constraint: constraint} ]
          true ->
            []
        end
        acc2 ++ res
      end)
    end)
  end

  def get_steps_with_label(plan, label, value) do
    Enum.reduce(plan.steps, [], fn {step_id, step}, acc ->
      labels = step["labels"]
      cond do
        Map.has_key?(labels, label) and labels[label] == value ->
          [step_id | acc]
        true ->
          acc
      end
    end)
  end

  def is_unconstrained(plan, step_id) do
    running_states = [
      :scheduled,
      :running,
      :awaiting_children,
      :failed,
    ]

    constraints = plan["constraints"]
    step = get_in(plan, [:steps, step_id])
    labels = step["labels"]
    IO.puts("step: #{step_id} labels: #{inspect(labels)}")

    constraints
    # find the matching constraints and labels
    |> constraints_x_labels(labels)
    # |> IO.inspect()
    # validate each constraint match
    |> Enum.map(fn %{label: label, value: value, constraint: constraint} ->
      %{"maxUnavailable" => max_unavailable, "selector" => %{"label" => c_label, "value" => c_value}} = constraint

      # matching_labels = get_steps_with_label(plan, label, value)
      # matching_states = get_steps_with_states(plan, running_states)
      matching_steps = MapSet.intersection(
        MapSet.new(get_steps_with_label(plan, label, value)),
        MapSet.new(get_steps_with_states(plan, running_states))
      )
      |> MapSet.to_list
      IO.puts("constraint match: step: #{step_id} #{label}=#{value} => max_unavailable=#{max_unavailable} matching: #{inspect(matching_steps)}")
      IO.puts("!! ZEBRA")
      IO.inspect(matching_steps)

      cond do
        length(matching_steps) < max_unavailable ->
          :ok
        true ->
          {:constrained, %{steps: matching_steps}}

      end
    end)
    # check all constraints evaluted to :ok
    |> Enum.all?(fn result -> result == :ok end)
  end

  def get_runnable_step(plan) do
    # add labels as arg, find steps running (or rather not started and not done) matching each label

    steps_ready = get_steps_with_state(plan, :ready)
    |> Enum.filter(fn step_id -> is_unconstrained(plan, step_id) end)

    case steps_ready do
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

        GenServer.cast(__MODULE__, {:tick, false})
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
    GenServer.cast(__MODULE__, {:tick, false})

    new_state = case step_state do
      :awaiting_children ->
        step = get_step(state, plan_id, step_id)
        # FIXME: this fails somehow. Loop over the children, subtract step_id from their dependencies, and create new state based on that.
        Enum.reduce(step.children, state, fn child_id, acc ->
          step_state = get_step_state(state, plan_id, child_id)

          # IO.inspect("child: #{child_id}, state: #{inspect(step_state)}")

          case step_state do
            {:blocked_by, child_ids} when is_list(child_ids) ->
              new_child_ids = Enum.reject(child_ids, fn child_id -> child_id == step_id end)
              new_step_state = {:blocked_by, new_child_ids}

              # IO.puts("child: #{child_id},\n- old state: #{inspect(step_state)},\n- new state: #{inspect(new_step_state)}")

              acc
              |> put_in([:plans, plan_id, :step_states, child_id], new_step_state)
            _ ->
              acc

          end
        end)

      _ ->
        state

    end
    |> set_step_state(plan_id, step_id, step_state)
    # |> IO.inspect()
    |> then(fn new_state -> {:reply, :ok, new_state} end)
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

  def pause do
    GenServer.call(__MODULE__, :pause)
  end

  def unpause do
    GenServer.call(__MODULE__, :unpause)
  end

  def plan_pause(plan_id) do
    GenServer.call(__MODULE__, {:plan_pause, plan_id})
  end

  def plan_unpause(plan_id) do
    GenServer.call(__MODULE__, {:plan_unpause, plan_id})
  end

  def plan_reset(plan_id) do
    GenServer.call(__MODULE__, {:plan_reset, plan_id})
  end

  def handle_call(:pause, _from, state) do
    GenServer.cast(__MODULE__, :broadcast_state)
    {:reply, :ok, put_in(state, [:paused], true)}
  end

  def handle_call(:unpause, _from, state) do
    GenServer.cast(__MODULE__, :broadcast_state)
    GenServer.cast(__MODULE__, {:tick, false})
    {:reply, :ok, put_in(state, [:paused], false)}
  end

  def handle_call({:plan_pause, plan_id}, _from, state) do
    GenServer.cast(__MODULE__, :broadcast_state)
    {:reply, :ok, put_in(state, [:plans, plan_id, :paused], true)}
  end

  def handle_call({:plan_unpause, plan_id}, _from, state) do
    GenServer.cast(__MODULE__, :broadcast_state)
    GenServer.cast(__MODULE__, %{tick_plan: plan_id})
    {:reply, :ok, put_in(state, [:plans, plan_id, :paused], false)}
  end

  def handle_call({:plan_reset, plan_id}, _from, state) do
    plan = get_in(state, [:plans, plan_id])

    state = state
    |> put_in([:plans, plan_id], initialize_plan(plan))
    |> put_in([:plans, plan_id, :paused], true)

    GenServer.cast(__MODULE__, :broadcast_state)
    {:reply, :ok, state}
  end
end
