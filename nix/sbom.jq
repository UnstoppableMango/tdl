# Merges cyclonedx-gomod's SBOM of each binary on each target into one, without
# the per-binary hashes and per-platform purl qualifiers.
(.[0].metadata.component | .["bom-ref"] |= sub("#.*$"; "")
  | { "bom-ref", type, name, version, purl: .["bom-ref"], externalReferences }) as $subject
| (map(.components // []) | add | map(.purl = .["bom-ref"]) | unique_by(.["bom-ref"])) as $modules
| .[0]
| del(.metadata.properties)
| .metadata.component = $subject
| .components = $modules
| .dependencies = [{ ref: $subject["bom-ref"], dependsOn: ($modules | map(.["bom-ref"])) }]
| .compositions = [
    { aggregate: "complete", dependencies: [$subject["bom-ref"]] },
    { aggregate: "unknown", dependencies: ($modules | map(.["bom-ref"])) }
  ]
