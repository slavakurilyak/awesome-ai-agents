import json
from collections import defaultdict
from typing import List, Dict, Optional, Tuple
from pydantic import BaseModel, Field, ValidationError, conlist
import hashlib
import logging
import re
import yaml
import os # Added import
from datetime import datetime, timedelta, timezone # Added import

logging.basicConfig(level=logging.INFO)

class Source(BaseModel):
    source: str
    source_url: str
    stars: int = None
    stars_last_updated: str = None

class Project(BaseModel):
    project: str
    project_description: Optional[str] = None
    project_is_open_source: bool
    categories: List[str] = Field(default_factory=list)
    sources: conlist(Source, min_length=1)

class Category(BaseModel):
    category: str
    category_description: str

class JsonData(BaseModel):
    agents: List[Project]
    categories: List[Category]

def load_json(file_path: str) -> JsonData:
    try:
        with open(file_path, 'r') as file:
            data = json.load(file)
        return JsonData(**data)
    except json.JSONDecodeError as e:
        logging.error(f"Error decoding JSON: {e}")
        raise
    except ValidationError as e:
        logging.error("Validation errors found:")
        for error in e.errors():
            project_index = error['loc'][1]
            project_name = data['agents'][project_index].get('project', 'Unknown project')
            logging.error(f"- Project {project_index} ({project_name}): {error['msg']}")
            logging.error(f"  Details: {error}")
        raise

def format_sources(sources: List[Source]) -> str:
    formatted_sources = []
    for source in sources:
        source_str = f'<a href="{source.source_url}">{source.source}</a>'
        formatted_sources.append(source_str)
    return ' | '.join(formatted_sources)

def get_badge_url(project: Project) -> str:
    github_url = next((source.source_url for source in project.sources if source.source == "github"), None)
    if github_url:
        return github_url
    return next((source.source_url for source in project.sources), "")

def get_github_stars_badge(source: Source) -> str:
    try:
        match = re.search(r'github\.com/([^/]+)/([^/]+)', source.source_url)
        if not match:
            raise ValueError(f"Invalid GitHub URL format: {source.source_url}")
        owner, repo = match.groups()
        return f'<a href="{source.source_url}"><img src="https://img.shields.io/github/stars/{owner}/{repo}?style=social" alt="GitHub stars"></a>'
    except Exception as e:
        logging.error(f"Error generating GitHub stars badge: {str(e)}")
        return ""

def load_category_emojis(file_path: str) -> Dict[str, str]:
    try:
        with open(file_path, 'r') as file:
            data = yaml.safe_load(file)
        
        if isinstance(data, list):
            return {item['category']: item['emoji'] for item in data if 'category' in item and 'emoji' in item}
        elif isinstance(data, dict) and 'category_emojis' in data:
            return data['category_emojis']
        else:
            logging.warning(f"Unexpected YAML structure in {file_path}")
            return {}
    except FileNotFoundError:
        logging.warning(f"Category emojis file not found: {file_path}")
        return {}
    except yaml.YAMLError as e:
        logging.error(f"Error parsing YAML file: {e}")
        return {}

def generate_sections(data: JsonData, category_emojis: Dict[str, str]) -> str:
    projects_dict = defaultdict(lambda: {"project": None, "categories": set()})
    
    for project in data.agents:
        key = project.project.lower()
        if not projects_dict[key]["project"]:
            projects_dict[key]["project"] = project
        projects_dict[key]["categories"].update(project.categories)
    
    sorted_projects = sorted(projects_dict.values(), key=lambda x: x["project"].project.lower())
    return '\n'.join(format_project(p["project"], list(p["categories"]), category_emojis) for p in sorted_projects)

