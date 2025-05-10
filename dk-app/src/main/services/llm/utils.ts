#!/usr/bin/env node

import * as readline from 'readline-sync';
import axios from 'axios';
import * as dotenv from 'dotenv';
import chalk from 'chalk';
import * as fs from 'fs';
import * as path from 'path';
import { Client } from "@modelcontextprotocol/sdk/client/index.js";
import { StdioClientTransport } from "@modelcontextprotocol/sdk/client/stdio.js";




export async function setupMCPConfig(provider: string = 'openai') {
  // Initialize MCP client
  const client = new Client({
    name: "mcp-chatbot",
    version: "1.0.0"
  });

  // Initialize tool collection
  let allTools: any[] = [];

  // Read MCP config file
  const mcpConfig = readMCPConfig();

  if (mcpConfig !== null) {
    for (const name in mcpConfig.mcpServers) {
      if (mcpConfig.mcpServers.hasOwnProperty(name)) {
        console.log(chalk.green(`Starting ${name} server...`));
        const serverConfig = mcpConfig.mcpServers[name];
        const transport = new StdioClientTransport({
          command: serverConfig.command,
          args: serverConfig.args,
        });
        await client.connect(transport);

        try {
          // List tools from this server
          const toolsResult = await client.listTools();

          // Check if toolsResult has a tools property (it should be an object with a tools array)
          if (!toolsResult || !toolsResult.tools || !Array.isArray(toolsResult.tools)) {
            console.error(chalk.red(`Error: Invalid tools result from ${name}: ${JSON.stringify(toolsResult)}`));
            continue;
          }

          console.log(chalk.green(`Loaded ${toolsResult.tools.length} tools from ${name}`));

          // Convert tools to provider format and add to collection
          const providerTools = mcpToolsToProviderTools(toolsResult.tools, provider as ProviderType, name);
          allTools = [...allTools, ...providerTools];
        } catch (error) {
          console.error(chalk.red(`Error fetching tools from ${name}: ${(error as Error).message}`));
        }
      }
    }
  }

  if (allTools.length > 0) {
    console.log(chalk.green(`Loaded ${allTools.length} tools in total`));
  } else {
    console.log(chalk.yellow('No tools loaded. MCP functionality will be limited.'));
  }

  if (mcpConfig && Object.keys(mcpConfig.mcpServers).length > 0) {
    console.log(chalk.green(`Connected to MCP servers: ${Object.keys(mcpConfig.mcpServers).join(', ')}`));
  }

  return {client: client, tools: allTools}
}

interface OllamaResponse {
  message: Message;
  model: string;
  created_at: string;
  done: boolean;
}

interface AnthropicMessage {
  role: string;
  content: Array<{
    type: string;
    text?: string;
    id?: string;
    name?: string;
    input?: Record<string, any>;
    tool_use_id?: string;
  }>;
}

interface AnthropicResponse {
  content: Array<{
    type: string;
    text?: string;
  }>;
  id: string;
  model: string;
  role: string;
  tool_calls?: Array<{
    id: string;
    name: string;
    type: string;
    input: Record<string, any>;
  }>;
}

interface OpenAIResponse {
  id: string;
  choices: Array<{
    message: {
      role: string;
      content: string | null;
      tool_calls?: Array<{
        id: string;
        type: string;
        function: {
          name: string;
          arguments: string;
        }
      }>;
    };
  }>;
  model: string;
}

interface AxiosError {
  response?: {
    data: unknown;
  };
  message: string;
}

interface MCPServerConfig {
  command: string;
  args: string[];
}

interface MCPConfig {
  mcpServers: {
    [name: string]: MCPServerConfig;
  };
}

interface Tool {
  name: string;
  description: string;
  input_schema: {
    type: string;
    properties: Record<string, any>;
    required: string[];
  };
}

// Provider types
type ProviderType = 'openai' | 'anthropic' | 'ollama';

