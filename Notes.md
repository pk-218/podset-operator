## Introduction

A Kubernetes operator includes the resources, its controller, and domain specific knowledge coupled with them.
Operators take advantage of Custom Resource Definitions. 
A CRD needs a controller to act on its precense and that logic is to be written in an operator.

A good thought - Why do we need Operators? 
- To provide an 'as-a-service' platform experience. 
- To build an ecosystem on K8s that can be as easy, safe, and reliable to use and operate as a cloud service.

## Operator Lifecycle Manager (OLM)

The Operator Lifecycle Manager (OLM) extends Kubernetes to provide a declarative way to install, manage, and upgrade Operators on a cluster.

## Using the Operator SDK

1. Initialize the project by providing the domain and repository names, for example -
    ```bash
    operator-sdk init --domain github.com --repo github.com/pk-218/pod-set
    ```
    This will create the following:
    - `config` and the `hack` directories
    - `.dockerignore`
    - `.gitignore`
    - `Dockerfile`
    - `go.mod`
    - `go.sum`
    - `main.go`
    - The very important `Makefile`
    - `PROJECT`
    - `README.md`

2. `config` contains several YAML files, esp. those that are for Kustomize, which is a Kubernetes native configuration management tool. These are not to be used individually or edited.

3. `hack` contains a file `boilerplate.go.txt` which is just a license. It is the boilerplate that it introducted to every generated code file in a 'hacky' way - hence the name. Different annotations can be provided to Kubebuilder to specify the boilerplate needed for the generated files.

4. After the initialization of the project, time to create our CRD. Use the create api command of the SDK for this, for example -
    ```bash
    operator-sdk create api --group app --version  v1alpha1 --kind PodSet --resource --controller
    ```
    Here, the group name that will be attached to the new API extension of the CRD is provided along with the version, kind name that should be given for the CRD, and if required, providing the resource and controller tags to create boilerplate code for them.

5. Updated controller and types with the required logic for the custom resource.

6. First update types and run `make generate`, `make manifests`

7. Run the same commands after updating the controller.

8. Now, in order to create the final CRD for the custom resource, run `make install run` which in turn runs `make kustomize` and `make manifests`. 

9. Kustomize, the native K8s configuration manager will be installed. Issues faced -
    - Kustomize cannot be installed with the given script since certificate is not supported from usercontent.github.com
    - Had to install the tarball for Kustomize from GitHub releases using -
        ```bash
        wget --no-check-certificate  https://github.com/kubernetes-sigs/kustomize/releases/download/kustomize%2Fv4.5.5/kustomize_v4.5.5_linux_arm64.tar.gz
        sudo tar xzvf kustomize_v4.5.5_linux_arm64.tar.gz 
        ```
    - Now, instead of `make install run`, `make manifests` was executed separately and Kustomize build was executed using - `./kustomize build config/crd | kubectl apply -f -`

10. To apply the CRD created, use -
    ```bash
    kubectl apply -f config/crd/bases/app.github.com_podsets.yaml 
    ```

11. `kubectl` now knows about the new CRD and create objects of this kind. The CRD sample given in `/config/samples/` was used to created the custom resource.
    ```bash
    kubectl apply -f config/samples/app_v1alpha1_podset.yaml 
    ```

12. `kubectl get podset` or `kubectl get podsets` will give details of the custom resource. Use `kubectl describe crd podset` to get details about created CRD.
