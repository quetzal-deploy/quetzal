defmodule Quetzal.Engine.Runner do
  @actions %{
    "none" => Quetzal.Actions.None,
    "is-online" => Quetzal.Actions.IsOnline,
    "build" => Quetzal.Actions.Build,
    "deploy-switch" => Quetzal.Actions.DeploySwitch,
  }

  def run_step(plan_id, %{"id" => step_id, "action" => action} = step) do
    # TODO: implement cache between steps
    cache = %{}
    IO.puts("running: #{step_id}")
    # Process.sleep(2000)
    Process.sleep(500)
    case Map.fetch(@actions, action) do
      {:ok, module} -> module.run(cache, step)
      :error -> {:error, {:unknown_action, action}}
    end
    IO.puts("done: #{step_id}")
    Quetzal.Engine.mark_step_done(plan_id, step_id)
  end
end
