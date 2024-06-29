import json
from collections import defaultdict
from typing import List, Dict
from pydantic import BaseModel, Field, ValidationError, conlist
import hashlib
import logging
import re

logging.basicConfig(level=logging.INFO)

class Source(BaseModel):
    source: str
    source_url: str
    stars: int = None
    stars_last_updated: str = None

class Project(BaseModel):
    project: str
    project_description: str
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

def get_category_color(category: str) -> str:
    hash_value = hashlib.md5(category.encode()).hexdigest()
    return hash_value[:6]

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
        return f'<img src="https://img.shields.io/github/stars/{owner}/{repo}?style=social" alt="GitHub stars">'
    except Exception as e:
        logging.error(f"Error generating GitHub stars badge: {str(e)}")
        return ""

def format_project(project: Project) -> str:
    badge_url = next((source.source_url for source in project.sources if source.source == "github"), next((source.source_url for source in project.sources), ""))
    open_source_badge = f'<a href="{badge_url}"><img src="https://img.shields.io/badge/Open%20Source-{"Yes" if project.project_is_open_source else "No"}-{"green" if project.project_is_open_source else "red"}" alt="Open Source"></a>'
    
    github_stars_badge = next((get_github_stars_badge(source) for source in project.sources if source.source == "github"), "")
    
    badges = f"{open_source_badge} {github_stars_badge}".strip()
    categories = " | ".join([f'`{category}`' for category in project.categories])
    
    full_sources = format_sources(project.sources)
    
    return f"""### {project.project}
<div>{badges}</div>
<div>{categories}</div>

<p>{project.project_description}</p>

<p>{full_sources}</p>
</div>
"""

def generate_sections(data: JsonData) -> str:
    projects = sorted(data.agents, key=lambda x: x.project.lower())
    return '\n'.join(map(format_project, projects))

def load_template(template_file: str) -> str:
    with open(template_file, 'r') as file:
        return file.read()

def generate_readme_content(json_file: str, template_file: str) -> str:
    data = load_json(json_file)
    template = load_template(template_file)
    sections = generate_sections(data)
    return template.replace("${SECTIONS}", sections)

def write_output(output_file: str, content: str) -> None:
    with open(output_file, 'w') as file:
        file.write(content)

def main():
    json_file = "awesome-agents.json"
    template_file = "README.template.md"
    output_file = "README.md"

    try:
        data = load_json(json_file)
        logging.info(f"Loaded {len(data.agents)} projects from {json_file}")
        
        template = load_template(template_file)
        logging.info(f"Loaded template from {template_file}")
        
        sections = generate_sections(data)
        logging.info(f"Generated {len(sections.split('<div>'))} project sections")
        
        readme_content = template.replace("${SECTIONS}", sections)
        write_output(output_file, readme_content)
        logging.info(f"Successfully generated {output_file}")
    except Exception as e:
        logging.error(f"An error occurred: {str(e)}", exc_info=True)

if __name__ == "__main__":
    main()
