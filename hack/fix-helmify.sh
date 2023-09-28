#!/bin/bash

search_string='{{ include "{{ .Release.Namespace }}-operator.fullname" . }}'
replace_string='{{ include "armada-operator.fullname" . }}'
file_path="charts/armada-operator/templates/serving-cert.yaml"

sed -i '' -e "s|${search_string}|${replace_string}|g" "$file_path"
