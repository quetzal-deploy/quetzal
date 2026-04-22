defmodule Quetzal.Actions.DeploySwitch do
  @behaviour Action

  @impl true
  def run(step), do: IO.inspect(step)
end
