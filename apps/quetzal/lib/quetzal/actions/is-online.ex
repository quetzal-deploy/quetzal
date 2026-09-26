defmodule Quetzal.Actions.IsOnline do
  @behaviour Action

  @impl true
  def run(step), do: IO.inspect(step)
end
