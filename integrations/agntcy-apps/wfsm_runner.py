import os
import sys
import subprocess
import logging

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

if __name__ == "__main__":
    working_dir = os.path.dirname(os.path.abspath(__file__)) + "/agentic-apps/marketing-campaign"
    wfsm_bin_path = "../../wfsm"
    log_file = "../../wfsm.log"

    # Esegui il processo wfsm
    if not run_wfsm_binary(working_dir, wfsm_bin_path, log_file):
        logger.error("Failed to start the wfsm process.")
        sys.exit(1)