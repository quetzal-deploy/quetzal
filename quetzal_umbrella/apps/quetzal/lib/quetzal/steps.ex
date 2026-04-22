defmodule Quetzal.Steps do
  def default_step do
    %{
      "id" => UUID.uuid4(),
      "dependencies" => [],
    }
  end

  def add_defaults(step \\ %{}), do: Map.merge(default_step(), step)

  def flatten(step) do
      # ensure step has an ID and other defaults
      step = add_defaults(step)
      %{"id" => id} = step


      child_steps = Map.get(step, "steps", [])
      # give defaults to children
      |> Enum.map(&add_defaults/1)

      child_step_ids = Enum.map(child_steps, fn %{"id" => child_id} -> child_id end)

      step_without_children = Map.delete(step, "steps")
      |> Map.put(:children, child_step_ids)

      # Map.get(step, "steps", [])
      # give defaults to children
      # |> Enum.map(&add_defaults/1)
      child_steps
      # add each parent as dependecy to its children
      |> Enum.map(fn %{"dependencies" => dependencies} = child ->
        %{ child | "dependencies" => [ id | dependencies ] }
      end)
      # turn in to map while recursing
      |> Enum.reduce(%{id => step_without_children}, fn child, acc ->
        Map.merge(acc, flatten(child))
      end)
    end
end
