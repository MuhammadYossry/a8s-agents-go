# AI Agents Service Documentation

Welcome to our AI Agents service documentation. This service hosts several AI agents, each providing specific capabilities through well-documented endpoints. Below you'll find detailed information about each agent, their capabilities, and how to interact with them.

## python-code-agent

**Description:** Advanced Python code generation, testing, and deployment agent

**Base URL:** `http://localhost:9200`

## Capabilities

The following sections detail the specific capabilities of this agent:

### Development

**Capability Details:**
- expertise: advanced

### Development → Backend → Python

**Capability Details:**
- expertise: advanced
- versions: 3.8, 3.9, 3.10
- frameworks: Django, FastAPI

### Development → Backend → Python → CodeGeneration

**Capability Details:**
- versions: 3.8, 3.9, 3.10
- frameworks: Django, FastAPI
- tools: black, pylint

### Development → Testing → Python

**Capability Details:**
- expertise: advanced
- frameworks: pytest, unittest

### Development → Deployment → Python

**Capability Details:**
- expertise: basic
- platforms: AWS, GCP, Azure

## Available Endpoints

This section describes all available endpoints for interacting with the agent:

### deployPreview

**Endpoint:** `POST /v1/code_agent/python/deploy_preview`

#### Input Schema

Input model for deployment preview endpoint.

**Properties:**

- `branchId`: 
  * Type: `string`
  * Required: Yes

- `isPrivate`: 
  * Type: `boolean`
  * Required: Yes

- `environmentVars`: 
  * Type: `any`
  * Default: `null`



#### Output Schema

Output model for deployment preview endpoint.

**Properties:**

- `previewUrl`: 
  * Type: `string`
  * Required: Yes

- `isPrivate`: 
  * Type: `boolean`
  * Required: Yes

- `httpAuth`: 
  * Type: `any`
  * Default: `null`

- `deploymentTime`: 
  * Type: `string`
  * Required: Yes




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

**Status 400**: Bad Request - Invalid input parameters
Example:
```json
{
  "error": {
    "code": "400",
    "message": "Bad Request - Invalid input parameters",
    "details": "Additional error context would appear here"
  }
}
```

**Status 401**: Unauthorized - Authentication required
Example:
```json
{
  "error": {
    "code": "401",
    "message": "Unauthorized - Authentication required",
    "details": "Additional error context would appear here"
  }
}
```

**Status 403**: Forbidden - Insufficient permissions
Example:
```json
{
  "error": {
    "code": "403",
    "message": "Forbidden - Insufficient permissions",
    "details": "Additional error context would appear here"
  }
}
```

**Status 404**: Not Found - Resource not found
Example:
```json
{
  "error": {
    "code": "404",
    "message": "Not Found - Resource not found",
    "details": "Additional error context would appear here"
  }
}
```

**Status 422**: Unprocessable Entity - Validation error
Example:
```json
{
  "error": {
    "code": "422",
    "message": "Unprocessable Entity - Validation error",
    "details": "Additional error context would appear here"
  }
}
```

**Status 500**: Internal Server Error - Server-side error occurred
Example:
```json
{
  "error": {
    "code": "500",
    "message": "Internal Server Error - Server-side error occurred",
    "details": "Additional error context would appear here"
  }
}
```

### generateCode

**Endpoint:** `POST /v1/code_agent/python/generate_code`

#### Input Schema

Input model for code generation endpoint.

**Properties:**

- `codeRequirements`: 
  * Type: `any`
  * Required: Yes

- `styleGuide`: 
  * Type: `any`
  * Default: `null`

- `includeTests`: 
  * Type: `boolean`
  * Default: `true`

- `documentationLevel`: 
  * Type: `string (enum: minimal, standard, detailed)`
  * Default: `"standard"`



#### Output Schema

Output model for code generation endpoint.

**Properties:**

- `generatedCode`: 
  * Type: `string`
  * Required: Yes

- `description`: 
  * Type: `string`
  * Required: Yes

- `testCases`: 
  * Type: `array of string`
  * Required: Yes

- `documentation`: 
  * Type: `string`
  * Required: Yes




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

**Status 400**: Bad Request - Invalid input parameters
Example:
```json
{
  "error": {
    "code": "400",
    "message": "Bad Request - Invalid input parameters",
    "details": "Additional error context would appear here"
  }
}
```

**Status 401**: Unauthorized - Authentication required
Example:
```json
{
  "error": {
    "code": "401",
    "message": "Unauthorized - Authentication required",
    "details": "Additional error context would appear here"
  }
}
```

**Status 403**: Forbidden - Insufficient permissions
Example:
```json
{
  "error": {
    "code": "403",
    "message": "Forbidden - Insufficient permissions",
    "details": "Additional error context would appear here"
  }
}
```

