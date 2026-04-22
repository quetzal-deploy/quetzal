defmodule Quetzal do
  @moduledoc """
  Documentation for `Quetzal`.
  """

  @doc """
  Hello world.

  ## Examples

      iex> Quetzal.hello()
      :world

  """
  def hello do

    morph_evaluator = "/home/adtu/src/quetzal-evaluators/quetzal-morph.nix"
    deployment = "/home/adtu/src/quetzal-rs/test/deployments/1.nix"
    {:ok, resources} = deployment_resources(morph_evaluator, deployment)

    args = "/home/adtu/src/quetzal-rs/tmp/build-args-file.json"
    {:ok, plan} = deployment_plan(morph_evaluator, deployment, "switch", args)

    # IO.inspect(resources)
    # IO.inspect(plan)

    {:ok, plan_id} = Quetzal.Engine.schedule_plan(plan)
    IO.puts "Scheduled plan with id=#{plan_id}"
    # make an engine that is a GenServer where plans (and substeps) can be submitted as jobs

  end

  def nix_instantiate(file, args, schema) do
    result = System.cmd("nix-instantiate", [
      "--eval",
      "--strict",
      "--json",
      file,
    ] ++ args)

    with {output, 0} <- result,
         {:ok, data} <- Jason.decode(output),
         {:ok, validated} <- Norm.conform(data, schema)
    do
         {:ok, validated}
    else
      {:error, reason} -> {:error, reason}
      {output, _exit_code} -> {:err, output}
    end
  end

  def deployment_resources(evaluator, deployment) do
    nix_instantiate(
      evaluator,
      [
        "--arg", "deployment", deployment,
        "--attr", "resources",
      ],
      Quetzal.Schemas.deployment_resources()
    )
  end

  def deployment_plan(evaluator, deployment, plan, args_file) do
    nix_instantiate(
      evaluator,
      [
        "--arg", "deployment", deployment,
        "--argstr", "plan", plan,
        "--argstr", "args", args_file,
        "--attr", "plans",
      ],
      Quetzal.Schemas.plan()
    )
  end
end
