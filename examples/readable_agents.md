# AI Agents Service Documentation

Welcome to our AI Agents service documentation. This service hosts several AI agents, each providing specific capabilities through well-documented endpoints. Below you'll find detailed information about each agent, their capabilities, and how to interact with them.

## python-code-agent

**Description:** Advanced Python code generation, testing, and deployment agent

**Base URL:** `http://localhost:9200`

## Capabilities

The following sections detail the specific capabilities of this agent:

### Development

| Property      | Value    |
|:--------------|:---------|
| **expertise** | advanced |

### Development → Backend → Python

| Property       | Value           |
|:---------------|:----------------|
| **expertise**  | advanced        |
| **versions**   | 3.8, 3.9, 3.10  |
| **frameworks** | Django, FastAPI |

### Development → Backend → Python → CodeGeneration

| Property       | Value           |
|:---------------|:----------------|
| **versions**   | 3.8, 3.9, 3.10  |
| **frameworks** | Django, FastAPI |
| **tools**      | black, pylint   |

### Development → Testing → Python

| Property       | Value            |
|:---------------|:-----------------|
| **expertise**  | advanced         |
| **frameworks** | pytest, unittest |

### Development → Deployment → Python

| Property      | Value           |
|:--------------|:----------------|
| **expertise** | basic           |
| **platforms** | AWS, GCP, Azure |

## Available Endpoints

This section describes all available endpoints for interacting with the agent:

### deployPreview

**Endpoint:** `POST /v1/code_agent/python/deploy_preview`

#### Input Schema

Input model for deployment preview endpoint.

**Properties:**

| Field             | Type      | Required   | Description   | Default   | Constraints   |
|:------------------|:----------|:-----------|:--------------|:----------|:--------------|
| `branchId`        | `string`  | ✓          |               | -         | -             |
| `isPrivate`       | `boolean` | ✓          |               | -         | -             |
| `environmentVars` | `any`     |            |               | `null`    | -             |

#### Output Schema

Output model for deployment preview endpoint.

**Properties:**

| Field            | Type      | Required   | Description   | Default   | Constraints   |
|:-----------------|:----------|:-----------|:--------------|:----------|:--------------|
| `previewUrl`     | `string`  | ✓          |               | -         | -             |
| `isPrivate`      | `boolean` | ✓          |               | -         | -             |
| `httpAuth`       | `any`     |            |               | `null`    | -             |
| `deploymentTime` | `string`  | ✓          |               | -         | -             |


#### Examples

**Valid Requests:**

Example 1:
```json
{
  "branchId": "feature-123",
  "isPrivate": true,
  "environmentVars": {
    "DEBUG": "true",
    "API_KEY": "preview-key"
  }
}
```


#### Error Responses

