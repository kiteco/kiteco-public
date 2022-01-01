# JupyterHub + Kite

[Zero to JupyterHub with Kubernetes](https://zero-to-jupyterhub.readthedocs.io/en/latest/) provides a tutorial on setting up JupyterHub with Kubernetes from scratch, as well as various configuration options.

Currently, our cluster is hosted on Google Cloud, and the Helm chart is configured to use [JupyterHub-Kite](https://hub.docker.com/r/kiteco/jupyterhub-kite).

Our dev cluster can be accessed at http://jupyterhub-dev.kite.com/.

## Helm chart

[Helm](https://zero-to-jupyterhub.readthedocs.io/en/latest/reference/glossary.html?highlight=config.yaml#term-helm-values) is used to manage the Kubernetes cluster.

Our overrides to the base Helm chart is in `config.yaml`.

### Deplying Updates

To deploy updates, such as new versions of the Kite Docker image:

- Update `config.yaml` with the desired changes
- Connect to the cluster:
  ```sh
  gcloud container clusters get-credentials <insert-cluster-name-here> --zone us-west1-b --project kite-dev-XXXXXXX
  ```
- Run:

  ```sh
  RELEASE=jhub
  NAMESPACE=jhub

  helm upgrade --install $RELEASE jupyterhub/jupyterhub \
  --namespace $NAMESPACE  \
  --values config.yaml
  ```

- If you get:
  ```
  Error: failed to download "jupyterhub/jupyterhub" (hint: running `helm repo update` may help)
  ```
  Run:
  ```
  helm repo add jupyterhub https://jupyterhub.github.io/helm-chart/
  helm repo update
  ```

## Useful commands

- `kubectl config set-context --current --namespace=<insert-namespace-name-here>` changes the namespace for all subsequent `kubectl` commands in that context. Otherwise, the namespace will need to be set for each request.
- `kubectl get pods` returns a list of pods running in the container
- `kubectl get services` returns the external IP which can be used to access JupyterHub.
- `kubectl logs <insert-pod-name-here>` shows the logs for the given pod
- `kubectl exec -ti <insert-pod-name-here> -- bash` logs into the pod
