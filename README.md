# Wing - Kubernetes Replica Autoscaler Operator

Wing is a Kubernetes operator that provides advanced replica autoscaling capabilities through custom resources. It enables fine-grained control over pod scaling in Kubernetes clusters, offering a more flexible alternative to the standard Horizontal Pod Autoscaler (HPA).

## Description

Wing extends Kubernetes with a custom `ReplicaAutoscaler` resource that allows you to define sophisticated autoscaling rules for your workloads. Built using the Kubernetes operator pattern and Kubebuilder framework, Wing provides:

- Custom autoscaling strategies through a plugin system
- Fine-grained control over scaling behaviors
- Native Kubernetes integration
- Extensible architecture for custom metrics and scaling logic

## Getting Started

You’ll need a Kubernetes cluster to run against. You can use [KIND](https://sigs.k8s.io/kind) to get a local cluster for testing, or run against a remote cluster.

**Note:** Your controller will automatically use the current context in your kubeconfig file (i.e. whatever cluster `kubectl cluster-info` shows).

### Prerequisites

- Kubernetes cluster v1.16+
- kubectl configured to communicate with your cluster
- Go 1.19+ (for development)

### Installation

#### Quick Start

1. Install the Custom Resource Definitions (CRDs):

```sh
kubectl apply -f config/samples/
```

2. Build and push the controller image:

```sh
make docker-build docker-push IMG=<some-registry>/wing:tag
```
	
3. Deploy the controller:

```sh
make deploy IMG=<some-registry>/wing:tag
```

### Usage

1. Create a ReplicaAutoscaler resource:
```yaml
apiVersion: wing.xscaling.dev/v1
kind: ReplicaAutoscaler
metadata:
  name: example-autoscaler
spec:
  # Add your autoscaler configuration here
```

2. Apply the resource:
```sh
kubectl apply -f your-autoscaler.yaml
```

## Development

### Building from Source

1. Clone the repository:
```sh
git clone https://github.com/xscaling/wing.git
cd wing
```

2. Install the CRDs:
```sh
make install
```

3. Run the controller locally:
```sh
make run
```

### Running Tests

```sh
make test
```

## Contributing

Contributions are welcome! Here’s how you can help:

1. Fork the repository
2. Create a feature branch
3. Commit your changes
4. Push to your branch
5. Create a Pull Request

Please ensure your code follows the project’s coding conventions and includes appropriate tests.

### Adding New Plugins

Wing supports a plugin architecture for custom autoscaling strategies. To create a new plugin:

1. Create a new package in the `plugins` directory
2. Implement the required plugin interfaces
3. Register your plugin in the plugin discovery system

## License

Copyright 2022 xScaling.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