|   Status Code | Description                                        | Example Response                                                     |
|--------------:|:---------------------------------------------------|:---------------------------------------------------------------------|
|           400 | Bad Request - Invalid input parameters             | ```json                                                              |
|               |                                                    | {                                                                    |
|               |                                                    |   "error": {                                                         |
|               |                                                    |     "code": "400",                                                   |
|               |                                                    |     "message": "Bad Request - Invalid input parameters",             |
|               |                                                    |     "details": "Additional error context would appear here"          |
|               |                                                    |   }                                                                  |
|               |                                                    | }                                                                    |
|               |                                                    | ```                                                                  |
|           401 | Unauthorized - Authentication required             | ```json                                                              |
|               |                                                    | {                                                                    |
|               |                                                    |   "error": {                                                         |
|               |                                                    |     "code": "401",                                                   |
|               |                                                    |     "message": "Unauthorized - Authentication required",             |
|               |                                                    |     "details": "Additional error context would appear here"          |
|               |                                                    |   }                                                                  |
|               |                                                    | }                                                                    |
|               |                                                    | ```                                                                  |
|           403 | Forbidden - Insufficient permissions               | ```json                                                              |
|               |                                                    | {                                                                    |
|               |                                                    |   "error": {                                                         |
|               |                                                    |     "code": "403",                                                   |
|               |                                                    |     "message": "Forbidden - Insufficient permissions",               |
|               |                                                    |     "details": "Additional error context would appear here"          |
|               |                                                    |   }                                                                  |
|               |                                                    | }                                                                    |
|               |                                                    | ```                                                                  |
|           404 | Not Found - Resource not found                     | ```json                                                              |
|               |                                                    | {                                                                    |
|               |                                                    |   "error": {                                                         |
|               |                                                    |     "code": "404",                                                   |
|               |                                                    |     "message": "Not Found - Resource not found",                     |
|               |                                                    |     "details": "Additional error context would appear here"          |
|               |                                                    |   }                                                                  |
|               |                                                    | }                                                                    |
|               |                                                    | ```                                                                  |
|           422 | Unprocessable Entity - Validation error            | ```json                                                              |
|               |                                                    | {                                                                    |
|               |                                                    |   "error": {                                                         |
|               |                                                    |     "code": "422",                                                   |
|               |                                                    |     "message": "Unprocessable Entity - Validation error",            |
|               |                                                    |     "details": "Additional error context would appear here"          |
|               |                                                    |   }                                                                  |
|               |                                                    | }                                                                    |
|               |                                                    | ```                                                                  |
|           500 | Internal Server Error - Server-side error occurred | ```json                                                              |
|               |                                                    | {                                                                    |
|               |                                                    |   "error": {                                                         |
|               |                                                    |     "code": "500",                                                   |
|               |                                                    |     "message": "Internal Server Error - Server-side error occurred", |
|               |                                                    |     "details": "Additional error context would appear here"          |
|               |                                                    |   }                                                                  |
|               |                                                    | }                                                                    |
|               |                                                    | ```                                                                  |
### generateCode

**Endpoint:** `POST /v1/code_agent/python/generate_code`

#### Input Schema

Input model for code generation endpoint.

**Properties:**

| Field                | Type                                         | Required   | Description   | Default      | Constraints   |
|:---------------------|:---------------------------------------------|:-----------|:--------------|:-------------|:--------------|
| `codeRequirements`   | `any`                                        | ✓          |               | -            | -             |
| `styleGuide`         | `any`                                        |            |               | `null`       | -             |
| `includeTests`       | `boolean`                                    |            |               | `true`       | -             |
| `documentationLevel` | `string (enum: minimal, standard, detailed)` |            |               | `"standard"` | -             |

#### Output Schema

Output model for code generation endpoint.

**Properties:**

| Field           | Type              | Required   | Description   | Default   | Constraints   |
|:----------------|:------------------|:-----------|:--------------|:----------|:--------------|
| `generatedCode` | `string`          | ✓          |               | -         | -             |
| `description`   | `string`          | ✓          |               | -         | -             |
| `testCases`     | `array of string` | ✓          |               | -         | -             |
| `documentation` | `string`          | ✓          |               | -         | -             |


#### Examples

**Valid Requests:**

Example 1:
```json
{
  "codeRequirements": {
    "language": "Python",
    "framework": "FastAPI",
    "description": "Create a REST API endpoint",
    "requirements": [
      "FastAPI",
      "RESTful API design",
      "HTTP methods"
    ],
    "requiredFunctions": [
      "create_endpoint",
      "handle_request",
      "validate_input"
    ],
    "testingRequirements": [
      "test_endpoint_creation",
      "test_request_handling",
      "test_input_validation"
    ],
    "codingStyle": {
      "patterns": [
        "REST API",
        "Clean Architecture"
      ],
      "conventions": [
        "PEP 8",
        "FastAPI best practices"
      ]
    }
  },
  "styleGuide": {
    "formatting": "black",
    "maxLineLength": 88
  },
  "includeTests": true,
  "documentationLevel": "detailed"
}
```


#### Error Responses

