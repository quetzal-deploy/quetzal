defmodule Quetzal.Schemas do
  import Norm

  def deployment_resources do
    host_spec = selection(
      schema(%{}),
      :all
    )

    selection(
      schema(
        %{
          "hosts" => map_of(spec(is_binary()), host_spec),
          "inputs" => map_of(spec(is_binary()), spec(is_binary())),
        }
      ),
      :all)
  end

  def plan do
    schema(%{})
  end
end
