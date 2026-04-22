defmodule QuetzalWeb.PlanHTML do
  @moduledoc """
  This module contains pages rendered by PageController.

  See the `plan_html` directory for all templates available.
  """
  use QuetzalWeb, :html

  embed_templates "html/plan/*"
end