|   Status Code | Description                                        | Example Response                                                     |
|--------------:|:---------------------------------------------------|:---------------------------------------------------------------------|
|           400 | Bad Request - Invalid input parameters             | ```json                                                              |
|               |                                                    | {                                                                    |
|               |                                                    |   "error": {                                                         |
|               |                                                    |     "code": "400",                                                   |
|               |                                                    |     "message": "Bad Request - Invalid input parameters",             |
|               |                                                    |     "details": "Additional error context would appear here"          |
|               |                                                    |   }                                                                  |
|               |                                                    | }                                                                    |
|               |                                                    | ```                                                                  |
|           401 | Unauthorized - Authentication required             | ```json                                                              |
|               |                                                    | {                                                                    |
|               |                                                    |   "error": {                                                         |
|               |                                                    |     "code": "401",                                                   |
|               |                                                    |     "message": "Unauthorized - Authentication required",             |
|               |                                                    |     "details": "Additional error context would appear here"          |
|               |                                                    |   }                                                                  |
|               |                                                    | }                                                                    |
|               |                                                    | ```                                                                  |
|           403 | Forbidden - Insufficient permissions               | ```json                                                              |
|               |                                                    | {                                                                    |
|               |                                                    |   "error": {                                                         |
|               |                                                    |     "code": "403",                                                   |
|               |                                                    |     "message": "Forbidden - Insufficient permissions",               |
|               |                                                    |     "details": "Additional error context would appear here"          |
|               |                                                    |   }                                                                  |
|               |                                                    | }                                                                    |
|               |                                                    | ```                                                                  |
|           404 | Not Found - Resource not found                     | ```json                                                              |
|               |                                                    | {                                                                    |
|               |                                                    |   "error": {                                                         |
|               |                                                    |     "code": "404",                                                   |
|               |                                                    |     "message": "Not Found - Resource not found",                     |
|               |                                                    |     "details": "Additional error context would appear here"          |
|               |                                                    |   }                                                                  |
|               |                                                    | }                                                                    |
|               |                                                    | ```                                                                  |
|           422 | Unprocessable Entity - Validation error            | ```json                                                              |
|               |                                                    | {                                                                    |
|               |                                                    |   "error": {                                                         |
|               |                                                    |     "code": "422",                                                   |
|               |                                                    |     "message": "Unprocessable Entity - Validation error",            |
|               |                                                    |     "details": "Additional error context would appear here"          |
|               |                                                    |   }                                                                  |
|               |                                                    | }                                                                    |
|               |                                                    | ```                                                                  |
|           500 | Internal Server Error - Server-side error occurred | ```json                                                              |
|               |                                                    | {                                                                    |
|               |                                                    |   "error": {                                                         |
|               |                                                    |     "code": "500",                                                   |
|               |                                                    |     "message": "Internal Server Error - Server-side error occurred", |
|               |                                                    |     "details": "Additional error context would appear here"          |
|               |                                                    |   }                                                                  |
|               |                                                    | }                                                                    |
|               |                                                    | ```                                                                  |
### improveCode

**Endpoint:** `POST /v1/code_agent/python/improve_code`

#### Input Schema

Input model for code improvement endpoint.

**Properties:**

| Field                  | Type           | Required   | Description   | Default   | Constraints   |
|:-----------------------|:---------------|:-----------|:--------------|:----------|:--------------|
| `changesList`          | `array of any` | ✓          |               | -         | -             |
| `applyBlackFormatting` | `boolean`      |            |               | `true`    | -             |
| `runLinter`            | `boolean`      |            |               | `true`    | -             |

#### Output Schema

Output model for code improvement endpoint.

**Properties:**

| Field                | Type           | Required   | Description   | Default   | Constraints   |
|:---------------------|:---------------|:-----------|:--------------|:----------|:--------------|
| `codeChanges`        | `array of any` | ✓          |               | -         | -             |
| `changesDescription` | `string`       | ✓          |               | -         | -             |
| `qualityMetrics`     | `any`          | ✓          |               | -         | -             |


#### Examples

**Valid Requests:**

