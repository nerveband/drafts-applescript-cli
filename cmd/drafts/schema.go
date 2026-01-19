package main

// Schema returns the tool-use formatted schema for all commands
func getSchema(command string) interface{} {
	schema := map[string]interface{}{
		"name":    "drafts",
		"version": "0.2.0",
		"tools":   getTools(),
	}

	if command != "" {
		// Return schema for single command
		tools := getTools()
		for _, tool := range tools {
			t := tool.(map[string]interface{})
			if t["name"] == "drafts_"+command {
				return t
			}
		}
		outputError("UNKNOWN_COMMAND",
			"Unknown command: "+command,
			"Use 'drafts schema' to see all available commands")
	}

	return schema
}

func getTools() []interface{} {
	return []interface{}{
		map[string]interface{}{
			"name":        "drafts_new",
			"description": "Create a new draft in Drafts.app",
			"parameters": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"content": map[string]interface{}{
						"type":        "string",
						"description": "The draft content",
					},
					"tags": map[string]interface{}{
						"type":        "array",
						"items":       map[string]interface{}{"type": "string"},
						"description": "Tags to apply to the draft",
					},
					"folder": map[string]interface{}{
						"type":        "string",
						"enum":        []string{"inbox", "archive"},
						"default":     "inbox",
						"description": "Folder to create draft in",
					},
					"flagged": map[string]interface{}{
						"type":        "boolean",
						"default":     false,
						"description": "Whether to flag the draft",
					},
					"action": map[string]interface{}{
						"type":        "string",
						"description": "Action name to run after creation",
					},
				},
				"required": []string{},
			},
		},
		map[string]interface{}{
			"name":        "drafts_get",
			"description": "Get a draft by UUID, returns full draft metadata",
			"parameters": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"uuid": map[string]interface{}{
						"type":        "string",
						"description": "UUID of the draft (omit for active draft)",
					},
				},
				"required": []string{},
			},
		},
		map[string]interface{}{
			"name":        "drafts_list",
			"description": "List drafts with optional filtering",
			"parameters": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"filter": map[string]interface{}{
						"type":        "string",
						"enum":        []string{"inbox", "flagged", "archive", "trash", "all"},
						"default":     "inbox",
						"description": "Filter drafts by folder",
					},
					"tags": map[string]interface{}{
						"type":        "array",
						"items":       map[string]interface{}{"type": "string"},
						"description": "Filter by tags",
					},
				},
				"required": []string{},
			},
		},
		map[string]interface{}{
			"name":        "drafts_append",
			"description": "Append text to an existing draft",
			"parameters": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"uuid": map[string]interface{}{
						"type":        "string",
						"description": "UUID of the draft (omit for active draft)",
					},
					"content": map[string]interface{}{
						"type":        "string",
						"description": "Text to append",
					},
					"tags": map[string]interface{}{
						"type":        "array",
						"items":       map[string]interface{}{"type": "string"},
						"description": "Tags to add",
					},
					"action": map[string]interface{}{
						"type":        "string",
						"description": "Action to run after appending",
					},
				},
				"required": []string{"content"},
			},
		},
		map[string]interface{}{
			"name":        "drafts_prepend",
			"description": "Prepend text to an existing draft",
			"parameters": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"uuid": map[string]interface{}{
						"type":        "string",
						"description": "UUID of the draft (omit for active draft)",
					},
					"content": map[string]interface{}{
						"type":        "string",
						"description": "Text to prepend",
					},
					"tags": map[string]interface{}{
						"type":        "array",
						"items":       map[string]interface{}{"type": "string"},
						"description": "Tags to add",
					},
					"action": map[string]interface{}{
						"type":        "string",
						"description": "Action to run after prepending",
					},
				},
				"required": []string{"content"},
			},
		},
		map[string]interface{}{
			"name":        "drafts_replace",
			"description": "Replace the content of an existing draft",
			"parameters": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"uuid": map[string]interface{}{
						"type":        "string",
						"description": "UUID of the draft (omit for active draft)",
					},
					"content": map[string]interface{}{
						"type":        "string",
						"description": "New content for the draft",
					},
				},
				"required": []string{"content"},
			},
		},
		map[string]interface{}{
			"name":        "drafts_run",
			"description": "Run a Drafts action on text or an existing draft",
			"parameters": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"action": map[string]interface{}{
						"type":        "string",
						"description": "Name of the action to run",
					},
					"content": map[string]interface{}{
						"type":        "string",
						"description": "Text to process (ignored if uuid provided)",
					},
					"uuid": map[string]interface{}{
						"type":        "string",
						"description": "UUID of draft to run action on",
					},
				},
				"required": []string{"action"},
			},
		},
	}
}
