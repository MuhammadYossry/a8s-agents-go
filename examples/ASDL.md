# Agent Service Definition Language (ASDL) v1.0

ASDL is a domain-specific language designed to define AI agent capabilities, behaviors, and interaction protocols in a clear, structured format. It provides a standardized way to describe how agents expose their services and interact with users.

## Basic Structure

The basic structure of an ASDL definition follows this pattern:

```asdl
agent agent_name {
    category: type
    version: "version_number"
    base_url: "service_endpoint"
    purpose: """agent_description"""
    
    cognitive_abilities {
        // Agent capabilities
    }
    
    behaviors {
        // Interaction definitions
    }
}
```

## Core Keywords

### Agent Definition
- `agent`: Top-level declaration of an agent service
- `category`: Type of agent (e.g., "external", "internal")
- `version`: Service version number
- `base_url`: Base URL for all agent endpoints
- `purpose`: Detailed description of the agent's purpose and functionality

### Cognitive Abilities
The `cognitive_abilities` block defines what the agent knows and can do:

```asdl
cognitive_abilities {
    knowledge_domain DomainName {
        proficiency_expertise: level
        
        specialization Area {
            proficiency_expertise: level
            capabilities: [
                "capability1",
                "capability2"
            ]
        }
        
        skill SkillName {
            expertise: level
            // Additional skill properties
        }
    }
}
```

- `knowledge_domain`: Main area of expertise
- `specialization`: Specific area within a domain
- `skill`: Specific capability or tool proficiency
- `proficiency_expertise`: Expertise level (e.g., "basic", "advanced")
- `capabilities`: List of specific capabilities within a specialization

### Behaviors
The `behaviors` block defines how to interact with the agent:

```asdl
behaviors {
    interaction actionName {
        endpoint: "/path/to/endpoint"
        protocol: HTTP_METHOD
        
        expects {
            // Input parameters
        }
        
        provides {
            // Output structure
        }
        
        behavior_example {
            input {
                // Example request
            }
        }
        
        error_handlers {
            // Error definitions
        }
    }
}
```

- `interaction`: Definition of a specific agent action
- `endpoint`: API endpoint path
- `protocol`: HTTP method (GET, POST, etc.)
- `expects`: Input parameters specification
- `provides`: Output structure specification
- `behavior_example`: Example usage
- `error_handlers`: Error handling definitions

## Type System

ASDL provides several built-in types for defining parameters:

- `content`: Text or string content
- `numeric`: Numerical values (can include ranges: `numeric(0..100)`)
- `logical`: Boolean values
- `collection<type>`: List or array of specified type
- `oneof(value1, value2)`: Enumeration of possible values
- `fixed(value)`: Constant value

Parameter modifiers:
- `required`: Mandatory parameter
- `optional`: Optional parameter (default)
- Default values can be specified using `= value`

## Examples

### Parameter Definition Examples

```asdl
expects {
    required name: content
    age: numeric(0..120)
    tags: collection<content>
    status: oneof("active", "inactive") = "active"
    isEnabled: logical = true
}
```

### Behavior Example

```asdl
behavior_example {
    input {
        "name": "example",
        "age": 25,
        "tags": ["tag1", "tag2"],
        "status": "active",
        "isEnabled": true
    }
}
```

## Best Practices

1. **Clear Naming**
   - Use descriptive names for interactions
   - Follow camelCase for property names
   - Use clear domain terminology

2. **Documentation**
   - Provide detailed purpose descriptions
   - Include meaningful behavior examples
   - Document error conditions

3. **Type Safety**
   - Use appropriate types for parameters
   - Specify constraints where applicable
   - Include default values when relevant

4. **Organization**
   - Group related capabilities in domains
   - Structure behaviors logically
   - Keep examples clear and minimal

## Syntax Rules

1. Use curly braces `{}` for blocks
2. Use colons `:` for property assignments
3. Use commas for list items
4. Use triple quotes `"""` for multi-line descriptions
5. Indent nested blocks for readability
6. Use square brackets `[]` for capability lists

This documentation provides a foundation for writing ASDL definitions. For more specific examples or advanced usage patterns, refer to the example agents in our repository.