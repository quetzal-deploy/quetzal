defmodule Quetzal.Git.RepoManager do
  use GenServer

  # TODO:
  # - git fetch on each repo on a timer
  # - webhook that triggers git fetch

  def start_link(_opts \\ []) do
    GenServer.start_link(__MODULE__, %{}, name: __MODULE__)
  end

  def init(_state) do
    IO.puts("repo manager: hello")

    # state_dir = System.tmp_dir!()

    {:ok, state_dir} = Briefly.create(type: :directory)
    repo_dir = Path.join([state_dir, "repositories"])
    File.mkdir(repo_dir)

    GenServer.cast(__MODULE__, :clone_repositories)

    {
      :ok,
      %{
        state_dir: state_dir,
        repositories: %{
          "deployments" => %{
            origin: "https://gitlab.dbc.dk/platform/deployments.git",
            path: Path.join(repo_dir, "deployments"), # create this automatically based on repository name/key
            ready: false,
            head: nil,
            deployments: nil,
          },
          "deployments2" => %{
            origin: "https://gitlab.dbc.dk/platform/deployments.git",
            path: Path.join(repo_dir, "deployments2"), # create this automatically based on repository name/key
            ready: false,
            head: nil,
            deployments: nil,
          },
        }
      }
    }
  end

  def get_repositories do
    GenServer.call(__MODULE__, :get_repositories)
  end

  def get_repository(repository) do
    GenServer.call(__MODULE__, {:get_repository, repository})
  end

  def handle_call(:get_repositories, _from, state) do
    # repositories = get_in(state, [:repositories])
    # {:reply, repositories, state}

    repositories = get_in(state, [:repositories])
    |> Map.new( fn {id, repo} ->
      # remove :path from what is returned
      {id, Map.delete(repo, :path)}
    end)
    {:reply, repositories, state}

  end

  def handle_call({:get_repository, repository}, _from_, state) do
    {:reply, get_in(state, [:repositories, repository]), state}
  end

  def handle_cast(:clone_repositories, state) do
    state.repositories
    |> Map.keys()
    |> Enum.each(fn repository_id ->
      # spawn(fn -> GenServer.cast(__MODULE__, {:clone_repository, repository_id}) end)
      GenServer.cast(__MODULE__, {:clone_repository, repository_id})
      # GenServer.cast(__MODULE__, {:clone_repository, repository_id})
    end)

    # deployments = get_in(state, [:repositories, "deployments"])

    # clone_repository(deployments.origin, deployments.path)
    {:noreply, state}
  end

  def handle_cast({:clone_repository, repository_id}, state) do
    spawn(fn ->
      # TODO: pattern match on this
      repository = get_in(state, [:repositories, repository_id])

      {:ok, :done} = Git.clone(repository.origin, directory: repository.path, branch: "quetzal")
      # TODO: Only do this on success:
      GenServer.cast(__MODULE__, {:refresh_repository, repository_id})
    end)

    {:noreply, state}
  end

  def scan_deployments(root_path) do
    path = Path.join(root_path, "quetzal.json")
    case File.read(path) do

      {:error, :enoent} -> IO.puts("not found: #{path}" )

      {:ok, f} ->
        File.read(Path.join(path, "quetzal.json"))
        IO.puts(f)
        {:ok, x} = Jason.decode(f)
        {:ok, Map.get(x, "deployments")}

    end
  end

  def handle_cast({:refresh_repository, repository_id}, state) do
    spawn(fn ->
      # TODO: pattern match on this
      path = get_in(state, [:repositories, repository_id, :path])

      {:ok, repo} = Git.Repo.open(path)
      # {:ok, branches} = Git.Repo.branch(repo, all: true)
      {:ok, [head]} = Git.Repo.log(repo, max_count: 1)

      {:ok, deployments} = scan_deployments(path)

      morph_evaluator = "/home/adtu/src/quetzal-evaluators/quetzal-morph.nix"

      Enum.each(deployments, fn {deployment, spec} ->
        deployment_path = Path.join(path, Map.get(spec, "path"))
        IO.puts(deployment_path)

        # {:ok, plans} = Quetzal.deployment_plans(morph_evaluator, deployment)
        Quetzal.deployment_plans(morph_evaluator, deployment_path)
        # IO.puts("deployment plans:")
        # IO.inspect(plans)
      end)

      deployments2 = Map.new(deployments, fn {deployment, spec} ->
        deployment_path = Path.join(path, Map.get(spec, "path"))
        IO.puts(deployment_path)

        # {:ok, plans} = Quetzal.deployment_plans(morph_evaluator, deployment)
        {:ok, plans} = Quetzal.deployment_plans(morph_evaluator, deployment_path)
        # IO.puts("deployment plans:")
        # IO.inspect(plans
        {deployment, Map.put(spec, :plans, plans)}
      end)

      IO.inspect(deployments2)

      # deployment = "/home/adtu/src/quetzal-rs/test/deployments/1.nix"
      # {:ok, resources} = deployment_resources(morph_evaluator, deployment)



      # TODO: Only do this on success:
      GenServer.cast(__MODULE__, {:update_repository, repository_id, head, deployments2})
    end)

    {:noreply, state}
  end

  def handle_cast({:update_repository, repository_id, head, deployments}, state) do
    new_state = state
    |> put_in([:repositories, repository_id, :ready], true)
    |> put_in([:repositories, repository_id, :head], head)
    |> put_in([:repositories, repository_id, :deployments], deployments)

    GenServer.cast(__MODULE__, :broadcast_state)

    {:noreply, new_state}
  end

  def handle_cast({:repository_ready, repository_id}, state) do
    new_state = put_in(state, [:repositories, repository_id, :ready], true)
    GenServer.cast(__MODULE__, :broadcast_state)

    {:noreply, new_state}
  end

  def handle_cast(:broadcast_state, state) do
    IO.puts("#{__MODULE__}: broadcast state")
    # TODO: refactor this out so that the filtering is not done in multiple places
    repositories = get_in(state, [:repositories])
    |> Map.new( fn {id, repo} ->
      # remove :path from what is returned
      {id, Map.delete(repo, :path)}
    end)

    QuetzalWeb.Endpoint.broadcast("repositories", "updated", repositories)
    {:noreply, state}
  end

  def get_state do
    GenServer.call(__MODULE__, :get_state)
  end

  def handle_call(:get_state, _from, state) do
    {:reply, state, state}
  end
end
