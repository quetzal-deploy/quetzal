defmodule Quetzal.Actions.Morph.DeployBuild do
  @behaviour Action

  @impl true
  def run(step), do: IO.inspect(step)
end