Example 1:
```json
{
  "changesList": [
    {
      "type": "refactor",
      "description": "Improve function structure",
      "target": "main.py",
      "priority": "medium"
    }
  ],
  "applyBlackFormatting": true,
  "runLinter": true
}
```


#### Error Responses

|   Status Code | Description                                        | Example Response                                                     |
|--------------:|:---------------------------------------------------|:---------------------------------------------------------------------|
|           400 | Bad Request - Invalid input parameters             | ```json                                                              |
|               |                                                    | {                                                                    |
|               |                                                    |   "error": {                                                         |
|               |                                                    |     "code": "400",                                                   |
|               |                                                    |     "message": "Bad Request - Invalid input parameters",             |
|               |                                                    |     "details": "Additional error context would appear here"          |
|               |                                                    |   }                                                                  |
|               |                                                    | }                                                                    |
|               |                                                    | ```                                                                  |
|           401 | Unauthorized - Authentication required             | ```json                                                              |
|               |                                                    | {                                                                    |
|               |                                                    |   "error": {                                                         |
|               |                                                    |     "code": "401",                                                   |
|               |                                                    |     "message": "Unauthorized - Authentication required",             |
|               |                                                    |     "details": "Additional error context would appear here"          |
|               |                                                    |   }                                                                  |
|               |                                                    | }                                                                    |
|               |                                                    | ```                                                                  |
|           403 | Forbidden - Insufficient permissions               | ```json                                                              |
|               |                                                    | {                                                                    |
|               |                                                    |   "error": {                                                         |
|               |                                                    |     "code": "403",                                                   |
|               |                                                    |     "message": "Forbidden - Insufficient permissions",               |
|               |                                                    |     "details": "Additional error context would appear here"          |
|               |                                                    |   }                                                                  |
|               |                                                    | }                                                                    |
|               |                                                    | ```                                                                  |
|           404 | Not Found - Resource not found                     | ```json                                                              |
|               |                                                    | {                                                                    |
|               |                                                    |   "error": {                                                         |
|               |                                                    |     "code": "404",                                                   |
|               |                                                    |     "message": "Not Found - Resource not found",                     |
|               |                                                    |     "details": "Additional error context would appear here"          |
|               |                                                    |   }                                                                  |
|               |                                                    | }                                                                    |
|               |                                                    | ```                                                                  |
|           422 | Unprocessable Entity - Validation error            | ```json                                                              |
|               |                                                    | {                                                                    |
|               |                                                    |   "error": {                                                         |
|               |                                                    |     "code": "422",                                                   |
|               |                                                    |     "message": "Unprocessable Entity - Validation error",            |
|               |                                                    |     "details": "Additional error context would appear here"          |
|               |                                                    |   }                                                                  |
|               |                                                    | }                                                                    |
|               |                                                    | ```                                                                  |
|           500 | Internal Server Error - Server-side error occurred | ```json                                                              |
|               |                                                    | {                                                                    |
|               |                                                    |   "error": {                                                         |
|               |                                                    |     "code": "500",                                                   |
|               |                                                    |     "message": "Internal Server Error - Server-side error occurred", |
|               |                                                    |     "details": "Additional error context would appear here"          |
|               |                                                    |   }                                                                  |
|               |                                                    | }                                                                    |
|               |                                                    | ```                                                                  |
### testCode

**Endpoint:** `POST /v1/code_agent/python/test_code`

#### Input Schema

Input model for code testing endpoint.

**Properties:**

| Field              | Type           | Required   | Description   | Default   | Constraints                  |
|:-------------------|:---------------|:-----------|:--------------|:----------|:-----------------------------|
| `testType`         | `any`          | ✓          |               | -         | -                            |
| `requirePassing`   | `boolean`      | ✓          |               | -         | -                            |
| `testInstructions` | `array of any` | ✓          |               | -         | -                            |
| `codeToTest`       | `string`       | ✓          |               | -         | -                            |
| `minimumCoverage`  | `number`       |            |               | `80.0`    | minimum: 0.0, maximum: 100.0 |

