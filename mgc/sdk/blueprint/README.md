# Blueprint

Sometimes we want to create executors that are a combination of
existing executors, calling them conditionally, mapping parameters,
configs and results.

Blueprints serve that purpose: they are declared in YAML and describe
basic information (name, version, description) as well as the groups
and their children, that can be other groups or executors.

## Reading

The SDK may contain embedded Blueprint files if built with `-tags "embed"`.
In addition to that, it will look into the directory defined by the
environment variable `$MGC_SDK_BLUEPRINTS_DIR` or `./blueprints` if not set.

> **NOTE:**
> if using a binary with embedded files, one may still provide overrides
> by using a file `./blueprints/file-to-be-overridden.blueprint.yaml`.
> In order to add a new file, one must create the `index.blueprint.yaml`
> including that file.


## Entry Point (index.blueprint.yaml)

The entrypoint listing all the desired schema files using the format below.
This file is a JSON or YAML file with the name `index.blueprint.yaml`
(regardless if it's JSON or YAML, always use this **exact** filename).

```yaml
version: 1.0.0 # index version, must be "1.0.0"
modules:
-   description: Your Module Description
    name: module-name
    path: module-name.blueprint.yaml
    version: 1.2.3 # your module version
```

> **NOTE:**
> it's easier to generate the index using `scripts/blueprint_index_gen.py`


## Header

Each module **MUST** start with with the following header:

```yaml
blueprint: 1.0.0
name: your-name-here
url: https://your-name-here.magalu.cloud
version: 1.2.3
description: Your Description Here
```

## Groups (Children)

The document root is a group should then include children,
which in turn can include other groups or the executors.

Groups are declared by creating a `children` property with an array
of other groups or executors, example:

```yaml
children:
  - name: root-group
    description: Showcase group (root)
    children:
      - name: sub-group
        description: Showcase group (child)
        children:

          - name: executor-name-here
            # see more about executors below
```

## Executors (Actions)

The executors describes the schema of its input parameters and configs,
as well as the result schema.

Then declares the execution steps.

```yaml
name: executor-name-here
description: Your description here

## Explicitly declare parametersSchema or configsSchema:
# parametersSchema:
#   type: object
#   properties:
#     a:
#       type: integer
#       description: this is a simple integer
#       default: 123 # if required, then it becomes optional
#     id:
#       # JSON Pointer to another resource, copy specific property
#       $ref:  /path/to/executor/parametersSchema/properties/id
#
## or use the shortcuts parameters and configs,
## which creates an object schema with the following properties.
parameters: # or explicitly as parametersSchema
  a:
    type: integer
    description: this is a simple integer
    default: 123 # if required, then it becomes optional
  id:
    # JSON Pointer to another resource, copy specific property
    $ref:  /path/to/executor/parametersSchema/properties/id

configsSchema: # or configs shortcut
  # JSON Pointer to another resource, copy all configs
  $ref: /path/to/executor/configsSchema

## There are no shortcuts to declare the result, as it can be types other than 'object'
resultSchema:
  type: object
  required:
    - id
    - key_name
  properties:
    id:
      $ref: /path/to/executor/resultSchema/properties/id
    key_name:
      $ref: /path/to/executor/resultSchema/properties/key_name

## Build the executor result
## By default, it's $.last.result, but one can create whole new value.
result: | # Remember to quote or use flow scalars
  {
    "id": $.steps["some-id"].result.id,
    "key_name": $.steps["some-id"].result.key_name
  }

steps:
 - target: /path/to/executor
   # see more about steps below
```

### Targets (JSON Pointers)

Blueprints addresses groups, executors and their internal fields using
[JSON Pointers - RFC6901](https://datatracker.ietf.org/doc/html/rfc6901).

It's easy to use and looks like a path `/path/to/element`. However
one needs to be careful with special characters `/` and `~`: if the
names contains those, they must be escaped.

For `core.Grouper` nodes, it will first try to get the child with that
name, if that fails it will see if the name is
`name`, `version` or `description`.

For `core.Executor` nodes, it will handle the names `name`, `version`,
`description`, `parametersSchema`, `configsSchema`, `resultSchema`,
`links` and `related`.

The schemas (`parametersSchema`, `configsSchema` and `resultSchema`)
can be accessed as usual, ex:
`/path/to/executor/parametersSchema/properties/propNameHere`.

The `related` is a map to other executors, which can be traversed as
expected, ex:
`/path/to/executor/related/relatedName/parametersSchema/properties/propNameHere`.

The `links` is a map to linkers, which can be traversed as expected, ex:
`/path/to/executor/links/linkName/additionalConfigsSchema/properties/propNameHere`.

#### Target To Blueprint Documents

If one wants to target the current blueprint document, for instance
to reuse some definition, then use the `blueprint#` prefix, this is
particularly useful paired with the top level `components` field.

```yaml
blueprint: 1.0.0
name: your-name-here
url: https://your-name-here.magalu.cloud
# ...
components:
  parametersSchemas:
    some-name:
      type: object
      properties:
        a:
          type: integer

children:
- name: blueprint-ref-example
  # ...
  parametersSchema:
    $ref: blueprint#/components/parametersSchemas/some-name
- name: document-url-ref-example
  # ...
  parametersSchema:
    $ref: https://your-name-here.magalu.cloud#/components/parametersSchemas/some-name
```

Notes:
- `blueprint#...` is a shortcut to use the current blueprint's url;
- any other URL may be used, there must exist a **locally available**
  blueprint with the same name in `index.blueprint.yaml`. It will
  **not** download any files.
- URLs are handled as opaque strings, they are not parsed, validated
  or simplified. They must match **exactly** the declared string in
  the other file.

### Components

The top level `components` field can contain the following subfields:
- `parametersSchemas`
- `configsSchemas`
- `resultSchemas`
- `schemas`, for any other (more broad) schemas that does not fit above.
- no other fields are allowed. Be aware the exact names must be used,
  case matters.

All of them are a map of free-form strings to JSON Schemas, which can
also contain other internal references (`$ref`).

### Document Query Format (Template and JSON Path)

Blueprints are heavily based on
[JSON Path](https://goessner.net/articles/JsonPath/) to evaluate
conditions or to build values to be used.

Conditions can also be evaluated using
[Golang's text/template](https://pkg.go.dev/text/template), which
may be easier to use than JSON Path to build complex logic.

In both cases, they are given the same document. This document is
based on the result built so far, which is composed of:
- `parameters` the blueprint executor parameters.
- `configs` the blueprint executor parameters.
- `steps` a map to `step.id` and the execution status.
  See [Steps Document Structure](./README.md#steps-document-structure) below.
- `last` the last execution status, if any, otherwise it's `null`.
- `result` is present after the execution is finalized and is usually
  used by `waitTermination` checks or `links`.


### Binary (true/false) queries

In some context, such as `if`, `retryUntil` and `waitTermination` one
can pass `jsonPathQuery` or `templateQuery` and it must return a truthy
value in order to continue.

For [jsonPathQuery](https://goessner.net/articles/JsonPath/) we consider
true a boolean `true` (explicit) or non-empty objects or arrays.

For [templateQuery](https://pkg.go.dev/text/template) we consider
true the strings `finished`, `terminated` and `true`.

Anything else is handled as false.

## Steps

The executor should include a non-empty array of steps which will
be evaluated in order.

Each step has the structure:
- `id` is the key used to index the document to be queried as in `$.steps[id].result`.
  Optional, defaults to the index number.
- `if` specifies the condition to execute this step.
  Optional, defaults to run if there are no previous errors.
- `target` JSON pointer to a valid executor.
  Required and must be a valid **Executor** [target (JSON Pointer)](./README.md#targets-json-pointers).
- `parameters` object mapping the parameter name to a JSON Path to query the document.
  Optional if the target parameter properties matches the blueprint schema.
- `configs` similar to `parameters`, but deals with global configurations.
- `waitTermination` wait for the executor to finish its state transition.
  Optional, defaults to false.
- `retryUntil` may provide `maxRetries`, `interval` (duration) and one of
  `jsonPathQuery` or `templateQuery` that must evaluate to truthy value
  in order to proceed.
  Optional, by default won't retry the execution.
- `check` may provide  one of `jsonPathQuery` or `templateQuery` that
  must evaluate to truthy value in order to proceed. An optional
  `errorMessageTemplate` can be provided to format the error message,
  defaults to the original error message. The template will receive
  the full document with additional keys `error` and `error_message`,
  containing the original error. The current step is only available
  as the root field `current`, similar to `retryUntil`.

### Steps Document Structure
All queries are done on top of the full document, with the parameters,
configs, steps, last (step). The `retryUntil` queries includes `current`
step. The property `steps` is indexed by `step.id` and provides the
following structure:
- `id` the step identifier.
- `parameters` the parameters given to the step executor.
- `configs` the configuration given to the step executor.
- `result` the step execution result value, if any. Otherwise is `null`.
- `error` the step execution failure, if any. Otherwise is `null`.
- `skipped` boolean indicating if the step was executed or not.

```yaml
id: some-id # defaults to the index
if: $.last == null || $.last.error == null # default condition, can be omitted

target:  /path/to/executor # JSON Pointer to another Executor

## if parameters is not given as a mapping of property names and jsonpath to get it,
## it will be used from the parent parameters if schema matches, which is the same as:
# parameters:
#   id: $.parameters.id

## if the target executor can wait for state transitions to finish:
waitTermination: true # defaults to false

## alternatively retry the executor until some query returns true
retryUntil:
  maxRetries: 1
  interval: 10s
  jsonPathQuery: $.current.result.status == "active"
  # or templateQuery
```

Example of a step that uses the result of another one as input parameter:

```yaml
steps:
 - id: firstStep
   target: /path/to/executor
   parameters:
    # use the blueprint parameter
    p1: $.parameters.id

 - id: secondStep
   target: /path/to/another/executor
   parameters:
    # use another step result as parameter
    p2: $.steps["firstStep"].result.someProperty
```

### Confirmable Executors

Executors can prompt the user with a template message and are only
executed if the user confirms the execution. The message will
be handled using [text/template](https://pkg.go.dev/text/template)
with the [Document Query](./README.md#document-query-format-template-and-json-path).

```yaml
confirm: |
  Do you really want to delete {{ .parameters.id }}?
  This action is not recoverable!
```

### Wait Termination

Executors can declare how to wait for a state transition to finish,
this is done by re-executing the whole set of steps until it succeeds.

```yaml
waitTermination:
  maxRetries: 2
  interval: 1s
  jsonPathQuery: | # $.result is the final result to be returned to the user
    $.result.id == $.parameters.id
```

or:

```yaml
waitTermination:
  maxRetries: 2
  interval: 1s
  templateQuery: | # .result is the final result to be returned to the user
    {{ if eq .result.id .parameters.id }}
    true
    {{ end }}
```

### Default Output Flag

If you want to provide a default output flag for the Command Line Interface (CLI),
then use the `outputFlag` key with the value you'd pass to `--cli.output`.

```yaml
outputFlag: yaml
```

### Related Executors

If you want to list other related executors that may be associated or
operate together with the one being declared, pass an object. Values
must be valid **Executor**
[targets (JSON Pointers)](./README.md#targets-json-pointers).

```yaml
related:
    some-name: /path/to/executor
```

### Links

Links can ease using the return of the execution to call another
executor, automatically mapping parameters and config as described.

Parameters must be explicitly declared. Configs will be automatically
passed since they are cross-executor schemas.

Target must be a valid **Executor**
[target (JSON Pointer)](./README.md#targets-json-pointers).

```yaml
links:
  some-name:
    target: /path/to/executor
    description: describe your link here
    parameters:
      id: $.result.id # $.result is the final result to be returned to the user
    configs:
      env: $.config.env # explicitly pass env
```
