# 01-generate-json.py
# This script converts the awesome-agents.yaml file to a JSON file

import yaml
import json
import logging

logging.basicConfig(level=logging.INFO, format='%(asctime)s - %(levelname)s - %(message)s')

def read_categories(categories_file_path):
    with open(categories_file_path, 'r') as categories_file:
        return yaml.safe_load(categories_file)

def yaml_to_json(yaml_file_path, json_file_path, categories):
    with open(yaml_file_path, 'r') as yaml_file:
        yaml_data = yaml.safe_load(yaml_file)
    
    output_data = {
        "agents": yaml_data,
        "categories": categories
    }
    
    with open(json_file_path, 'w') as json_file:
        json.dump(output_data, json_file, indent=2)

    num_categories = len(categories)
    num_projects = len(yaml_data)
    
    logging.info(f"Conversion complete. JSON file saved as {json_file_path}")
    logging.info(f"Number of categories: {num_categories}")
    logging.info(f"Number of projects: {num_projects}")

categories = read_categories('awesome-categories.yaml')
yaml_to_json('awesome-agents.yaml', 'awesome-agents.json', categories)
