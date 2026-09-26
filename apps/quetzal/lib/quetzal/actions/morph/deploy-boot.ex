defmodule Quetzal.Actions.Morph.DeployBuild do
  @behaviour Quetzal.Actions.Action

  @impl true
  def run(_cache, step), do: IO.inspect(step)
end