def format_project(project: Project, all_categories: List[str], category_emojis: Dict[str, str]) -> str:
    badge_url = next((source.source_url for source in project.sources if source.source == "github"), next((source.source_url for source in project.sources), ""))
    open_source_badge = f'<a href="{badge_url}"><img src="https://img.shields.io/badge/Open%20Source-{"Yes" if project.project_is_open_source else "No"}-{"green" if project.project_is_open_source else "red"}" alt="Open Source"></a>'
    
    github_stars_badge = next((get_github_stars_badge(source) for source in project.sources if source.source == "github"), "")
    
    stars_info_line = ""
    github_source_with_stars = next((s for s in project.sources if s.source == "github" and s.stars is not None), None)
    if github_source_with_stars:
        stars_count_formatted = f"{github_source_with_stars.stars:,}"
        updated_date_formatted = ""
        if github_source_with_stars.stars_last_updated:
            try:
                # Parse ISO format string, handle potential 'Z' for UTC
                dt_obj = datetime.fromisoformat(github_source_with_stars.stars_last_updated.replace('Z', '+00:00'))
                updated_date_formatted = f"(Updated: {dt_obj.strftime('%Y-%m-%d')})"
            except ValueError:
                updated_date_formatted = f"(Updated: {github_source_with_stars.stars_last_updated[:10]})" # Fallback
        stars_info_line = f'<p>⭐ {stars_count_formatted} stars {updated_date_formatted}</p>'

    badges = f"{open_source_badge} {github_stars_badge}".strip()
    categories = " | ".join([f'{category_emojis.get(category, "")} {category}' for category in all_categories])
    
    full_sources = format_sources(project.sources)
    
    description = project.project_description or "No description provided."
    
    return f"""### {project.project}
<div>{badges}</div>
{stars_info_line}
<p>{categories}</p>

<p>{description}</p>

<p>{full_sources}</p>
</div>
"""

def generate_project_list_html(projects: List[Project], category_emojis: Dict[str, str], list_type: str) -> str:
    if not projects:
        return f"<p><em>No projects to display for {list_type}.</em></p>"
    
    items_html = []
    for i, proj_data in enumerate(projects):
        project = proj_data["project"] # Assuming proj_data is a dict with "project" key
        
        # Find GitHub source for URL and stars
        github_source = next((s for s in project.sources if s.source == "github"), None)
        project_url = github_source.source_url if github_source else project.sources[0].source_url # Fallback to first source
        
        stars_display = ""
        if github_source and github_source.stars is not None:
            stars_display = f" - {github_source.stars:,} stars"
            if github_source.stars_last_updated:
                try:
                    dt_obj = datetime.fromisoformat(github_source.stars_last_updated.replace('Z', '+00:00'))
                    stars_display += f" (Updated: {dt_obj.strftime('%Y-%m-%d')})"
                except ValueError:
                     stars_display += f" (Updated: {github_source.stars_last_updated[:10]})"


        description_html = f"<br>{project.project_description}" if project.project_description else ""
        
        items_html.append(f'<li><a href="{project_url}"><strong>{project.project}</strong></a>{stars_display}{description_html}</li>')
        
    return f"<ol>\n" + "\n".join(items_html) + "\n</ol>"

def generate_top_starred_section(data: JsonData, category_emojis: Dict[str, str], top_n: int = 10) -> Tuple[str, List[str]]:
    starred_projects = []
    for project in data.agents:
        github_source = next((s for s in project.sources if s.source == "github" and s.stars is not None), None)
        if github_source:
            starred_projects.append({"project": project, "stars": github_source.stars})
    
    sorted_starred_projects = sorted(starred_projects, key=lambda x: x["stars"], reverse=True)[:top_n]
    logging.info(f"Found {len(starred_projects)} projects with stars. Generating Top {top_n} list from {len(sorted_starred_projects)} projects.")
    
    # Extract project names for exclusion
    top_project_names = [p["project"].project for p in sorted_starred_projects]
    
    return generate_project_list_html(sorted_starred_projects, category_emojis, "Top Starred Projects"), top_project_names

