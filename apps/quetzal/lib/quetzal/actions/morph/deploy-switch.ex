defmodule Quetzal.Actions.Morph.DeploySwitch do
  @behaviour Quetzal.Actions.Action

  @impl true
  def run(_cache, step), do: IO.inspect(step)
end
