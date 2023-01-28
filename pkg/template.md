# {{ .Name }}

## Overview

{{ trim .Doc}}

Imports: {{ len .Imports }}

## Index
{{- range $val := .Funcs }}
{{ funcElem $val }}
{{- end }}
{{- range $val := .Types }}
{{ typeElem $val }}
{{- end }}

## Examples
{{- if .Examples }}
{{-   range $val := .Examples }}
- {{ $val }}
{{-   end}}
{{- else }}
This section is empty.
{{- end}}

## Constants
{{- if .Consts }}
{{-   range $val := .Consts }}
- {{ $val.Doc }}
{{-   end }}
{{- else }}
This section is empty.
{{- end }}

## Variables
{{- if .Vars }}
{{-   range $val := .Vars }}
{{-     varElem $val }}
{{- end }}
{{- else }}
This section is empty.

{{ end }}

{{-   if .Funcs }}## Functions
{{-     range $val := .Funcs }}
### {{ funcHeading $val }}

{{       funcSection $val }}{{ if $val.Doc }}{{ $val.Doc }}
{{      end }}
{{-   end }}
{{- end }}
{{- if .Types }}## Types
{{-   range $val := .Types }}
### type {{ $val.Name }}
{{      typeSection $val }}
{{-     if $val.Doc }}{{ $val.Doc }}
{{-     end }}
{{-     if $val.Funcs }}
{{-       range $valFunc := $val.Funcs }}
### {{ funcHeading $valFunc }}
{{          funcSection $valFunc }}
{{-         if $valFunc.Doc }}{{ $valFunc.Doc }}
{{-         end }}
{{-       end }}
{{-     end }}
{{-     if $val.Methods }}
{{-       range $valMethods := $val.Methods }}
### {{ funcHeading $valMethods }}
{{          funcSection $valMethods }}
{{-         if $valMethods.Doc }}{{ $valMethods.Doc }}
{{-         end }}
{{-       end }}
{{-     end }}
{{-   end }}
{{- end }}
