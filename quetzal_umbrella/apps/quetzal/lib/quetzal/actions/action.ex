defmodule Quetzal.Actions.Action do
  @callback run(step :: any) :: any
end
