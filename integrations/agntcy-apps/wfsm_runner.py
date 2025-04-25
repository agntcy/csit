import os
import sys
import subprocess
import logging
import yaml

# Configure logging
logging.basicConfig(
    level=logging.DEBUG,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)

def run_wfsm_binary(working_dir: str, wfsm_bin_path: str, log_file: str) -> bool:
    """
    Starts the wfsm process in the background, writes logs to the file, and continues execution.

    Args:
        working_dir (str): The working directory where the wfsm binary is located.
        wfsm_bin_path (str): Path to the wfsm binary.
        log_file (str): Path to the log file.

    Returns:
        bool: True if the process was started successfully, False otherwise.
    """
    try:
        # Change the working directory
        os.chdir(working_dir)

        command = [
            wfsm_bin_path,
            "deploy",
            "-m", "./deploy/marketing-campaign.json",
            "-e", "../../marketing_campaign_cfg_yaml.env"
        ]

        logger.info(f"Starting the wfsm process: {' '.join(command)}")
        with open(log_file, "w") as log:
            # Process in background
            process = subprocess.Popen(
                command,
                stdout=log,
                stderr=subprocess.STDOUT,
                preexec_fn=os.setpgrp  # Disassociate from parent process
            )

        logger.info(f"wfsm process started with PID: {process.pid}")
        return True

    except Exception as e:
        logger.error(f"Error while starting the wfsm process: {e}")
        return False

def flatten_keys(data, parent_key=""):
    """
    Recursively flattens nested dictionary keys.

    Args:
        data (dict): The dictionary to flatten.
        parent_key (str): The base key for recursion.

    Returns:
        list: A list of flattened keys.
    """
    keys = []
    for k, v in data.items():
        full_key = f"{parent_key}.{k}" if parent_key else k
        if isinstance(v, dict):
            keys.extend(flatten_keys(v, full_key))
        else:
            keys.append(full_key)
    return keys

def validate_env_file(working_dir: str, env_file: str, example_file: str) -> bool:
    """
    Validates that the env file contains the keys required by the marketing campaign
    and ensures no keys have empty values.

    Args:
        env_file (str): Path to the environment file.
        example_file (str): Path to the example YAML file.

    Returns:
        bool: True if the keys match and no empty values are found, False otherwise.
    """
    try:
        os.chdir(working_dir)

        # Load keys from the example YAML file
        with open(example_file, "r") as f:
            example_data = yaml.safe_load(f)
        example_keys = set(flatten_keys(example_data))

        # Load keys from the env YAML file
        with open(env_file, "r") as f:
            env_data = yaml.safe_load(f)
        env_keys = set(flatten_keys(env_data))

        # Compare keys
        missing_keys = example_keys - env_keys
        extra_keys = env_keys - example_keys

        if missing_keys:
            logger.error(f"Missing keys in {env_file}: {missing_keys}")
        if extra_keys:
            logger.error(f"Extra keys in {env_file}: {extra_keys}")

        # Check for empty values
        empty_keys = [
            key for key in flatten_keys(env_data)
            if not get_nested_value(env_data, key)
        ]
        if empty_keys:
            logger.error(f"Keys with empty values in {env_file}: {empty_keys}")

        return not missing_keys and not extra_keys and not empty_keys

    except Exception as e:
        logger.error(f"Error while validating env file: {e}")
        return False

def get_nested_value(data, key):
    """
    Retrieves the value of a nested key in a dictionary.

    Args:
        data (dict): The dictionary to search.
        key (str): The nested key, represented as a dot-separated string.

    Returns:
        Any: The value of the key, or None if the key does not exist.
    """
    keys = key.split(".")
    for k in keys:
        if isinstance(data, dict) and k in data:
            data = data[k]
        else:
            return None
    return data

if __name__ == "__main__":
    working_dir = os.path.dirname(os.path.abspath(__file__)) + "/agentic-apps/marketing-campaign"
    wfsm_bin_path = "../../wfsm"
    log_file = "../../wfsm.log"
    env_file = "../../marketing_campaign_cfg_yaml.env"
    example_file = "./deploy/marketing_campaign_example.yaml"

    # Validate the env file
    if not validate_env_file(working_dir, env_file, example_file):
        logger.error("Validation of the env file failed. Please fix the issues and try again.")
        sys.exit(1)
    logger.info("Validation of keys in the env file passed.")

    # Execute wfsm
    if not run_wfsm_binary(working_dir, wfsm_bin_path, log_file):
        logger.error("Failed to start the wfsm process.")
        sys.exit(1)