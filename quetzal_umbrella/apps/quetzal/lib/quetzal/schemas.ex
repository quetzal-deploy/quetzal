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

  def list_of_strings do
    # list of strings
    coll_of(spec(is_binary()))
  end

  def any do
    spec(fn _ -> true end)
  end

  def plan do
    schema(%{})
  end
end
