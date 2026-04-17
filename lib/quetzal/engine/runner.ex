defmodule Quetzal.Engine.Runner do
  @actions %{
    "none" => Quetzal.Actions.None,
    "is-online" => Quetzal.Actions.IsOnline,
    "build" => Quetzal.Actions.Build,
    # "deploy-switch" => Quetzal.Actions.DeploySwitch,
  }

  def run_step(%{"id" => id} = step) do
    IO.puts("running: #{id}")
    Process.sleep(100)
    do_run(step)
    IO.puts("done: #{id}")
    Quetzal.Engine.mark_step_done(id)
  end

  defp do_run(%{"action" => action} = step) do
    case Map.fetch(@actions, action) do
      {:ok, module} -> module.run(step)
      :error -> {:error, {:unknown_action, action}}
    end
  end
end
