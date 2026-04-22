defmodule QuetzalWeb.PlanController do
  use QuetzalWeb, :controller

  def show(conn, _params) do
    state = Quetzal.Engine.get_state
    render(conn, :show, steps: state[:steps], step_states: state[:step_states])
  end
end
