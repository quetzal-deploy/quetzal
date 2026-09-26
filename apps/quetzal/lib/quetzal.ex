defmodule Quetzal do
  @moduledoc """
  Documentation for `Quetzal`.
  """

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

  def deployment_plans(evaluator, deployment) do
    nix_instantiate(
      evaluator,
      [
        "--arg", "deployment", deployment,
        "--attr", "plans2",
      ],
      Quetzal.Schemas.any()
      # Quetzal.Schemas.list_of_strings()
    )
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
