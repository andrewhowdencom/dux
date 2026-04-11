# How-to: Group Tools into Toolsets

As configuration sizes grow, defining every standalone tool individually per agent becomes bloated. You can organize logical operations—such as Docker deployments, Git interactions, or Filesystem utilities—into reusable "Toolsets".

## Creating a Tool Group Header

A Toolset in `dux` is simply a standard definition in the global configuration that contains nested tools via the `tools` array.

To create a grouping, define an empty parent tool block and place your actual tools inside:

```yaml
tools:
  - name: my_infrastructure_tools
    tools:
      - name: docker_apply
        requirements:
            supervision: true
        binary:
          executable: docker-compose
          args: ["up", "-d"]
      - name: kubectl_get_pods
        binary:
          executable: kubectl
          args: ["get", "pods", "-n", "{namespace}"]
          inputs:
            namespace:
              type: string
              description: "The kubernetes namespace"
              required: true
```

## Referencing the Toolset in an Agent

Agents selectively opt-in to tools and capabilities using their Context definitions. 

Instead of naming `docker_apply` and `kubectl_get_pods` manually within `agent.yaml`, simply reference the name of your Toolset:

```yaml
name: sre_agent
provider: openai
context:
  tools:
    - name: stdlib
    - name: my_infrastructure_tools
```

During initialization, `dux` automatically recurses through the hierarchy attached to `my_infrastructure_tools` and expands its capabilities to the provider, registering both the `docker_apply` and `kubectl_get_pods` primitives to the AI securely.
