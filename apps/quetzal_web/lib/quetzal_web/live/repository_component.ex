defmodule QuetzalWeb.RepositoryComponent do
  use QuetzalWeb, :verified_routes
  use Phoenix.Component

  def list(assigns) do
    ~H"""
    <div class="repository-list">
      {inspect(@repositories, pretty: true)}
      <table>
        <th>
          <td>repository</td>
          <td>origin</td>
          <td>ready</td>
          <td>commit</td>
        </th>
        <tr :for={ {id, repository} <- @repositories}>
          <td>{id}</td>
          <td>{repository.origin}</td>
          <td>{repository.ready}</td>
          <td>
            <%= if repository.head, do: "#{repository.head.abbreviated_hash}:" %>
            <%= if repository.head, do: repository.head.subject %>
            <%= if repository.deployments do %>
              <div :for={ {deployment_id, spec} <- repository.deployments }>
                { deployment_id }
                <div :for={ plan_id <- spec.plans }>
                  <button
                    phx-click="schedule_plan"
                    phx-value-repository={id}
                    phx-value-deployment={deployment_id}
                    phx-value-plan_id={plan_id}>
                    { plan_id }
                  </button>
                </div>
              </div>
            <% end %>
          </td>
        </tr>
      </table>
    </div>
    """
  end
end
