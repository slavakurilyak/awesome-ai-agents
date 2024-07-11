# scripts/01-generate-json.py
# This script converts the awesome-agents.yaml file to a awesome-agents.json file

import yaml
import json
import logging
from deepdiff import DeepDiff
from rich import print
from rich.console import Console
from rich.prompt import Confirm
from rich.logging import RichHandler
from rich.progress import Progress, SpinnerColumn, TextColumn
from rich.table import Table
from rich.panel import Panel

console = Console()

logging.basicConfig(
    level=logging.INFO,
    format="%(message)s",
    datefmt="[%X]",
    handlers=[RichHandler(rich_tracebacks=True)]
)

log = logging.getLogger("rich")

def read_categories(categories_file_path):
    with open(categories_file_path, 'r') as categories_file:
        return yaml.safe_load(categories_file)

def yaml_to_json(yaml_file_path, json_file_path, categories):
    with console.status("[bold green]Reading existing JSON data...") as status:
        try:
            with open(json_file_path, 'r') as json_file:
                existing_data = json.load(json_file)
                existing_agents = existing_data.get('agents', [])
            log.info(f"Found {len(existing_agents)} existing agents")
        except FileNotFoundError:
            existing_agents = []
            log.info("No existing JSON file found. Starting fresh.")

    with console.status("[bold green]Reading new YAML data...") as status:
        with open(yaml_file_path, 'r') as yaml_file:
            new_agents = yaml.safe_load(yaml_file)
        log.info(f"Loaded {len(new_agents)} agents from YAML")

    log.info("Comparing new data with existing data...")
    diff = DeepDiff(existing_agents, new_agents, ignore_order=True)
    
    non_append_changes = any(key for key in diff.keys() if key != 'dictionary_item_added')
    
    if non_append_changes:
        should_proceed = Confirm.ask("Changes other than additions detected. Do you want to proceed with a full update?")
        if not should_proceed:
            log.info("Operation cancelled by user.")
            return

    with Progress(
        SpinnerColumn(),
        TextColumn("[progress.description]{task.description}"),
        transient=True,
    ) as progress:
        task = progress.add_task("[cyan]Processing agents...", total=len(new_agents))
        
        for agent in new_agents:
            existing_agent = next((a for a in existing_agents if a['project'] == agent['project']), None)
            if existing_agent:
                for key, value in agent.items():
                    if key != 'sources':
                        existing_agent[key] = value
                    else:
                        for new_source in value:
                            existing_source = next((s for s in existing_agent['sources'] if s['source'] == new_source['source']), None)
                            if existing_source:
                                stars_last_updated = existing_source.get('stars_last_updated')
                                existing_source.update(new_source)
                                if stars_last_updated:
                                    existing_source['stars_last_updated'] = stars_last_updated
                            else:
                                existing_agent['sources'].append(new_source)
            else:
                existing_agents.append(agent)
            
            for source in agent.get('sources', []):
                if source['source'] == 'github' and 'stars_last_updated' not in source:
                    source['stars_last_updated'] = None
            
            progress.update(task, advance=1)

    output_data = {
        "agents": existing_agents,
        "categories": categories
    }
    
    with console.status("[bold green]Saving JSON file...") as status:
        with open(json_file_path, 'w') as json_file:
            json.dump(output_data, json_file, indent=2)
    
    num_categories = len(categories)
    num_projects = len(existing_agents)
    num_new_projects = len([a for a in new_agents if a not in existing_agents])
    
    summary_table = Table(title="Summary")
    summary_table.add_column("Metric", style="cyan")
    summary_table.add_column("Value", style="magenta")
    summary_table.add_row("Number of categories", str(num_categories))
    summary_table.add_row("Total number of projects", str(num_projects))
    summary_table.add_row("Number of new projects added", str(num_new_projects))
    
    console.print(Panel(summary_table, title="Conversion Complete", expand=False))

categories = read_categories('awesome-categories.yaml')
yaml_to_json('awesome-agents.yaml', 'awesome-agents.json', categories)