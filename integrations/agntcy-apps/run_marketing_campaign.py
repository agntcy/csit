import os
import re
import sys
import time
import json
from urllib.parse import urlparse
import requests
import logging
import subprocess

# Configure logging
logging.basicConfig(
    level=logging.DEBUG,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)

ITERATION_SECONDS = 1
TIMEOUT_SECONDS = 120

marketing_campaign_host = ""
marketing_campaign_id = ""
marketing_campaign_api_key = ""

def read_log_file(working_dir: str = "./", log_file_path="./wfsm.log"):
    """
    Reads the specified log file every ITERATION_SECONDS searching for specific patterns
    to extract agent_id, api_key, and host URL. Sets the environment variables based on matches.

    Terminates after TIMEOUT_SECONDS if all matches are not found.

    Args:
        log_file_path: Path to the log file to monitor.
        timeout_seconds: Maximum wait time in seconds.

    Returns:
        bool: True if all required information is found, False otherwise.
    """
    global marketing_campaign_host, marketing_campaign_id, marketing_campaign_api_key
    start_time = time.time()

    # Change the directory to the project path
    os.chdir(working_dir)

    # Check if the log file exists
    if not os.path.isfile(log_file_path):
        logger.error(f"Log file {log_file_path} not found")
        return False

    count_servers = 0
    while time.time() - start_time < TIMEOUT_SECONDS:

        # Regular expression to remove ANSI escape codes
        ansi_escape = re.compile(r'\x1b\[([0-9;]*[mK])')

        with open(log_file_path, 'r', encoding='utf8') as log_file:
            log_entries = log_file.readlines()

        # Analyze the new lines
        for line in log_entries:
            line = ansi_escape.sub('', line)
            # Search for Agent ID

            # Search for Host URL
            if not marketing_campaign_host:
                match_1 = re.search(
                    r"listening for ACP requests? on: (https?://[^\s\n]+)",
                    line,
                )
                if match_1:
                    acp_url = match_1.group(1).strip()
                    parsed_url = urlparse(acp_url)
                    if parsed_url.scheme and parsed_url.netloc:
                        marketing_campaign_host = acp_url
                        continue
                    else:
                        logger.error("Invalid URL")
                        sys.exit(1)

            # Search for Agent ID
            if not marketing_campaign_id:
                match_2 = re.search(
                    r"Agent ID:\s*([0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12})",
                    line,
                )
                if match_2:
                    marketing_campaign_id = match_2.group(1).strip()
                    continue

            # Search for API Key
            if not marketing_campaign_api_key:
                match_3 = re.search(
                    r"API Key:\s*([0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12})",
                    line,
                )
                if match_3:
                    marketing_campaign_api_key = match_3.group(1).strip()

            # Check starting completion
            if marketing_campaign_id and marketing_campaign_api_key and marketing_campaign_host:

                start_campaign = re.search(r"Uvicorn running on", line)
                if start_campaign:
                    count_servers += 1
                    if count_servers == 3:
                        logger.info("Workflow Server started successfully.")
                        return True

        # Wait for ITERATION_SECONDS seconds before next check
        time.sleep(ITERATION_SECONDS)

    logger.error("Timeout reached: error starting the workflow server.")
    with open(log_file_path, 'r', encoding='utf8') as log_file:
        logger.error("Timeout reached: error starting the workflow server.\n" + log_file.read())
    return False


def send_acp_runs_wait_request() -> requests.Response:
    """
    Sends a request to the ACP runs/wait endpoint with the specified headers and payload.
    The payload includes the agent_id, input, metadata, and config.
    """

    global marketing_campaign_host, marketing_campaign_id, marketing_campaign_api_key
    if not all([marketing_campaign_id, marketing_campaign_api_key, marketing_campaign_host]):
        logger.error("Missing required information. Please check the log file.")
        sys.exit(1)

    acp_runs_wait_url = f"{marketing_campaign_host}/runs/wait"

    # Define the URL and headers
    headers = {
        'Content-Type': 'application/json',
        'Accept': 'application/json',
        'x-api-key': marketing_campaign_api_key
    }

    # Define the payload
    payload = {
        "agent_id": marketing_campaign_id,
        "input": {
            "messages": [
                {
                    "type": "ai",
                    "content": "**************\n\nSubject: OK\n\nDear [Client'\''s Name],\nYes.\nRegards,\n[Your Full Name]\n**************"
                },
                {
                    "content": "OK",
                    "type": "human"
                }
            ]
        },
        "metadata": {},
        "config": {
            "configurable": {
                "recipient_email_address": "Name Surname <namesurname@example.com>",
                "sender_email_address": "agntcy@demo.com",
                "target_audience": "academic"
            }
        }
    }

    return requests.post(acp_runs_wait_url, headers=headers, json=payload)


def run_echo_server():
    """
    Runs the echo server Docker container.
    Stops and removes any existing echo-server container before starting a new one.
    """
    try:
        logger.info("Stopping and removing any existing echo-server container...")
        subprocess.run(["docker", "rm", "-f", "echo-server"], check=False)

        logger.info("Starting ealen/echo-server Docker container on localhost:8080...")
        subprocess.run([
            "docker", "run", "--rm", "-d", "-p", "8080:80",
            "--network", "orgagntcymarketing-campaign_default",
            "--name", "echo-server", "ealen/echo-server"
        ], check=True)

        logger.info("Echo server is running at http://localhost:8080")
    except Exception as e:
        logger.error(f"Failed to run echo server: {e}")
        raise

def check_echo_server_logs():
    """
    Executes `docker logs echo-server`, parses the output, and checks if contains "originalUrl" with the value "/sendgrid/".

    Returns:
        bool: True if the condition is met, False otherwise.
    """
    try:
        # Execute `docker logs echo-server` and capture the output
        result = subprocess.run(
            ["docker", "logs", "echo-server"],
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            text=True,
            check=True
        )

        logs = result.stdout.splitlines()
        found_listening = False

        for line in logs:
            if "Listening on port 80" in line:
                found_listening = True
                continue

            if found_listening:
                try:
                    #  Parse the line as JSON
                    log_entry = json.loads(line)
                    logger.debug(f"Parsed JSON: {json.dumps(log_entry, indent=2)}")

                    # Check if the key "originalUrl" exists and has the value "/sendgrid/"
                    if log_entry.get("http", {}).get("originalUrl") == "/sendgrid/":
                        logger.info("Found matching log entry with 'originalUrl': '/sendgrid/'")
                        return True
                except json.JSONDecodeError:
                    logger.warning(f"Skipping non-JSON line: {line}")
                    continue

        logger.info("No matching log entry found.")
        return False

    except subprocess.CalledProcessError as e:
        logger.error(f"Failed to execute docker logs: {e}")
        return False


if __name__ == "__main__":
    # Example usage
    working_dir = os.path.dirname(os.path.abspath(__file__))

    if not read_log_file(working_dir):
        logger.error("Failed to read wfsm logs.")
        sys.exit(1)
    run_echo_server()
    response = send_acp_runs_wait_request()
    if not check_echo_server_logs():
        logger.error("Sendgrid call failed.")
        sys.exit(1)
    logger.debug(response.status_code)
    logger.debug(response.text)
    # sys.exit(1)