#### Output Schema

Output model for code testing endpoint.

**Properties:**

| Field              | Type     | Required   | Description   | Default   | Constraints   |
|:-------------------|:---------|:-----------|:--------------|:----------|:--------------|
| `codeTests`        | `string` | ✓          |               | -         | -             |
| `testsDescription` | `string` | ✓          |               | -         | -             |
| `coverageStatus`   | `any`    | ✓          |               | -         | -             |


#### Examples

**Valid Requests:**

Example 1:
```json
{
  "testType": "unit",
  "requirePassing": true,
  "testInstructions": [
    {
      "description": "Test API endpoints",
      "assertions": [
        "test_status_code",
        "test_response_format"
      ],
      "testType": "unit"
    }
  ],
  "codeToTest": "def example(): return True",
  "minimumCoverage": 80.0
}
```


#### Error Responses

|   Status Code | Description                                        | Example Response                                                     |
|--------------:|:---------------------------------------------------|:---------------------------------------------------------------------|
|           400 | Bad Request - Invalid input parameters             | ```json                                                              |
|               |                                                    | {                                                                    |
|               |                                                    |   "error": {                                                         |
|               |                                                    |     "code": "400",                                                   |
|               |                                                    |     "message": "Bad Request - Invalid input parameters",             |
|               |                                                    |     "details": "Additional error context would appear here"          |
|               |                                                    |   }                                                                  |
|               |                                                    | }                                                                    |
|               |                                                    | ```                                                                  |
|           401 | Unauthorized - Authentication required             | ```json                                                              |
|               |                                                    | {                                                                    |
|               |                                                    |   "error": {                                                         |
|               |                                                    |     "code": "401",                                                   |
|               |                                                    |     "message": "Unauthorized - Authentication required",             |
|               |                                                    |     "details": "Additional error context would appear here"          |
|               |                                                    |   }                                                                  |
|               |                                                    | }                                                                    |
|               |                                                    | ```                                                                  |
|           403 | Forbidden - Insufficient permissions               | ```json                                                              |
|               |                                                    | {                                                                    |
|               |                                                    |   "error": {                                                         |
|               |                                                    |     "code": "403",                                                   |
|               |                                                    |     "message": "Forbidden - Insufficient permissions",               |
|               |                                                    |     "details": "Additional error context would appear here"          |
|               |                                                    |   }                                                                  |
|               |                                                    | }                                                                    |
|               |                                                    | ```                                                                  |
|           404 | Not Found - Resource not found                     | ```json                                                              |
|               |                                                    | {                                                                    |
|               |                                                    |   "error": {                                                         |
|               |                                                    |     "code": "404",                                                   |
|               |                                                    |     "message": "Not Found - Resource not found",                     |
|               |                                                    |     "details": "Additional error context would appear here"          |
|               |                                                    |   }                                                                  |
|               |                                                    | }                                                                    |
|               |                                                    | ```                                                                  |
|           422 | Unprocessable Entity - Validation error            | ```json                                                              |
|               |                                                    | {                                                                    |
|               |                                                    |   "error": {                                                         |
|               |                                                    |     "code": "422",                                                   |
|               |                                                    |     "message": "Unprocessable Entity - Validation error",            |
|               |                                                    |     "details": "Additional error context would appear here"          |
|               |                                                    |   }                                                                  |
|               |                                                    | }                                                                    |
|               |                                                    | ```                                                                  |
|           500 | Internal Server Error - Server-side error occurred | ```json                                                              |
|               |                                                    | {                                                                    |
|               |                                                    |   "error": {                                                         |
|               |                                                    |     "code": "500",                                                   |
|               |                                                    |     "message": "Internal Server Error - Server-side error occurred", |
|               |                                                    |     "details": "Additional error context would appear here"          |
|               |                                                    |   }                                                                  |
|               |                                                    | }                                                                    |
|               |                                                    | ```                                                                  |