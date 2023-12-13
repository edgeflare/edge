import * as yaml from 'js-yaml';

export const jsonToYaml = (json: any): string => {
  try {
    // Convert a JSON object to a YAML string
    const yamlStr = yaml.dump(json, { noArrayIndent: true});
    return yamlStr;
  } catch (e) {
    console.error('Error converting JSON to YAML:', e);
    return '';
  }
}
