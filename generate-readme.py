import yaml
from collections import defaultdict
from typing import List, Dict
from pydantic import BaseModel, Field, ValidationError, conlist, field_validator

class Source(BaseModel):
    source: str
    source_url: str

class Project(BaseModel):
    project: str
    project_description: str
    categories: List[str] = Field(default_factory=list)
    sources: conlist(Source, min_length=1)

class YamlData(BaseModel):
    projects: List[Project]

def load_yaml(file_path: str) -> YamlData:
    with open(file_path, 'r') as file:
        data = yaml.safe_load(file)
    try:
        return YamlData(projects=data)
    except ValidationError as e:
        print("Validation errors found:")
        for error in e.errors():
            project_index = error['loc'][1]
            project_name = data[project_index].get('project', 'Unknown project')
            print(f"- Project {project_index} ({project_name}): {error['msg']}")
            print(f"  Details: {error}")
        raise

def format_sources(sources: List[Source]) -> str:
    return ' | '.join(f"[{source.source}]({source.source_url})" for source in sources)

def format_project(project: Project) -> str:
    return f"**{project.project}**: {project.project_description} | {format_sources(project.sources)}\n"

def load_category_descriptions(file_path: str) -> Dict[str, str]:
    with open(file_path, 'r') as file:
        data = yaml.safe_load(file)
    return {category['category']: category['category_description'] for category in data}

def generate_sections(data: YamlData, category_descriptions: Dict[str, str]) -> str:
    categories = defaultdict(list)
    for project in data.projects:
        for category in project.categories:
            categories[category].append(format_project(project))
    
    sections = []
    for category, projects in sorted(categories.items()):
        section = f"## {category}\n\n"
        description = category_descriptions.get(category, f"{category}")
        section += f"Here's an awesome list of {description}:\n\n"
        section += '\n\n'.join(f"{i}. {project.strip()}" for i, project in enumerate(projects, 1))
        section += '\n'
        sections.append(section)
    
    return '\n'.join(sections)

def load_template(template_file: str) -> str:
    with open(template_file, 'r') as file:
        return file.read()

def generate_readme_content(yaml_file: str, template_file: str, categories_file: str) -> str:
    data = load_yaml(yaml_file)
    template = load_template(template_file)
    category_descriptions = load_category_descriptions(categories_file)
    sections = generate_sections(data, category_descriptions)
    return template.replace("${SECTIONS}", sections)

def write_output(output_file: str, content: str) -> None:
    with open(output_file, 'w') as file:
        file.write(content)

def main():
    yaml_file = "awesome-agents.yaml"
    template_file = "README.template.md"
    categories_file = "awesome-categories.yaml"
    output_file = "README.md"

    try:
        readme_content = generate_readme_content(yaml_file, template_file, categories_file)
        write_output(output_file, readme_content)
        print("README.md has been generated successfully.")
    except Exception as e:
        print(f"An error occurred: {e}")

if __name__ == "__main__":
    main()
