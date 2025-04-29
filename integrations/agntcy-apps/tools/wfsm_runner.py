import os
import sys
import subprocess
import logging
import yaml
import argparse

# Configure logging
logging.basicConfig(
    level=logging.DEBUG,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)


def parse_arguments():
    """
    Command-line arguments parser.
    """
    parser = argparse.ArgumentParser(
        description="Run the wfsm binary and validate environment files",
        usage=("The --working-dir has to be the path to the directory where the agentic application is located.\n" +
            "All other paths are relative to that folder.")
    )
    parser.add_argument("-w", "--working-dir", required=True, help="Working directory where to run the wfsm CLI.")
    parser.add_argument("-b", "--wfsm-bin-path", required=True, help="Path to where the wfsm binary is located.")
    parser.add_argument("-l", "--log-file", required=True, help="Path to the log file that will be created.")
    parser.add_argument("-m", "--manifest", required=True, help="Path to the model file.")
    parser.add_argument("-e", "--env-file", required=True, help="Path to the environment file.")
    parser.add_argument("-f", "--env-file-example", required=True, help="Path to the example YAML env file.")
    return parser.parse_args()


def run_wfsm_binary(working_dir: str, wfsm_bin_path: str, log_file: str, model_file: str, env_file: str) -> bool:
    """
    Starts the wfsm process in the background, writes logs to the file, and continues execution.

    Args:
        working_dir (str): The working directory where the wfsm binary is located.
        wfsm_bin_path (str): Path to the wfsm binary.
        log_file (str): Path to the log file.
        model_file (str): Path to the model file.
        env_file (str): Path to the environment file.

    Returns:
        bool: True if the process was started successfully, False otherwise.
    """
    try:
        # Change the working directory
        os.chdir(working_dir)

        command = [
            wfsm_bin_path,
            "deploy",
            "-m", model_file,
            "-e", env_file
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

def validate_env_file(working_dir: str, env_file: str, example_env_file: str) -> bool:
    """
    Validates that the env file contains the keys required by the application and if they are
    the same as the example file. It checks for missing keys, extra keys, and empty values.

    Args:
        env_file (str): Path to the environment file.
        example_file (str): Path to the example YAML file.

    Returns:
        bool: True if the keys match and no empty values are found, False otherwise.
    """
    try:
        os.chdir(working_dir)

        # Load keys from the example YAML file
        with open(example_env_file, "r") as f:
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
    args = parse_arguments()
    working_dir = args.working_dir
    try:
        # If the path is relative, construct the absolute path
        if not os.path.isabs(working_dir):
            script_dir = os.path.dirname(os.path.abspath(__file__))
            if not working_dir.startswith("./") and not working_dir.startswith("../"):
                working_dir = f"./{working_dir}"
            working_dir = os.path.abspath(os.path.join(script_dir, working_dir))

        # Validate the env file
        if not validate_env_file(working_dir, args.env_file, args.env_file_example):
            raise RuntimeError("Validation of the env file failed. Please fix the issues and try again.")
        logger.info("Validation of env file passed.")

        # Execute wfsm
        if not run_wfsm_binary(working_dir, args.wfsm_bin_path, args.log_file, args.manifest, args.env_file):
            raise RuntimeError("Failed to start the wfsm process.")

    except Exception as e:
        logger.error(e)
        sys.exit(1)