// Function to read the MCP configuration file
export function readMCPConfig(configName: string = 'mcpconfig.json'): MCPConfig | null {
  try {
    // Get app paths to find userDataPath directory
    const { app } = require('electron');
    const userDataPath = app.getPath('userData');

    // First try to find the MCP config in the app's userData directory
    const configFilePath = path.join(userDataPath, configName);

    console.log(chalk.blue(`Looking for MCP config at: ${configFilePath}`));

    if (fs.existsSync(configFilePath)) {
      console.log(chalk.green(`Found MCP config at: ${configFilePath}`));
      const configContent = fs.readFileSync(configFilePath, 'utf8');
      const config = JSON.parse(configContent) as MCPConfig;
      return config;
    }

    // As a fallback, check for the file in the current directory
    const fallbackPath = path.resolve(configName);

    console.log(chalk.blue(`Looking for MCP config at fallback location: ${fallbackPath}`));

    if (fs.existsSync(fallbackPath)) {
      console.log(chalk.green(`Found MCP config at fallback location: ${fallbackPath}`));
      const configContent = fs.readFileSync(fallbackPath, 'utf8');
      const config = JSON.parse(configContent) as MCPConfig;
      return config;
    }

    console.log(chalk.yellow(`MCP config not found at ${configFilePath} or ${fallbackPath}`));
    return null;
  } catch (error) {
    console.error(chalk.red(`Error reading MCP config: ${(error as Error).message}`));
    return null;
  }
}

// Function to convert MCP tools to provider-specific format
export function mcpToolsToProviderTools(
  tools: any,
  provider: ProviderType,
  serverName: string
): any[] {
  // Ensure tools is an array before calling map
  if (!Array.isArray(tools)) {
    console.error(chalk.red(`Error: Tools is not an array (type: ${typeof tools})`));
    return [];
  }

  return tools.map(tool => {
    // Validate the tool object and its required properties
    if (!tool || !tool.name) {
      console.error(chalk.red(`Invalid tool object: ${JSON.stringify(tool)}`));
      return null;
    }

    const namespacedName = `${serverName}__${tool.name}`;

    // Validate input_schema (renamed to inputSchema in the MCP SDK)
    const inputSchema = tool.input_schema || tool.inputSchema;
    if (!inputSchema) {
      console.error(chalk.red(`Tool ${tool.name} is missing input_schema/inputSchema property`));
      return null;
    }

    if (provider === 'openai') {
      return {
        type: 'function',
        function: {
          name: namespacedName,
          description: tool.description || '',
          parameters: {
            type: inputSchema.type || 'object',
            properties: inputSchema.properties || {},
            required: inputSchema.required || []
          }
        }
      };
    } else if (provider === 'anthropic') {
      return {
        name: namespacedName,
        description: tool.description || '',
        input_schema: {
          type: inputSchema.type || 'object',
          properties: inputSchema.properties || {},
          required: inputSchema.required || []
        }
      };
    } else if (provider === 'ollama') {
      return {
        type: 'function',
        function: {
          name: namespacedName,
          description: tool.description || '',
          parameters: {
            type: inputSchema.type || 'object',
            properties: inputSchema.properties || {},
            required: inputSchema.required || []
          }
        }
      };
    }

    return null;
  }).filter(Boolean);
}

