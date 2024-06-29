# 02-update-github-stars.py
# This script updates the stars count for each GitHub repository in the awesome-agents.json file

import json
import requests
from github import Github
import os
from dotenv import load_dotenv
import logging
from datetime import datetime, timedelta
from colorama import Fore, Style, init

init(autoreset=True)

logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(levelname)s - %(message)s',
    datefmt='%Y-%m-%d %H:%M:%S'
)

class ColoredFormatter(logging.Formatter):
    FORMATS = {
        logging.DEBUG: Fore.CYAN + '%(asctime)s - %(levelname)s - %(message)s' + Style.RESET_ALL,
        logging.INFO: Fore.GREEN + '%(asctime)s - %(levelname)s - %(message)s' + Style.RESET_ALL,
        logging.WARNING: Fore.YELLOW + '%(asctime)s - %(levelname)s - %(message)s' + Style.RESET_ALL,
        logging.ERROR: Fore.RED + '%(asctime)s - %(levelname)s - %(message)s' + Style.RESET_ALL,
        logging.CRITICAL: Fore.RED + Style.BRIGHT + '%(asctime)s - %(levelname)s - %(message)s' + Style.RESET_ALL
    }

    def format(self, record):
        log_fmt = self.FORMATS.get(record.levelno)
        formatter = logging.Formatter(log_fmt, datefmt='%Y-%m-%d %H:%M:%S')
        return formatter.format(record)

handler = logging.StreamHandler()
handler.setFormatter(ColoredFormatter())
logging.getLogger().handlers = [handler]

load_dotenv()

github_token = os.getenv("GITHUB_TOKEN")
g = Github(github_token)

def get_github_stars(repo_url):
    if 'github.com' not in repo_url:
        logging.warning(f"Not a GitHub URL: {repo_url}")
        return None

    # Remove 'tree/main' or similar from the URL
    parts = repo_url.rstrip('/').split('/')
    if 'tree' in parts:
        tree_index = parts.index('tree')
        parts = parts[:tree_index]
    
    # Extract owner and repo
    if len(parts) < 2:
        logging.warning(f"Invalid GitHub URL format: {repo_url}")
        return None
    
    owner, repo = parts[-2:]
    
    logging.info(f"Fetching stars for {Fore.CYAN}{owner}/{repo}{Style.RESET_ALL}")
    try:
        repo = g.get_repo(f"{owner}/{repo}")
        stars = repo.stargazers_count
        logging.info(f"Successfully fetched stars for {Fore.CYAN}{owner}/{repo}{Style.RESET_ALL}: {Fore.YELLOW}{stars}{Style.RESET_ALL} stars")
        return stars
    except Exception as e:
        logging.error(f"Error fetching stars for {Fore.CYAN}{repo_url}{Style.RESET_ALL}: {e}")
        return None

def should_update_stars(stars_last_updated):
    if not stars_last_updated:
        return True
    last_updated_date = datetime.fromisoformat(stars_last_updated)
    return datetime.now() - last_updated_date > timedelta(days=7)

def update_project_stars(project, json_data):
    updated = False
    project_name = project.get('project', 'Unknown')
    project_url = None
    
    logging.debug(f"Project structure: {json.dumps(project, indent=2)}")
    
    if project.get('project_is_open_source'):
        for source in project.get('sources', []):
            if source['source'] == 'github':
                project_url = source['source_url']
                if should_update_stars(source.get('stars_last_updated')):
                    stars = get_github_stars(project_url)
                    if stars is not None:
                        source['stars'] = stars
                        source['stars_last_updated'] = datetime.now().isoformat()
                        updated = True
                else:
                    logging.info(f"Skipping update for {Fore.CYAN}{project_url}{Style.RESET_ALL} - updated less than a week ago")
    return updated, project_name, project_url

def update_json_with_stars(json_file_path):
    with open(json_file_path, 'r') as file:
        json_data = json.load(file)

    for project in json_data['agents']:
        updated, project_name, project_url = update_project_stars(project, json_data)
        if updated:
            with open(json_file_path, 'w') as file:
                json.dump(json_data, file, indent=2)
            logging.info(f"JSON file updated for project: {Fore.CYAN}{project_name}{Style.RESET_ALL} (URL: {project_url})")
        else:
            logging.info(f"No update needed for project: {Fore.CYAN}{project_name}{Style.RESET_ALL}")

    num_projects = len(json_data['agents'])
    num_categories = len(json_data['categories'])

    logging.info(f"Number of projects: {Fore.YELLOW}{num_projects}{Style.RESET_ALL}")
    logging.info(f"Number of categories: {Fore.YELLOW}{num_categories}{Style.RESET_ALL}")

# Set debug logging level
logging.getLogger().setLevel(logging.INFO)

update_json_with_stars('awesome-agents.json')
