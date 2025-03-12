# SPDX-FileCopyrightText: Copyright (c) 2025 Cisco and/or its affiliates.
# SPDX-License-Identifier: Apache-2.0

import json
import time

import urllib3

payloads = []
response_data_array = []


def wait_for_service(url, retries=5, delay=2):
    """Wait for the service to be available by polling the health endpoint."""
    http = urllib3.PoolManager()

    for attempt in range(retries):
        try:
            response = http.request("GET", url)
            if response.status == 200:
                return True
        except urllib3.exceptions.HTTPError as e:
            print(f"HTTPError: {e}")

        print(f"Attempt {attempt + 1} failed, retrying in {delay} seconds...")
        time.sleep(delay)

    return False


def make_request(url, method="GET", payload=None):
    # Create a PoolManager instance to make requests
    http = urllib3.PoolManager()

    # Make a request to the API
    response = http.request(
        method,
        url,
        body=json.dumps(payload),
        headers={"Content-Type": "application/json"},
    )

    return response


def test_api_post_request():
    # Define the health check URL of the API
    health_url = "http://localhost:8000/healthz"

    # Wait for the service to be ready
    assert wait_for_service(health_url, retries=10), (
        "Service did not become ready in time."
    )

    # Sleep for a few seconds to ensure that the service is ready
    time.sleep(2)

    # Create a PoolManager instance to make requests
    http = urllib3.PoolManager()

    # Define the URL of the API endpoint for POST requests
    post_url = "http://localhost:8000/message"

    # Define the payload for the POST request
    payload = {
        "message": "What is my name?",
        "context": "ctx",
        "intent": "asd",
    }
    payloads.append(payload)

    # Make a POST request to the API
    response = http.request(
        "POST",
        post_url,
        body=json.dumps(payload),
        headers={"Content-Type": "application/json"},
    )

    # Assert that the status code is 404 (OK)
    # as there is no agent to handle the request for the given intent
    assert response.status == 404, (
        f"Expected status code 404, but got {response.status}: {response.data}"
    )

    # Now let's make a valid request for the intent "hr"
    payload["intent"] = "hr"
    payload["message"] = "My name is Python"
    payloads.append(payload)

    # Make a POST request to the API
    response = http.request(
        "POST",
        post_url,
        body=json.dumps(payload),
        headers={"Content-Type": "application/json"},
    )

    # Assert that the status code is 200 (OK)
    assert response.status == 200, (
        f"Expected status code 200, but got {response.status}"
    )

    # Decode the response body
    response_data = json.loads(response.data.decode("utf-8"))
    response_data_array.append(response_data)

    # Assert that the response contains the expected keys
    assert "agent_id" in response_data, "Response does not contain 'agent_id' key"

    # Save agent_id
    agent_id = response_data["agent_id"]

    # Send another request with a different context
    payload["context"] = "ctx2"
    payloads.append(payload)

    # Make a POST request to the API
    response = http.request(
        "POST",
        post_url,
        body=json.dumps(payload),
        headers={"Content-Type": "application/json"},
    )

    # Assert that the status code is 200 (OK)
    assert response.status == 200, (
        f"Expected status code 200, but got {response.status}"
    )

    # Decode the response body
    response_data = json.loads(response.data.decode("utf-8"))
    response_data_array.append(response_data)

    # Assert that the response contains the expected keys
    assert "agent_id" in response_data, "Response does not contain 'agent_id' key"

    # Assert that the agent_id is different from the previous response
    assert response_data["agent_id"] != agent_id, (
        "Agent ID should be different for different contexts"
    )

    # Optionally, release the connection back to the pool
    response.release_conn()
