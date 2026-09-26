defmodule Quetzal.Actions.Morph.Build do
  @behaviour Action

  @impl true
  def run(step), do: IO.inspect(step)
end
