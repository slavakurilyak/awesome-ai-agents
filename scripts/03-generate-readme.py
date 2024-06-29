import json
from collections import defaultdict
from typing import List, Dict
from pydantic import BaseModel, Field, ValidationError, conlist

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
    with open(file_path, 'r') as file:
        data = json.load(file)
    try:
        return JsonData(**data)
    except ValidationError as e:
        print("Validation errors found:")
        for error in e.errors():
            project_index = error['loc'][1]
            project_name = data['agents'][project_index].get('project', 'Unknown project')
            print(f"- Project {project_index} ({project_name}): {error['msg']}")
            print(f"  Details: {error}")
        raise

def format_sources(sources: List[Source]) -> str:
    formatted_sources = []
    for source in sources:
        source_str = f"[{source.source}]({source.source_url})"
        formatted_sources.append(source_str)
    return ' | '.join(formatted_sources)

def format_project(project: Project) -> str:
    open_source_badge = '<img src="https://img.shields.io/badge/Open%20Source-Yes-green" alt="Open Source">' if project.project_is_open_source else '<img src="https://img.shields.io/badge/Open%20Source-No-red" alt="Open Source">'
    
    github_stars_badge = ""
    for source in project.sources:
        if source.source == "github" and source.stars is not None:
            github_stars_badge = f'<img src="https://img.shields.io/github/stars/{source.source_url.split("/")[-2]}/{source.source_url.split("/")[-1]}?style=social" alt="GitHub stars">'
            break
    
    badges = f"{open_source_badge} {github_stars_badge}".strip()
    sources = format_sources(project.sources)
    categories = " ".join([f"`{category}`" for category in project.categories])
    
    return f"""<details>
<summary><b>{project.project}</b> {badges} {categories}</summary>

<p>{project.project_description}</p>

<p>{sources}</p>
</details>
"""

def generate_sections(data: JsonData) -> str:
    projects = sorted(data.agents, key=lambda x: x.project.lower())
    
    sections = []
    for project in projects:
        sections.append(format_project(project))
    
    return '\n'.join(sections)

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
        readme_content = generate_readme_content(json_file, template_file)
        write_output(output_file, readme_content)
        print("README.md has been generated successfully.")
    except Exception as e:
        print(f"An error occurred: {e}")

if __name__ == "__main__":
    main()
