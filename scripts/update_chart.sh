#!/bin/bash

if ! command -v jq&>/dev/null; then
  echo "jq is not installed!"
  exit 1
fi

latest_semver=$(git -c 'versionsort.suffix=-rc' tag --list --sort=version:refname | grep -Eo '^v[0-9]{1,}.[0-9]{1,}.[0-9]{1,}$' | tail -n 2 | tail -1)
export version=${latest_semver#v}

yq eval -i '.controllerManager.manager.image.tag = env(version)' "charts/armada-operator/values.yaml"
yq eval -i '.version = env(version)' "charts/armada-operator/Chart.yaml"
yq eval -i '.appVersion = env(version)' "charts/armada-operator/Chart.yaml"
