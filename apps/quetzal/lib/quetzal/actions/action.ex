defmodule Quetzal.Actions.Action do
  # tighten to maps
  @callback run(cache :: any, step :: any) :: any
end
