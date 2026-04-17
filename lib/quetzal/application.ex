defmodule Quetzal.Application do
  # See https://hexdocs.pm/elixir/Application.html
  # for more information on OTP Applications
  @moduledoc false

  use Application

  @impl true
  def start(_type, _args) do
    children = [
      # Starts a worker by calling: Quetzal.Worker.start_link(arg)
      # {Quetzal.Worker, arg}
      Quetzal.Engine
    ]

    # See https://hexdocs.pm/elixir/Supervisor.html
    # for other strategies and supported options
    opts = [strategy: :one_for_one, name: Quetzal.Supervisor]
    Supervisor.start_link(children, opts)
  end
end
