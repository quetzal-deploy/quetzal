defmodule QuetzalWeb.PageController do
  use QuetzalWeb, :controller

  def home(conn, _params) do
    state = Quetzal.Engine.get_state
    render(conn, :home, steps: state[:steps], step_states: state[:step_states])
  end
end
