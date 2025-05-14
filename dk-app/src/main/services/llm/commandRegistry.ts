import { z } from 'zod'
import { SlashCommandMeta, CommandRegistry } from '@shared/commandTypes'
import logger from '@shared/logging'
import { LLMService } from './llmService'

// Import LLM types needed for chat completion requests
import * as LLMTypes from '@shared/llmTypes'
import { LLMProvider } from './types'
type ChatCompletionRequest = LLMTypes.ChatCompletionRequest
type ChatCompletionResponse = LLMTypes.ChatCompletionResponse

// Create the command registry
class SlashCommandRegistryImpl implements CommandRegistry {
  private commands = new Map<string, SlashCommandMeta>()

  register(command: SlashCommandMeta): void {
    if (this.commands.has(command.name)) {
      logger.debug(`Command ${command.name} already exists, overwriting`)
    }
    this.commands.set(command.name, command)
    logger.debug(`Registered slash command: ${command.name}`)
  }

  get(name: string): SlashCommandMeta | undefined {
    return this.commands.get(name)
  }

  getAll(): SlashCommandMeta[] {
    return Array.from(this.commands.values())
  }
}

// Singleton instance
export const commandRegistry = new SlashCommandRegistryImpl()

// Create a singleton LLM service instance for the commands to use
let llmServiceInstance: LLMService | null = null

// Function to get or create the LLM service instance
async function getLLMService(): Promise<LLMService> {
  if (!llmServiceInstance) {
    try {
      // First try to get the existing llmService instance from the main process
      try {
        logger.debug('Attempting to get existing LLMService from main process')
        // Import the LLMService class from llm module
        const llmModule = await import('./llmService')

        // Check if we already have an instance created elsewhere
        try {
          const services = await import('../index')
          // Check if LLMService is already initialized elsewhere
          if (services.LLMService) {
            logger.debug('Found LLMService class from main process')
          }
        } catch (error) {
          logger.debug(
            'Could not check for LLMService in services',
            error instanceof Error ? error.message : String(error)
          )
        }
        // Create a new LLMService instance
        logger.debug('Creating new LLMService instance')
        llmServiceInstance = new llmModule.LLMService()
        logger.debug('LLMService instance created')
      } catch (importError) {
        logger.debug(
          'Could not import main llmService:',
          importError instanceof Error ? importError.message : String(importError)
        )
        // Continue to fallback approach
      }

      // If that fails, create a new instance as fallback
      logger.debug('Creating new LLMService instance as fallback')
      llmServiceInstance = new LLMService()

      // Get the proper LLM configuration from the app config
      try {
        // Import the config service, using a different path
        const { getLLMConfig } = await import('../config')
        const llmConfig = getLLMConfig()
        logger.debug(`Using configuration with active provider: ${llmConfig.activeProvider}`)

        // Initialize providers from config
        Object.entries(llmConfig.providers).forEach(([providerName, providerConfig]) => {
          if (providerConfig && (providerName === 'ollama' || providerConfig.apiKey)) {
            logger.debug(`Initializing provider ${providerName}`)
            llmServiceInstance!.initProvider(providerName as LLMProvider, providerConfig)
          }
        })

        // Set the active provider from config
        if (llmConfig.activeProvider) {
          logger.debug(`Setting active provider to: ${llmConfig.activeProvider}`)
          llmServiceInstance!.setActiveProvider(llmConfig.activeProvider as LLMProvider)
        }
      } catch (configError) {
        logger.debug(
          `Could not load LLM config, using default provider: ${configError instanceof Error ? configError.message : String(configError)}`
        )
        // As a last resort, use OpenAI as it's more commonly configured
        llmServiceInstance.setActiveProvider(LLMProvider.OPENAI)
      }

      logger.debug('Successfully created and configured fallback LLMService instance')
    } catch (error) {
      logger.error(
        'Error creating LLMService instance:',
        error instanceof Error ? error.message : String(error)
      )
      throw new Error(
        `Failed to initialize LLM service: ${error instanceof Error ? error.message : String(error)}`
      )
    }
  }

  if (!llmServiceInstance) {
    logger.error('llmServiceInstance is still null after creation attempts')
    throw new Error('Failed to initialize LLM service')
  }

  return llmServiceInstance
}

