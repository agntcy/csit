# Configuration Information

In our ConfigMap, we define a series of variables that are passed to pods as environment (ENV) variables. 
However, the variables for the message queue and control plane are
not explicitly used in the code (`deploy_control_plane.py` and `deploy_msg_queue.py`).
These variables are actually set through the `llama-deploy` library instead. Similarly, the workflow 
service can also be configured using ENV variables if there is only one service running.
You can find the relevant code here:
 - [control plane](https://github.com/run-llama/llama_deploy/blob/main/llama_deploy/control_plane/config.py#L10)
 - [simple message queue](https://github.com/run-llama/llama_deploy/blob/main/llama_deploy/message_queues/simple.py#L36)
 - [workflow service](https://github.com/run-llama/llama_deploy/blob/main/llama_deploy/services/workflow.py#L44)