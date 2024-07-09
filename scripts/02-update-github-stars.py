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

    parts = repo_url.rstrip('/').split('/')
    logging.debug(f"Parsed URL parts: {parts}")
    if 'tree' in parts:
        tree_index = parts.index('tree')
        parts = parts[:tree_index]
        logging.debug(f"URL parts after removing 'tree': {parts}")
    
    if len(parts) < 2:
        logging.warning(f"Invalid GitHub URL format: {repo_url}")
        return None
    
    owner, repo = parts[-2:]
    logging.debug(f"Owner: {owner}, Repo: {repo}")
    
    logging.info(f"Fetching stars for {owner}/{repo}")
    try:
        repo = g.get_repo(f"{owner}/{repo}")
        stars = repo.stargazers_count
        logging.info(f"Successfully fetched stars for {owner}/{repo}: {stars} stars")
        return stars
    except Exception as e:
        logging.error(f"Error fetching stars for {repo_url}: {e}")
        if e.status == 404:
            logging.error(f"Repository not found: {repo_url}")
            return "badge not found"
        return None

def should_update_stars(stars_last_updated):
    if stars_last_updated is None:
        logging.debug("No previous update timestamp, setting initial timestamp")
        return True
    last_updated_date = datetime.fromisoformat(stars_last_updated)
    time_since_update = datetime.now() - last_updated_date
    should_update = time_since_update > timedelta(days=7)
    logging.debug(f"Last updated: {last_updated_date}, Time since update: {time_since_update}, Should update: {should_update}")
    return should_update

def update_project_stars(project, json_data):
    updated = False
    project_name = project.get('project', 'Unknown')
    project_url = None
    
    if project.get('project_is_open_source'):
        for source in project.get('sources', []):
            if source['source'] == 'github':
                project_url = source['source_url']
                stars_last_updated = source.get('stars_last_updated')
                logging.debug(f"Checking update for {project_name} (URL: {project_url}), Last updated: {stars_last_updated}")
                if should_update_stars(stars_last_updated):
                    if stars_last_updated is None:
                        logging.info(f"Setting initial timestamp for {project_name}")
                        source['stars_last_updated'] = datetime.now().isoformat()
                        updated = True
                    else:
                        stars = get_github_stars(project_url)
                        if stars is not None:
                            if stars == "badge not found":
                                source['badge'] = "badge not found"
                            else:
                                source['stars'] = stars
                                source['stars_last_updated'] = datetime.now().isoformat()
                            updated = True
                else:
                    logging.info(f"Skipping update for {project_url} - updated less than a week ago")
    return updated, project_name, project_url

def update_json_with_stars(json_file_path):
    with open(json_file_path, 'r') as file:
        json_data = json.load(file)

    for project in json_data['agents']:
        logging.debug(f"Processing project: {project.get('project', 'Unknown')}")
        for source in project.get('sources', []):
            if source['source'] == 'github':
                logging.debug(f"GitHub source found: {source['source_url']}")
                logging.debug(f"Stars last updated: {source.get('stars_last_updated')}")
        
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
logging.getLogger().setLevel(logging.DEBUG)

update_json_with_stars('awesome-agents.json')