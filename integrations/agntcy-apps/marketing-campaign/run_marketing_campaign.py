import os
import re
import sys
import time
import json
import argparse
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

# Constants
ITERATION_SECONDS = 1
TIMEOUT_SECONDS = 300
ACP_RUNS_WAIT_ENDPOINT = f"/runs/wait"


def parse_arguments():
    parser = argparse.ArgumentParser(description="Marketing Campaign Runner")
    parser.add_argument("-w", "--working-dir", required=True, help="Working directory where to run the wfsm CLI.")
    parser.add_argument("-l", "--log-file", required=True, help="Path to the log file of the wfsm CLI.")
    return parser.parse_args()


class MarketingCampaign:
    def __init__(self, working_dir: str, log_file: str):
        self.working_dir = working_dir
        self.log_file = log_file
        self.marketing_campaign_host = ""
        self.marketing_campaign_id = ""
        self.marketing_campaign_api_key = ""

    def read_log_file(self):
        """
        Reads the specified log file every ITERATION_SECONDS searching for specific patterns
        to extract agent_id, api_key, and host URL. Sets the instance variables based on matches.

        Raises:
            Exception: If the required information is not found within the timeout.
        """
        start_time = time.time()
        os.chdir(self.working_dir)

        # Check if the log file exists
        if not os.path.isfile(self.log_file):
            raise Exception(f"Log file {self.log_file} not found")

        count_servers = 0
        last_line_count = 0
        ansi_escape = re.compile(r'\x1b\[([0-9;]*[mK])')  # Remove ANSI escape codes

        while time.time() - start_time < TIMEOUT_SECONDS:
            with open(self.log_file, 'r', encoding='utf8') as log_file:
                log_entries = log_file.readlines()

            new_lines = log_entries[last_line_count:]
            last_line_count = len(log_entries)

            for line in new_lines:
                print(line.strip()) # show wfsm logs
                line = ansi_escape.sub('', line)

                # Search for Host URL
                if not self.marketing_campaign_host:
                    match_1 = re.search(
                        r"listening for ACP requests? on: (https?://[^\s\n]+)",
                        line,
                    )
                    if match_1:
                        acp_url = match_1.group(1).strip()
                        parsed_url = urlparse(acp_url)
                        if parsed_url.scheme and parsed_url.netloc:
                            self.marketing_campaign_host = acp_url
                            continue
                        else:
                            raise Exception("Invalid URL in log file")

                # Search for Agent ID
                if not self.marketing_campaign_id:
                    match_2 = re.search(
                        r"Agent ID:\s*([0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12})",
                        line,
                    )
                    if match_2:
                        self.marketing_campaign_id = match_2.group(1).strip()
                        continue

                # Search for API Key
                if not self.marketing_campaign_api_key:
                    match_3 = re.search(
                        r"API Key:\s*([0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12})",
                        line,
                    )
                    if match_3:
                        self.marketing_campaign_api_key = match_3.group(1).strip()

                # Check for "Uvicorn running on"
                if self.marketing_campaign_id and self.marketing_campaign_api_key and self.marketing_campaign_host:
                    start_campaign = re.search(r"Uvicorn running on", line)
                    if start_campaign:
                        count_servers += 1
                        if count_servers == 3:
                            logger.info("Workflow Server started successfully.")
                            return

            time.sleep(ITERATION_SECONDS)

        raise Exception("Timeout reached: Workflow server failed to start.")


    def send_acp_runs_wait_request(self, payload: dict) -> dict:
        """
        Sends a request to the ACP runs/wait endpoint with the specified headers and payload.
        The payload includes the agent_id, input, metadata, and config.
        """
        acp_runs_wait_url = f"{self.marketing_campaign_host}/{ACP_RUNS_WAIT_ENDPOINT}"

        # Define the URL and headers
        headers = {
            'Content-Type': 'application/json',
            'Accept': 'application/json',
            'x-api-key': self.marketing_campaign_api_key
        }

        response = requests.post(acp_runs_wait_url, headers=headers, json=payload)
        if response.status_code != 200:
            raise Exception(f"Request to {acp_runs_wait_url} failed: {response.status_code} {response.text}")
        logger.debug(f"Request to {acp_runs_wait_url} successful: {response.status_code} - {response.text}")
        return response.json() 


    def run_echo_server(self):
        """
        Runs the echo server Docker container.
        Stops and removes any existing echo-server container before starting a new one.
        """
        try:
            logger.debug("Stopping and removing any existing echo-server container...")
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


    def check_echo_server_logs(self):
        """
        Executes `docker logs echo-server`, parses the output, and checks if it contains "originalUrl" with the value "/sendgrid/".

        Raises:
            Exception: If the required condition is not met.
        """
        try:
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
                        log_entry = json.loads(line)
                        logger.debug(f"Sendgrid echo server response: {json.dumps(log_entry, indent=2)}")
                        if log_entry.get("http", {}).get("originalUrl") == "/sendgrid/":
                            logger.info("Found matching log entry with 'originalUrl': '/sendgrid/'")
                            return True
                    except json.JSONDecodeError:
                        logger.warning(f"Skipping non-JSON line: {line}")
                        continue

            raise Exception("No matching log entry found in echo server logs, sendgrid call not received.")
        except subprocess.CalledProcessError as e:
            raise Exception(f"Failed to execute docker logs: {e}")


    def test_composer(self):
        """
        Sends a request to the email composer to generate an email and logs the request and response.
        """
        payload = {
            "agent_id": self.marketing_campaign_id,
            "input": {
                "messages": [
                    {
                        "content": "For a new eco-friendly water bottle. Highlight its sustainability, durability, and modern design. Do not ask further questions.",
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

        json_response = self.send_acp_runs_wait_request(payload)
        request_prompt = json_response['output']['values']['mailcomposer_state']['input']['messages'][0]['content']
        composed_email = json_response['output']['values']['mailcomposer_state']['output']['messages'][1]['content']

        logger.info("========    Request Composition:    ========\n" + request_prompt)
        logger.info("========    Composed Email:    ========\n" + composed_email)


    def test_reviewer(self):
        """
        Sends a pre-composed synthetic email to the reviewer and logs the original and revised emails.
        """
        payload = {
            "agent_id": self.marketing_campaign_id,
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

        json_response = self.send_acp_runs_wait_request(payload)
        composed_email = json_response['output']['values']['email_reviewer_state']['input']['email']
        reviewed_email = json_response['output']['values']['email_reviewer_state']['output']['corrected_email']
        sendgrid_query = json_response['output']['values']['sendgrid_state']['input']['query']
        logger.info("========    Composed Email:    ========\n" + composed_email)
        logger.info("========    Reviewed Email:    ========\n" + reviewed_email)
        logger.info("========    Sendgrid Query:    ========\n" + sendgrid_query)


def main():
    args = parse_arguments()
    working_dir: str = args.working_dir
    try:
        # If the path is relative, construct the absolute path
        if not os.path.isabs(working_dir):
            script_dir = os.path.dirname(os.path.abspath(__file__))
            if not working_dir.startswith("./") and not working_dir.startswith("../"):
                working_dir = f"./{working_dir}"
            working_dir = os.path.abspath(os.path.join(script_dir, working_dir))
        mc = MarketingCampaign(working_dir, args.log_file)
        mc.read_log_file()
        if not all([mc.marketing_campaign_id, mc.marketing_campaign_api_key, mc.marketing_campaign_host]):
                logger.error("Missing wsfm required information. Please check the log file.")
                sys.exit(1)
        mc.run_echo_server()
        logger.info("Testing email composer")
        mc.test_composer()
        logger.info("Testing email reviewer")
        mc.test_reviewer()
        logger.info("Checking echo server logs for sendgrid call...")
        mc.check_echo_server_logs()
        logger.info("Sendgrid call received successfully.")
    except Exception as e:
        logger.error(f"An error occurred: {e}")
        logger.error("Exiting with error.")
        sys.exit(1)


if __name__ == "__main__":
    main()