def generate_rising_projects_section(data: JsonData, category_emojis: Dict[str, str], top_n: int = 10, days_recent: int = 30, exclude_projects: List[str] = None) -> str:
    recent_projects = []
    thirty_days_ago = datetime.now(timezone.utc) - timedelta(days=days_recent)
    exclude_set = set(exclude_projects) if exclude_projects else set()
    
    for project in data.agents:
        # Skip if project is in the exclude list
        if project.project in exclude_set:
            continue
            
        github_source = next((s for s in project.sources if s.source == "github" and s.stars is not None and s.stars_last_updated is not None), None)
        if github_source:
            try:
                updated_date_str = github_source.stars_last_updated
                # Ensure timezone awareness: if 'Z' is present, it's UTC. Otherwise, assume UTC.
                if updated_date_str.endswith('Z'):
                    updated_date = datetime.fromisoformat(updated_date_str.replace('Z', '+00:00'))
                else:
                    # Attempt to parse, if it's naive, assume UTC
                    dt_naive = datetime.fromisoformat(updated_date_str)
                    if dt_naive.tzinfo is None or dt_naive.tzinfo.utcoffset(dt_naive) is None:
                         updated_date = dt_naive.replace(tzinfo=timezone.utc)
                    else:
                        updated_date = dt_naive # Already timezone aware

                if updated_date >= thirty_days_ago:
                    recent_projects.append({"project": project, "stars": github_source.stars, "updated_date": updated_date})
            except ValueError as e:
                logging.warning(f"Could not parse date '{github_source.stars_last_updated}' for project {project.project}: {e}")
                continue # Skip if date is unparseable

    sorted_recent_projects = sorted(recent_projects, key=lambda x: x["stars"], reverse=True)[:top_n]
    logging.info(f"Found {len(recent_projects)} projects updated in the last {days_recent} days with stars. Generating Rising {top_n} list from {len(sorted_recent_projects)} projects.")
    return generate_project_list_html(sorted_recent_projects, category_emojis, "Rising Projects")

def load_template(template_file: str) -> str:
    with open(template_file, 'r') as file:
        return file.read()

def generate_readme_content(json_file: str, template_file: str, category_emojis_file: str) -> str:
    data = load_json(json_file)
    category_emojis = load_category_emojis(category_emojis_file)
    template = load_template(template_file)
    
    main_sections_content = generate_sections(data, category_emojis)
    top_starred_content, top_project_names = generate_top_starred_section(data, category_emojis)
    rising_projects_content = generate_rising_projects_section(data, category_emojis, exclude_projects=top_project_names)
    
    content = template.replace("${SECTIONS}", main_sections_content)
    content = content.replace("${TOP_STARRED_PROJECTS}", top_starred_content)
    content = content.replace("${RISING_PROJECTS}", rising_projects_content)
    
    return content

def write_output(output_file: str, content: str) -> None:
    with open(output_file, 'w') as file:
        file.write(content)

def main():
    # Determine paths relative to this script's location
    SCRIPT_DIR = os.path.dirname(os.path.abspath(__file__))
    PROJECT_ROOT = os.path.dirname(SCRIPT_DIR) # This is the 'awesome-ai-agents' directory

    json_file = os.path.join(PROJECT_ROOT, "awesome-agents.json")
    template_file = os.path.join(PROJECT_ROOT, "README.template.md")
    output_file = os.path.join(PROJECT_ROOT, "README.md") # Corrected output path
    category_emojis_file = os.path.join(PROJECT_ROOT, "awesome-categories.yaml")

    try:
        category_emojis = load_category_emojis(category_emojis_file)
        if category_emojis:
            logging.info(f"Loaded {len(category_emojis)} category emojis from {category_emojis_file}")
        else:
            logging.warning("No category emojis loaded. Proceeding without emojis.")

        data = load_json(json_file)
        logging.info(f"Loaded {len(data.agents)} projects from {json_file}")
        
        # Template loading and main sections generation are now part of generate_readme_content
        # Logging for those will happen within or after that call if needed.
        
        readme_content = generate_readme_content(json_file, template_file, category_emojis_file)
        # Logging for section generation (top, rising, main) is now inside their respective functions
        # or implicitly covered by the final "Successfully generated" log.
        
        write_output(output_file, readme_content)
        logging.info(f"Successfully generated {output_file}")
    except Exception as e:
        logging.error(f"An error occurred: {str(e)}", exc_info=True)

if __name__ == "__main__":
    main()
