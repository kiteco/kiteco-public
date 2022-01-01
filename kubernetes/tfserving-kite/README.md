# TensorFlow Serving + Kite

## Prerequisites

You'll need the following tools in the host machine, as well as the appropriate credentials:

- Google Cloud SDK with the Kubernetes command-line tool
- Helm package manager

Ensure the Google Cloud SDK is set to the correct project (`kite-dev-XXXXXXX` for our dev project).

Ensure `kubectl` is set to the correct context (`gke_kite-dev-XXXXXXX-west1-b_jupyterhub-kite-dev` for our dev cluster) and namespace (`jhub` is the namespace for our dev cluster).

In order to pull the private Tensorflow Serving image from Docker Hub, you'll need to [generate a Secret](https://kubernetes.io/docs/tasks/configure-pod-container/pull-image-private-registry/#create-a-secret-by-providing-credentials-on-the-command-line) on your host machine. The Helm chart looks for a value called `regcred`.

## Deploy

Helm is used to manage our deployments. The files in `template` were auto-created by Helm, and its values populated via `values.yaml`.

`deployment.yaml` corresponds to a Kubernetes Deployment object, `service.yaml` corresponds to a Service object.

0. Update [`kiteco/tfserving`](https://hub.docker.com/repository/docker/kiteco/tfserving)
1. Run `helm upgrade --install tfserving-kite .`
2. If you run `kubectl get pods`, you should see:
   ```
   ...
   tfserving-kite-XXXXXXX-XXXXXXX   1/1     Running   0          49s
   ...
   ```
3. You can run `kubectl get services` to see the IP of the `tfserving-kite` service.