// Process tool calls and execute them
export async function processToolCalls(
  toolCalls: any[],
  client: Client,
  provider: ProviderType
): Promise<Message[]> {
  if (!client) {
    console.error(chalk.red('MCP client is not initialized for tool calls'));
    return [];
  }

  const toolResponses: Message[] = [];
  console.log(chalk.cyan(`Processing ${toolCalls.length} tool calls for provider ${provider}`));

  for (const toolCall of toolCalls) {
    // Validate toolCall has required properties
    if (!toolCall || !toolCall.function) {
      console.error(chalk.red(`Invalid tool call object: ${JSON.stringify(toolCall)}`));
      continue;
    }

    // Declare variables in the outer scope so they're accessible in the catch block
    let serverName = '';
    let toolName = '';
    let args: Record<string, any> = {};

    try {
      // Extract server name and tool name from the namespaced name
      // Handle case where function name might not be namespaced
      let toolNameValue = toolCall.function.name || '';
      const nameParts = toolNameValue.split('__');

      if (nameParts.length !== 2) {
        console.error(chalk.red(`Invalid tool name format: ${toolNameValue}`));
        // If we can't split properly, try to use the whole name as the tool name
        // This is a fallback for non-namespaced tool names
        if (toolNameValue) {
          serverName = 'unknown';
          toolName = toolNameValue;
          console.log(chalk.yellow(`Using fallback: server=${serverName}, tool=${toolName}`));
        } else {
          continue;
        }
      } else {
        serverName = nameParts[0];
        toolName = nameParts[1];
      }

      // Parse the arguments
      try {
        if (typeof toolCall.function.arguments === 'string') {
          // First, ensure the JSON string is valid - sometimes it comes incomplete from streaming
          let argsStr = toolCall.function.arguments.trim();
          console.log(chalk.cyan(`Original arguments string: "${argsStr}"`));

          // Handle empty string case
          if (argsStr === '') {
            console.warn(chalk.yellow(`Empty arguments string, using empty object`));
            args = {};
          } else {
            // Try to fix common JSON formatting issues
            try {
              args = JSON.parse(argsStr);
              console.log(chalk.green(`Successfully parsed arguments: ${JSON.stringify(args)}`));
            } catch (parseError) {
              console.warn(chalk.yellow(`JSON parse error, attempting to fix: ${parseError.message}`));

              // Handle empty object expressed as empty string
              if (argsStr === '' || argsStr === '""') {
                args = {};
              }
              // Fix for arguments that are just a raw string without JSON formatting
              else if (!argsStr.startsWith('{') && !argsStr.endsWith('}')) {
                // Try to make it a proper JSON string
                try {
                  // If it looks like a JSON string without quotes, try to quote it
                  args = { value: argsStr };
                  console.log(chalk.green(`Converted non-JSON string to object: ${JSON.stringify(args)}`));
                } catch (e) {
                  console.error(chalk.red(`Failed to convert string: ${e.message}`));
                  args = { raw: argsStr };
                }
              }
              // Fix for unterminated strings - look for unterminated quotes
              else if (argsStr.includes('"') && (argsStr.match(/"/g) || []).length % 2 === 0) {
                argsStr += '"';
                console.log(chalk.yellow(`Fixed unterminated string: ${argsStr}`));

                try {
                  args = JSON.parse(argsStr);
                } catch (e) {
                  args = { _partialFixed: argsStr };
                }
              }
              // Fix for missing closing braces
              else if (argsStr.startsWith('{')) {
                let openBraces = (argsStr.match(/{/g) || []).length;
                let closeBraces = (argsStr.match(/}/g) || []).length;
                if (openBraces > closeBraces) {
                  argsStr += '}'.repeat(openBraces - closeBraces);
                  console.log(chalk.yellow(`Fixed missing braces: ${argsStr}`));
                }

                try {
                  args = JSON.parse(argsStr);
                  console.log(chalk.green(`Successfully fixed and parsed JSON arguments: ${JSON.stringify(args)}`));
                } catch (fixError) {
                  // If still failing, use empty object instead of failing
                  console.error(chalk.red(`Failed to fix JSON arguments: ${fixError.message}`));
                  args = { _raw: argsStr }; // At least preserve the raw string for debugging
                }
              }
              else {
                // Last resort, preserve the raw text in an object
                console.error(chalk.red(`Could not parse arguments, using raw text: ${argsStr}`));
                args = { _raw: argsStr };
              }
            }
          }
        } else if (typeof toolCall.function.arguments === 'object') {
          // Already an object, use directly
          args = toolCall.function.arguments || {};
          console.log(chalk.green(`Arguments already an object: ${JSON.stringify(args)}`));
        } else {
          // Fallback for undefined/null/etc
          console.warn(chalk.yellow(`Arguments not string or object (${typeof toolCall.function.arguments}), using empty object`));
          args = {};
        }
      } catch (error) {
        console.error(chalk.red(`Error processing tool arguments: ${(error as Error).message}`));
        args = {}; // Use empty object instead of failing
      }

      console.log(chalk.cyan(`Calling tool: ${toolName} on server: ${serverName}`));

      // Call the tool
      console.log(chalk.cyan(`Tool call payload:`, JSON.stringify({
        name: toolName,
        arguments: args
      }, null, 2)));

      // Make sure client has callTool method
      if (!client.callTool) {
        console.error(chalk.red(`MCP client missing callTool method. Client keys: ${Object.keys(client).join(', ')}`));
        console.error(chalk.red(`Client type: ${typeof client}`));

        // If the client has a property that looks like callTool but isn't exactly that
        const possibleMethods = Object.keys(client).filter(k => k.toLowerCase().includes('tool') || k.toLowerCase().includes('call'));
        if (possibleMethods.length > 0) {
          console.error(chalk.yellow(`Possible alternative methods: ${possibleMethods.join(', ')}`));
        }

        throw new Error('MCP client missing callTool method');
      }

      // Call the tool
      console.log(chalk.yellow(`Calling tool ${toolName} on server ${serverName}...`));
      let toolResult;
      try {
        // Try to log the client's callTool method
        console.log(chalk.cyan(`Client callTool type: ${typeof client.callTool}`));

        // Call the tool with proper error handling
        toolResult = await client.callTool({
          name: toolName,
          arguments: args,
        });
      } catch (e) {
        console.error(chalk.red(`Error during callTool execution: ${e.message}`));
        console.error(chalk.red(`Error stack: ${e.stack}`));

        // Try to get more info about the error
        if (e.cause) {
          console.error(chalk.red(`Error cause: ${JSON.stringify(e.cause)}`));
        }

        // Throw a more descriptive error
        throw new Error(`Error calling MCP tool ${toolName}: ${e.message}`);
      }

      console.log(chalk.green(`Tool call to ${toolName} successful!`));
      // Format the response based on provider
      if (provider === 'openai') {
        toolResponses.push({
          role: 'tool',
          content: JSON.stringify(toolResult),
          tool_call_id: toolCall.id
        });
      } else if (provider === 'anthropic') {
        toolResponses.push({
          role: 'tool',
          content: JSON.stringify(toolResult),
          tool_call_id: toolCall.id
        });
      } else if (provider === 'ollama') {
        toolResponses.push({
          role: 'tool',
          content: JSON.stringify(toolResult),
          tool_call_id: toolCall.id
        });
      }

      console.log(chalk.gray(`Tool response: ${JSON.stringify(toolResult)}`));
    } catch (error) {
      console.error(chalk.red(`Error calling tool: ${(error as Error).message}`));

      console.error(chalk.red('Tool call details:'), JSON.stringify({
        serverName,
        toolName,
        args
      }, null, 2));

      if (typeof error === 'object' && error !== null) {
        console.error(chalk.red('Detailed error:'), JSON.stringify(error, null, 2));
      }

      // Add error message as tool response
      toolResponses.push({
        role: 'tool',
        content: `Error: ${(error as Error).message}`,
        tool_call_id: toolCall.id
      });
    }
  }

  return toolResponses;
}

// Convert messages to provider-specific format
export function formatMessagesForProvider(messages: Message[], provider: ProviderType): any {
  if (provider === 'openai') {
    console.log(chalk.cyan(`Formatting ${messages.length} messages for OpenAI provider`));

    return messages.map(msg => {
      const formattedMsg: any = {
        role: msg.role,
        content: msg.role !== 'assistant' || !msg.tool_calls ? msg.content : null
      };

      if (msg.tool_calls) {
        // Log tool calls information
        console.log(chalk.cyan(`Message has ${msg.tool_calls.length} tool calls`));
        for (const tc of msg.tool_calls) {
          console.log(chalk.cyan(`Tool call: ${tc.function?.name || 'unnamed'}, args: ${tc.function?.arguments || '{}'}`));
        }

        formattedMsg.tool_calls = msg.tool_calls;
      }

      if (msg.tool_call_id) {
        console.log(chalk.cyan(`Message has tool_call_id: ${msg.tool_call_id}`));
        formattedMsg.tool_call_id = msg.tool_call_id;
      }

      return formattedMsg;
    });
  } else if (provider === 'anthropic') {
    return messages.map(msg => {
      if (msg.role === 'system') {
        return {
          role: 'system',
          content: msg.content
        };
      } else if (msg.role === 'user' || (msg.role === 'assistant' && !msg.tool_calls)) {
        return {
          role: msg.role,
          content: [{ type: 'text', text: msg.content }]
        };
      } else if (msg.role === 'assistant' && msg.tool_calls) {
        return {
          role: 'assistant',
          content: msg.tool_calls.map(tc => ({
            type: 'tool_use',
            id: tc.id,
            name: tc.function.name,
            input: JSON.parse(tc.function.arguments)
          }))
        };
      } else if (msg.role === 'tool') {
        return {
          role: 'tool',
          content: [{
            type: 'tool_result',
            tool_use_id: msg.tool_call_id,
            content: [{ type: 'text', text: msg.content }]
          }]
        };
      }

      return {
        role: msg.role,
        content: [{ type: 'text', text: msg.content }]
      };
    });
  } else if (provider === 'ollama') {
    return messages.map(msg => {
      const formattedMsg: any = {
        role: msg.role,
        content: msg.content
      };

      if (msg.tool_calls) {
        formattedMsg.tool_calls = msg.tool_calls.map(tc => ({
          function: {
            name: tc.function.name,
            arguments: JSON.parse(tc.function.arguments)
          }
        }));
      }

      return formattedMsg;
    });
  }

  return messages;
}

// Extract content and tool calls from LLM response
export function processLLMResponse(
  response: any,
  provider: ProviderType,
  conversationId?: string
): { content: string, toolCalls: any[] | null } {
  // Use random ID for logging if conversationId not provided
  const logId = conversationId || Math.random().toString(36).substring(2, 10);

  try {
    if (provider === 'openai') {
      // Validate the response structure
      if (!response || !response.choices || !Array.isArray(response.choices) || response.choices.length === 0) {
        console.error(`[${logId}] Invalid OpenAI response structure:`, JSON.stringify(response));
        return { content: '', toolCalls: null };
      }

      const responseObj = response as OpenAIResponse;
      const message = responseObj.choices[0].message;

      // Safety check for message
      if (!message) {
        console.error(`[${logId}] OpenAI response missing message:`, JSON.stringify(response.choices[0]));
        return { content: '', toolCalls: null };
      }

      // Process tool calls if present
      const toolCalls = message.tool_calls || null;

      // Validate tool calls if present
      if (toolCalls && Array.isArray(toolCalls)) {
        // Log tool calls for debugging
        console.log(`[${logId}] Processing ${toolCalls.length} tool calls from OpenAI response`);

        // Validate each tool call
        toolCalls.forEach((tc, idx) => {
          if (!tc.function) {
            console.warn(`[${logId}] Tool call ${idx} missing function property`);
          } else if (!tc.function.name) {
            console.warn(`[${logId}] Tool call ${idx} missing function name`);
          }

          // Try to parse arguments to validate them
          if (tc.function && typeof tc.function.arguments === 'string') {
            try {
              JSON.parse(tc.function.arguments);
            } catch (e) {
              console.warn(`[${logId}] Tool call ${idx} has invalid JSON arguments: ${e.message}`);
              console.warn(`[${logId}] Arguments string: ${tc.function.arguments}`);
            }
          }
        });
      }

      return {
        content: message.content || '',
        toolCalls: toolCalls
      };
    } else if (provider === 'anthropic') {
      // Validate Anthropic response structure
      if (!response || !response.content || !Array.isArray(response.content)) {
        console.error(`[${logId}] Invalid Anthropic response structure:`, JSON.stringify(response));
        return { content: '', toolCalls: null };
      }

      const responseObj = response as AnthropicResponse;
      const textContent = responseObj.content
        .filter(item => item.type === 'text')
        .map(item => item.text)
        .join('');

      // Process tool calls if present
      const toolCalls = responseObj.tool_calls || null;

      // Validate tool calls
      if (toolCalls && Array.isArray(toolCalls)) {
        console.log(`[${logId}] Processing ${toolCalls.length} tool calls from Anthropic response`);

        // Validate each tool call
        toolCalls.forEach((tc, idx) => {
          if (!tc.name) {
            console.warn(`[${logId}] Anthropic tool call ${idx} missing name property`);
          }

          // Validate input object
          if (!tc.input) {
            console.warn(`[${logId}] Anthropic tool call ${idx} missing input property`);
          }
        });
      }

      return {
        content: textContent,
        toolCalls: toolCalls
      };
    } else if (provider === 'ollama') {
      // Validate Ollama response structure
      if (!response || !response.message) {
        console.error(`[${logId}] Invalid Ollama response structure:`, JSON.stringify(response));
        return { content: '', toolCalls: null };
      }

      const responseObj = response as OllamaResponse;

      // Process tool calls if present
      const toolCalls = responseObj.message.tool_calls || null;

      // Validate tool calls
      if (toolCalls && Array.isArray(toolCalls)) {
        console.log(`[${logId}] Processing ${toolCalls.length} tool calls from Ollama response`);
      }

      return {
        content: responseObj.message.content,
        toolCalls: toolCalls
      };
    }

    console.warn(`[${logId}] Unknown provider: ${provider}`);
    return { content: '', toolCalls: null };
  } catch (error) {
    console.error(`[${logId}] Error processing LLM response:`, error);
    console.error(`[${logId}] Response that caused the error:`, JSON.stringify(response));
    return { content: '', toolCalls: null };
  }
}