// Register the core commands

// Clear command to clear the chat history
commandRegistry.register({
  name: 'clear',
  summary: 'Clear the chat history',
  handler: async (_, ctx) => {
    // The actual clearing will be handled by the client
    // We just return a message to be shown
    return 'Chat history cleared.'
  }
})

// Example command with parameters using zod
commandRegistry.register({
  name: 'echo',
  summary: 'Echo a message back',
  paramsSchema: z.string().min(1, 'Please provide a message to echo'),
  handler: async (params, ctx) => {
    return `Echo: ${params}`
  }
})

// RAG command to search RAG documents and use them to answer the query with LLM
commandRegistry.register({
  name: 'rag',
  summary: 'Search documents and answer the query with AI',
  paramsSchema: z.string().min(1, 'Please provide text to search for'),
  handler: async (params, ctx) => {
    try {
      logger.debug('Executing /rag command with params:', params)

      // Try to import documentService with different methods
      logger.debug('Attempting to import documentService...')
      let documentResults = []
      let hasFoundDocuments = false

      try {
        // Try direct import first
        const documentServiceModule = await import('../documentService')
        logger.debug('Successfully imported documentService via relative path')

        // Search the RAG server with the input text as query, limit to 5 results
        logger.debug('Calling documentService.searchDocuments with params:', params)
        const searchResults = await documentServiceModule.documentService.searchDocuments(
          String(params),
          5
        )
        logger.debug('Got search results:', JSON.stringify(searchResults))

        if (searchResults && searchResults.documents && searchResults.documents.length > 0) {
          logger.debug(`Found ${searchResults.documents.length} documents`)
          hasFoundDocuments = true

          // Create a clean list of document results for display
          documentResults = searchResults.documents.map((doc, index) => {
            return {
              id: index + 1,
              title: doc.file || `Document ${index + 1}`,
              content: doc.content || 'No content available'
            }
          })

          // Create a response with the document buttons for display
          let displayResponse = `Analyzing documents to answer: "${params}"\n\n[DOCUMENT_RESULTS_START]${JSON.stringify(documentResults)}[DOCUMENT_RESULTS_END]`

          // Create the context to send to the LLM
          let documentContext = ''
          documentResults.forEach((doc, index) => {
            documentContext += `DOCUMENT ${index + 1}: ${doc.title}\n${doc.content}\n\n`
          })

          try {
            // Get the LLM service
            logger.debug('Attempting to get LLM service')
            const llmService = await getLLMService()
            logger.debug('Successfully retrieved LLM service')

            // Verify the service has the necessary methods
            if (!llmService || typeof llmService.sendMessage !== 'function') {
              logger.error('LLM service missing sendMessage method', { service: !!llmService })
              throw new Error('LLM service is not properly initialized')
            }

            // Check the active provider and make sure API key is configured
            const activeProvider = llmService.getActiveProvider()
            logger.debug(`Active LLM provider is: ${activeProvider}`)

            // Get available providers to make sure we can use the active one
            const availableProviders = llmService.getAvailableProviders()
            if (!availableProviders.includes(activeProvider)) {
              logger.error(`Active provider ${activeProvider} is not available`, {
                active: activeProvider,
                available: availableProviders
              })
              throw new Error(
                `The active LLM provider (${activeProvider}) is not properly configured. Please check your API key settings.`
              )
            }

            // Prepare the messages with document context
            const messages = [
              {
                role: 'system',
                content:
                  'You are a helpful assistant that answers user questions based on provided document context.'
              },
              {
                role: 'user',
                content: `I have the following documents:\n\n${documentContext}\n\nBased on these documents, please answer the following question: ${params}`
              }
            ]

            // MAJOR CHANGE: Instead of returning data for client-side LLM request,
            // directly make the LLM request on the server side
            logger.debug(`Making direct LLM request with active provider: ${activeProvider}`)

            const request: ChatCompletionRequest = {
              messages: messages.map((msg) => ({
                role: msg.role as 'system' | 'user' | 'assistant' | 'tool',
                content: msg.content
              }))
            }
            logger.debug('Calling llmService.sendMessage')
            const result = await llmService.sendMessage(request)
            logger.debug('LLM request completed')

            logger.debug('Examining LLM response format:', JSON.stringify(result))

            // Handle different response formats from various providers
            let responseContent = ''

            // Handle OpenAI format (has choices[0].message.content)
            // Cast result to any since we need to handle multiple response formats
            const anyResult = result as any
            if (anyResult && anyResult.choices && anyResult.choices[0]?.message?.content) {
              logger.debug('Processing OpenAI-style response format')
              responseContent = anyResult.choices[0].message.content
            }
            // Handle Anthropic format (has content property)
            else if (
              anyResult &&
              anyResult.message &&
              typeof anyResult.message.content === 'string'
            ) {
              logger.debug('Processing Anthropic-style response format')
              responseContent = anyResult.message.content
            }
            // Handle Ollama format (has response property)
            else if (anyResult && typeof anyResult.response === 'string') {
              logger.debug('Processing Ollama-style response format')
              responseContent = anyResult.response
            }
            // Handle generic case - try to find content anywhere in the response
            else if (anyResult) {
              logger.debug('Processing unknown response format, looking for content')
              // Try to extract content from any property that looks like a message or content
              if (anyResult.content) {
                responseContent =
                  typeof anyResult.content === 'string'
                    ? anyResult.content
                    : JSON.stringify(anyResult.content)
              } else if (anyResult.text) {
                responseContent = anyResult.text
              } else if (anyResult.message) {
                responseContent =
                  typeof anyResult.message === 'string'
                    ? anyResult.message
                    : anyResult.message.content || JSON.stringify(anyResult.message)
              } else {
                // As a last resort, just stringify the result
                logger.debug('Could not find content property, stringifying entire result')
                try {
                  // Try to extract just the interesting parts
                  const { id, created, model, ...rest } = result
                  responseContent = JSON.stringify(rest)
                } catch (e) {
                  responseContent =
                    "Received a response from the AI service but couldn't format it properly."
                }
              }
            } else {
              logger.error('No usable response from LLM')
              return `I couldn't analyze the documents properly. Please try again.\n\n[DOCUMENT_RESULTS_START]${JSON.stringify(documentResults)}[DOCUMENT_RESULTS_END]`
            }

            // If we got a response, return it with the document buttons
            if (responseContent) {
              logger.debug('Successfully extracted content from LLM response')
              const response = `${responseContent}\n\n[DOCUMENT_RESULTS_START]${JSON.stringify(documentResults)}[DOCUMENT_RESULTS_END]`
              return response
            } else {
              logger.error('Failed to extract content from LLM response:', result)
              return `I couldn't analyze the documents properly. Please try again.\n\n[DOCUMENT_RESULTS_START]${JSON.stringify(documentResults)}[DOCUMENT_RESULTS_END]`
            }
          } catch (llmError) {
            logger.error('Error preparing LLM request:', llmError)
            logger.error('LLM error details:', {
              message: llmError instanceof Error ? llmError.message : String(llmError),
              stack: llmError instanceof Error ? llmError.stack : 'No stack trace'
            })

            // Still return the document results with an error message
            return `Analyzing documents for query: "${params}"\n\nI found some relevant documents but couldn't generate an AI response. ${llmError instanceof Error ? llmError.message : String(llmError) || 'The AI service is unavailable.'}\n\n[DOCUMENT_RESULTS_START]${JSON.stringify(documentResults)}[DOCUMENT_RESULTS_END]\n\n(Try again later when the AI service is available. You can still view the documents by clicking the icons above.)`
          }
        } else {
          logger.debug('No documents found in search results')
          return `No relevant documents found for: "${params}". Please try a different query.`
        }
      } catch (importError) {
        logger.error('Error importing documentService via relative path:', importError)

        // Try alternative import approaches
        try {
          // Try importing from root services
          const { documentService } = await import('@services/documentService')
          logger.debug('Successfully imported documentService via @services alias')

          // Search and format results
          const searchResults = await documentService.searchDocuments(String(params), 5)

          if (searchResults && searchResults.documents && searchResults.documents.length > 0) {
            hasFoundDocuments = true
            logger.debug(`Found ${searchResults.documents.length} documents via alias path`)

            // Create a clean list of document results for display
            documentResults = searchResults.documents.map((doc, index) => {
              return {
                id: index + 1,
                title: doc.file || `Document ${index + 1}`,
                content: doc.content || 'No content available'
              }
            })

            // Create a response with the document buttons for display
            let displayResponse = `Analyzing documents to answer: "${params}"\n\n[DOCUMENT_RESULTS_START]${JSON.stringify(documentResults)}[DOCUMENT_RESULTS_END]`

            // Create the context to send to the LLM
            let documentContext = ''
            documentResults.forEach((doc, index) => {
              documentContext += `DOCUMENT ${index + 1}: ${doc.title}\n${doc.content}\n\n`
            })

            try {
              // Get the LLM service
              logger.debug('Attempting to get LLM service (alias path)')
              const llmService = await getLLMService()
              logger.debug('Successfully retrieved LLM service (alias path)')

              // Verify the service has the necessary methods
              if (!llmService || typeof llmService.sendMessage !== 'function') {
                logger.error('LLM service missing sendMessage method (alias path)', {
                  service: !!llmService
                })
                throw new Error('LLM service is not properly initialized')
              }

              // Check the active provider and make sure API key is configured
              const activeProvider = llmService.getActiveProvider()
              logger.debug(`Active LLM provider is (alias path): ${activeProvider}`)

              // Get available providers to make sure we can use the active one
              const availableProviders = llmService.getAvailableProviders()
              if (!availableProviders.includes(activeProvider)) {
                logger.error(`Active provider ${activeProvider} is not available (alias path)`, {
                  active: activeProvider,
                  available: availableProviders
                })
                throw new Error(
                  `The active LLM provider (${activeProvider}) is not properly configured. Please check your API key settings.`
                )
              }

              // Prepare the messages with document context
              const messages = [
                {
                  role: 'system',
                  content:
                    'You are a helpful assistant that answers user questions based on provided document context.'
                },
                {
                  role: 'user',
                  content: `I have the following documents:\n\n${documentContext}\n\nBased on these documents, please answer the following question: ${params}`
                }
              ]

              // MAJOR CHANGE: Instead of returning data for client-side LLM request,
              // directly make the LLM request on the server side
              logger.debug(
                `Making direct LLM request with active provider: ${activeProvider} (alias path)`
              )

              const request: ChatCompletionRequest = {
                messages: messages.map((msg) => ({
                  role: msg.role as 'system' | 'user' | 'assistant' | 'tool',
                  content: msg.content
                }))
              }
              logger.debug('Calling llmService.sendMessage (alias path)')
              const result = await llmService.sendMessage(request)
              logger.debug('LLM request completed (alias path)')

              logger.debug('Examining LLM response format (alias path):', JSON.stringify(result))

              // Handle different response formats from various providers
              let responseContent = ''

              // Handle OpenAI format (has choices[0].message.content)
              // Cast result to any since we need to handle multiple response formats
              const anyResult = result as any
              if (anyResult && anyResult.choices && anyResult.choices[0]?.message?.content) {
                logger.debug('Processing OpenAI-style response format (alias path)')
                responseContent = anyResult.choices[0].message.content
              }
              // Handle Anthropic format (has content property)
              else if (
                anyResult &&
                anyResult.message &&
                typeof anyResult.message.content === 'string'
              ) {
                logger.debug('Processing Anthropic-style response format (alias path)')
                responseContent = anyResult.message.content
              }
              // Handle Ollama format (has response property)
              else if (anyResult && typeof anyResult.response === 'string') {
                logger.debug('Processing Ollama-style response format (alias path)')
                responseContent = anyResult.response
              }
              // Handle generic case - try to find content anywhere in the response
              else if (anyResult) {
                logger.debug('Processing unknown response format, looking for content (alias path)')
                // Try to extract content from any property that looks like a message or content
                if (anyResult.content) {
                  responseContent =
                    typeof anyResult.content === 'string'
                      ? anyResult.content
                      : JSON.stringify(anyResult.content)
                } else if (anyResult.text) {
                  responseContent = anyResult.text
                } else if (anyResult.message) {
                  responseContent =
                    typeof anyResult.message === 'string'
                      ? anyResult.message
                      : anyResult.message.content || JSON.stringify(anyResult.message)
                } else {
                  // As a last resort, just stringify the result
                  logger.debug(
                    'Could not find content property, stringifying entire result (alias path)'
                  )
                  try {
                    // Try to extract just the interesting parts
                    const { id, created, model, ...rest } = result
                    responseContent = JSON.stringify(rest)
                  } catch (e) {
                    responseContent =
                      "Received a response from the AI service but couldn't format it properly."
                  }
                }
              } else {
                logger.error('No usable response from LLM (alias path)')
                return `I couldn't analyze the documents properly. Please try again.\n\n[DOCUMENT_RESULTS_START]${JSON.stringify(documentResults)}[DOCUMENT_RESULTS_END]`
              }

              // If we got a response, return it with the document buttons
              if (responseContent) {
                logger.debug('Successfully extracted content from LLM response (alias path)')
                const response = `${responseContent}\n\n[DOCUMENT_RESULTS_START]${JSON.stringify(documentResults)}[DOCUMENT_RESULTS_END]`
                return response
              } else {
                logger.error('Failed to extract content from LLM response (alias path):', result)
                return `I couldn't analyze the documents properly. Please try again.\n\n[DOCUMENT_RESULTS_START]${JSON.stringify(documentResults)}[DOCUMENT_RESULTS_END]`
              }
            } catch (llmError) {
              logger.error('Error preparing LLM request (alias path):', llmError)
              logger.error('LLM error details (alias path):', {
                message: llmError instanceof Error ? llmError.message : String(llmError),
                stack: llmError instanceof Error ? llmError.stack : 'No stack trace'
              })

              // Still return the document results with an error message
              return `Analyzing documents for query: "${params}"\n\nI found some relevant documents but couldn't generate an AI response. ${llmError instanceof Error ? llmError.message : String(llmError) || 'The AI service is unavailable.'}\n\n[DOCUMENT_RESULTS_START]${JSON.stringify(documentResults)}[DOCUMENT_RESULTS_END]\n\n(Try again later when the AI service is available. You can still view the documents by clicking the icons above.)`
            }
          } else {
            logger.debug('No documents found in search results (alias path)')
            return `No relevant documents found for: "${params}". Please try a different query.`
          }
        } catch (aliasImportError) {
          logger.error('Error importing documentService via alias:', aliasImportError)
          throw aliasImportError
        }
      }
    } catch (error) {
      logger.error(
        'Error executing rag command:',
        error instanceof Error ? error.message : String(error)
      )
      return `Error processing query: "${params}"\n\n(${error instanceof Error ? error.message : String(error) || 'Unknown error'})`
    }
  }
})
