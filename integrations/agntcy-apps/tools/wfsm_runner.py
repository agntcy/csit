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
    parser = argparse.ArgumentParser(description="Run the wfsm binary and validate environment files")
    parser.add_argument("-b", "--wfsm-bin-path", required=True, help="Path to where the wfsm binary is located.")
    parser.add_argument("-l", "--log-file", required=True, help="Path to the log file that will be created.")
    parser.add_argument("-m", "--manifest", required=True, help="Path to the model file.")
    parser.add_argument("-c", "--cfg-file", required=True, help="Path to the configuration file.")
    return parser.parse_args()


def run_wfsm_binary(wfsm_bin_path: str, log_file: str, model_file: str, cfg_file: str) -> bool:
    """
    Starts the wfsm process in the background, writes logs to the file, and continues execution.

    Args:
        wfsm_bin_path (str): Path to the wfsm binary.
        log_file (str): Path to the log file.
        model_file (str): Path to the model file.
        cfg_file (str): Path to the configuration file.

    Returns:
        bool: True if the process was started successfully, False otherwise.
    """
    try:
        command = [
            wfsm_bin_path,
            "deploy",
            "-m", model_file,
            "-c", cfg_file,
            "--dryRun=false",
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


if __name__ == "__main__":
    args = parse_arguments()
    try:
        # Execute wfsm
        if not run_wfsm_binary(args.wfsm_bin_path, args.log_file, args.manifest, args.cfg_file):
            raise RuntimeError("Failed to start the wfsm process.")

    except Exception as e:
        logger.error(e)
        sys.exit(1)
