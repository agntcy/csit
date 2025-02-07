# Report

This tool allows the upload of a test report from a JSON file to a Confluence page.

## Instructions

### Environment Configuration

```plaintext
CONFLUENCE_BASE_URL=https://cisco-eti.atlassian.net
CONFLUENCE_PAT=your_api_token
CONFLUENCE_USERNAME=your_username
CONFLUENCE_TEMPLATE_ID=1030914075
CONFLUENCE_SPACE_ID=942407687
CONFLUENCE_PARENT_PAGE_ID=1029406783
```

### Sample JSON Report File

```json
{
    "test_name": "Simple Crew with CrewAI",
    "test_description": "This is a simple crew that uses CrewAI to perform a research task and a reporting task.",
    "test_confluence_parent_page_id": 123456789,
    "test_timestamp": "2025-01-02 13:52:34",
    "test_path": "/Users/user/Repository/phoenix/phoenix-csit/samples/crewai/simple",
    "test_duration": 97,
    "input_agents": {"researcher": "","reporting_analyst": ""},
    "input_tasks": {"research_task": "", "reporting_task": ""},
    "output": "test output",
    "model": "llama3.1",
    "model_provider": "Ollama",
    "token_usage": "",
    "task_duration": {"research_task": 21,"reporting_task": 52},
    "task_scores": {"research_task": 9.5,"reporting_task": 10},
    "task_scores_method": "evaluator agent"
}
```

The final report format is an HTML table with two columns where the first column corresponds to the JSON key and the second column corresponds to the JSON value.

The report is divided into four categories:

1. Test Metadata: Includes keys prefixed with test (e.g., test name, test description).
2. Input Data: Includes keys prefixed with input (e.g., agents, tasks, prompts).
3. Output Data: Includes keys prefixed with output.
4. Miscellaneous: Includes any other keys not covered by the above categories (e.g. models, token usage, evaluation).

Please use the respective prefixes (`test`, `input`, `output` or none, respectively) to link keys to categories. 

If `test_confluence_parent_page_id` is specified in the report, the report will be saved to this space instead of the space in environment configuration.

### Running the Command

To upload the test report to Confluence, run the following command:

```sh
task test:samples:upload_report --filePath="path/to/your/report.json"
```

Replace path/to/your/report.json with the path to your JSON report file.
