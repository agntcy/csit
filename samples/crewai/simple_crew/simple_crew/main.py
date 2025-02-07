#!/usr/bin/env python


import json
import sys
import warnings

from simple_crew.crew import Simple

warnings.filterwarnings("ignore", category=SyntaxWarning, module="pysbd")

# This main file is intended to be a way for you to run your
# crew locally, so refrain from adding unnecessary logic into this file.
# Replace with inputs you want to test with, it will automatically
# interpolate any tasks and agents information


def run():
    """
    Run the crew.
    """
    inputs = {"topic": "AI LLMs"}
    Simple().crew().kickoff(inputs=inputs)


def test():
    """
    Test the crew execution and returns the results.
    """
    inputs = {"topic": "AI LLMs"}
    try:
        Simple().crew().kickoff(inputs=inputs)

    except Exception as e:
        raise Exception(f"An error occurred while replaying the crew: {e}")


if __name__ == "__main__":
    if len(sys.argv) < 2:
        run()
    else:
        if sys.argv[1] == "test":
            test()
        else:
            run()