**Status 404**: Not Found - Resource not found
Example:
```json
{
  "error": {
    "code": "404",
    "message": "Not Found - Resource not found",
    "details": "Additional error context would appear here"
  }
}
```

**Status 422**: Unprocessable Entity - Validation error
Example:
```json
{
  "error": {
    "code": "422",
    "message": "Unprocessable Entity - Validation error",
    "details": "Additional error context would appear here"
  }
}
```

**Status 500**: Internal Server Error - Server-side error occurred
Example:
```json
{
  "error": {
    "code": "500",
    "message": "Internal Server Error - Server-side error occurred",
    "details": "Additional error context would appear here"
  }
}
```

### improveCode

**Endpoint:** `POST /v1/code_agent/python/improve_code`

#### Input Schema

Input model for code improvement endpoint.

**Properties:**

- `changesList`: 
  * Type: `array of any`
  * Required: Yes

- `applyBlackFormatting`: 
  * Type: `boolean`
  * Default: `true`

- `runLinter`: 
  * Type: `boolean`
  * Default: `true`



#### Output Schema

Output model for code improvement endpoint.

**Properties:**

- `codeChanges`: 
  * Type: `array of any`
  * Required: Yes

- `changesDescription`: 
  * Type: `string`
  * Required: Yes

- `qualityMetrics`: 
  * Type: `any`
  * Required: Yes




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

**Status 400**: Bad Request - Invalid input parameters
Example:
```json
{
  "error": {
    "code": "400",
    "message": "Bad Request - Invalid input parameters",
    "details": "Additional error context would appear here"
  }
}
```

**Status 401**: Unauthorized - Authentication required
Example:
```json
{
  "error": {
    "code": "401",
    "message": "Unauthorized - Authentication required",
    "details": "Additional error context would appear here"
  }
}
```

**Status 403**: Forbidden - Insufficient permissions
Example:
```json
{
  "error": {
    "code": "403",
    "message": "Forbidden - Insufficient permissions",
    "details": "Additional error context would appear here"
  }
}
```

**Status 404**: Not Found - Resource not found
Example:
```json
{
  "error": {
    "code": "404",
    "message": "Not Found - Resource not found",
    "details": "Additional error context would appear here"
  }
}
```

**Status 422**: Unprocessable Entity - Validation error
Example:
```json
{
  "error": {
    "code": "422",
    "message": "Unprocessable Entity - Validation error",
    "details": "Additional error context would appear here"
  }
}
```

**Status 500**: Internal Server Error - Server-side error occurred
Example:
```json
{
  "error": {
    "code": "500",
    "message": "Internal Server Error - Server-side error occurred",
    "details": "Additional error context would appear here"
  }
}
```

### testCode

**Endpoint:** `POST /v1/code_agent/python/test_code`

#### Input Schema

Input model for code testing endpoint.

**Properties:**

- `testType`: 
  * Type: `any`
  * Required: Yes

- `requirePassing`: 
  * Type: `boolean`
  * Required: Yes

- `testInstructions`: 
  * Type: `array of any`
  * Required: Yes

- `codeToTest`: 
  * Type: `string`
  * Required: Yes

- `minimumCoverage`: 
  * Type: `number`
  * Default: `80.0`
  * Constraints: minimum: 0.0, maximum: 100.0



#### Output Schema

Output model for code testing endpoint.

**Properties:**

- `codeTests`: 
  * Type: `string`
  * Required: Yes

- `testsDescription`: 
  * Type: `string`
  * Required: Yes

- `coverageStatus`: 
  * Type: `any`
  * Required: Yes




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

**Status 400**: Bad Request - Invalid input parameters
Example:
```json
{
  "error": {
    "code": "400",
    "message": "Bad Request - Invalid input parameters",
    "details": "Additional error context would appear here"
  }
}
```

**Status 401**: Unauthorized - Authentication required
Example:
```json
{
  "error": {
    "code": "401",
    "message": "Unauthorized - Authentication required",
    "details": "Additional error context would appear here"
  }
}
```

**Status 403**: Forbidden - Insufficient permissions
Example:
```json
{
  "error": {
    "code": "403",
    "message": "Forbidden - Insufficient permissions",
    "details": "Additional error context would appear here"
  }
}
```

**Status 404**: Not Found - Resource not found
Example:
```json
{
  "error": {
    "code": "404",
    "message": "Not Found - Resource not found",
    "details": "Additional error context would appear here"
  }
}
```

**Status 422**: Unprocessable Entity - Validation error
Example:
```json
{
  "error": {
    "code": "422",
    "message": "Unprocessable Entity - Validation error",
    "details": "Additional error context would appear here"
  }
}
```

**Status 500**: Internal Server Error - Server-side error occurred
Example:
```json
{
  "error": {
    "code": "500",
    "message": "Internal Server Error - Server-side error occurred",
    "details": "Additional error context would appear here"
  }
}
```
