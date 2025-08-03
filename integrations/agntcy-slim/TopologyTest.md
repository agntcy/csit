# TopologyTest

TopologyTest is a testing framework for validating SLIM topologies. It allows you to describe complex network topologies with clients, servers, and routing configurations in a single YAML file, then quickly deploy and test these configurations in a Kubernetes environment.

## Overview

The TopologyTest framework enables you to:

- Define SLIM network topologies decoratively in YAML
- Automatically deploy clients and servers to Kubernetes
- Test message routing and communication patterns
- Validate expected behaviors through SLIM client log assertions
- Support both secure (SPIRE mTLS) and insecure connections (not yet implemented)

## Quick Start

### Steps to Run Topology Test

1. **Navigate to the test directory:**
   ```bash
   cd integrations/agntcy-slim/agentic-apps
   ```

2. **Deploy the SLIM controller:**
   ```bash
   task test-env:deploy:controller
   ```

3. **Deploy the generated topology:**
   ```bash
   task test-env:deploy:generated
   ```

4. **Run the topology test:**
   ```bash
   task test:topology
   ```

### Tear Down

1. **Clean up generated resources:**
   ```bash
   task test-env:cleanup:generated
   ```

2. **Clean up controller:**
   ```bash
   task test-env:cleanup:controller
   ```

## Topology Configuration Format

The topology is described in a YAML file with the following structure:

### Basic Structure

```yaml
topology:
    clients:
        # Client configurations
    servers:
        # Server configurations
```

### Client Configuration

Each client in the topology is defined with the following properties:

```yaml
clients:
    "client-name":
        # Optional: Authentication configuration (currently commented out)
        # auth:
        #     spireJwt: true
        
        # Required: List of servers this client connects to
        connectedTo:
            - "server-name"
        
        # Required: Docker image for the client
        image: ghcr.io/agntcy/slim/bindings-examples:latest
        
        # Required: Command line arguments for the client
        args: ["command", "--flag", "value"]
        
        # Optional: String to search for in logs to assert success
        assertFor: "Expected log message"
```

### Server Configuration

Each server in the topology is defined with:

```yaml
servers:
    "server-name":
        # Optional: Enable/disable SPIRE mTLS (default: true)
        spireMtls: false
        
        # Required: Routing configuration
        routes:
            - "channelName > destination-server"
```

## Example: Fire-and-Forget Topology

The `config/fire-and-forget.yaml` sets up 3 SLIM nodes connected with two clients using fire & forget session:

```yaml
topology:
    clients:
        "alice":
            # auth:
            #     spireJwt: true
            connectedTo:
                - "slim-1"
            image: ghcr.io/agntcy/slim/bindings-examples:latest          
            args: ["ff", "--local", "org/ns/alice", "--shared-secret","secret123"]  
            assertFor: "replies: hello from"                                        
        "bob":
            # auth:
            #     spireJwt: true
            connectedTo:
                - "slim-2"
            image: ghcr.io/agntcy/slim/bindings-examples:latest                      
            args: ["ff", "--message", "hello", "--iterations", "10", "--local", "org/ns/bob", "--remote", "org/ns/alice", "--shared-secret","secret123"]
            assertFor: "Sent message hello - 10/10"
    servers:
        "slim-0":
            spireMtls: false
            routes:                
                - "org/ns/bob > slim-2"  
                - "org/ns/alice > slim-1"              
        "slim-1":
            routes:                                
                - "org/ns/bob > slim-0"
        "slim-2":
            routes:                
                - "org/ns/alice > slim-0"
```

This topology creates a three-server network where:
- **Alice** connects to `slim-1` and waits to receive messages
- **Bob** connects to `slim-2` and sends 10 "hello" messages to Alice
- **slim-0** acts as the central router
- Messages route through the servers: Bob → slim-2 → slim-0 → slim-1 → Alice

## Test Execution

The test framework performs the following steps:

1. **Parse the topology YAML** file and generate SLIM helm chart values & configs 
2. **Deploy SLIM controller & SLIM server nodes** using helm charts with appropriate configurations
3. **Create routes using Controller API** with `slimctl`
4. **Deploy client pods** with connection parameters
5. **Watch pod logs** for assertion strings
6. **Validate** that all expected behaviors occur within timeout periods
7. **Clean up** resources after test completion