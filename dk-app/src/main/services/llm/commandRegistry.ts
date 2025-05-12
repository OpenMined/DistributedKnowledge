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
      logger.warn({ cmd: command.name }, 'Command already exists, overwriting')
    }
    this.commands.set(command.name, command)
    logger.info({ cmd: command.name }, 'Registered slash command')
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
let llmServiceInstance: LLMService | null = null;

// Function to get or create the LLM service instance
async function getLLMService(): Promise<LLMService> {
  if (!llmServiceInstance) {
    try {
      // Create a new instance directly
      logger.info('Creating new LLMService instance directly')
      
      llmServiceInstance = new LLMService();
      
      // Initialize with default provider (Anthropic) and some basic settings
      // This is a minimal config just to make the service operational
      llmServiceInstance.setActiveProvider(LLMProvider.ANTHROPIC);
      
      logger.info('Successfully created and configured LLMService instance')
      
      // Try to get the actual llmService instance from the main process if available
      try {
        const { llmService } = await import('../index');
        if (llmService) {
          logger.info('Found existing llmService from main process, using it instead');
          llmServiceInstance = llmService;
        }
      } catch (importError) {
        logger.info('Could not import main llmService, using local instance instead:', importError.message);
        // Continue using our local instance
      }
    } catch (error) {
      logger.error('Error creating LLMService instance:', error);
      throw new Error(`Failed to initialize LLM service: ${error.message}`);
    }
  }
  
  if (!llmServiceInstance) {
    logger.error('llmServiceInstance is still null after creation attempts');
    throw new Error('Failed to initialize LLM service');
  }
  
  return llmServiceInstance;
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

// Answer command to search RAG documents and use them to answer the query with LLM
commandRegistry.register({
  name: 'answer',
  summary: 'Search documents and answer the query with AI',
  paramsSchema: z.string().min(1, 'Please provide text to search for'),
  handler: async (params, ctx) => {
    try {
      logger.info('Executing /answer command with params:', params)

      // Try to import documentService with different methods
      logger.info('Attempting to import documentService...')
      let documentResults = [];
      let hasFoundDocuments = false;

      try {
        // Try direct import first
        const documentServiceModule = await import('../documentService')
        logger.info('Successfully imported documentService via relative path')

        // Search the RAG server with the input text as query, limit to 5 results
        logger.info('Calling documentService.searchDocuments with params:', params)
        const searchResults = await documentServiceModule.documentService.searchDocuments(params, 5)
        logger.info('Got search results:', JSON.stringify(searchResults))

        if (searchResults && searchResults.documents && searchResults.documents.length > 0) {
          logger.info(`Found ${searchResults.documents.length} documents`)
          hasFoundDocuments = true;

          // Create a clean list of document results for display
          documentResults = searchResults.documents.map((doc, index) => {
            return {
              id: index + 1,
              title: doc.title || doc.filename || `Document ${index + 1}`,
              content: doc.content || 'No content available'
            };
          });

          // Create a response with the document buttons for display
          let displayResponse = `Analyzing documents to answer: "${params}"\n\n[DOCUMENT_RESULTS_START]${JSON.stringify(documentResults)}[DOCUMENT_RESULTS_END]`;

          // Create the context to send to the LLM
          let documentContext = "";
          documentResults.forEach((doc, index) => {
            documentContext += `DOCUMENT ${index + 1}: ${doc.title}\n${doc.content}\n\n`;
          });

          try {
            // Get the LLM service
            logger.info('Attempting to get LLM service');
            const llmService = await getLLMService();
            logger.info('Successfully retrieved LLM service');

            // Verify the service has the necessary methods
            if (!llmService || typeof llmService.sendMessage !== 'function') {
              logger.error('LLM service missing sendMessage method', { service: !!llmService });
              throw new Error('LLM service is not properly initialized');
            }

            // Prepare the messages with document context
            const messages = [
              {
                role: 'system',
                content: 'You are a helpful assistant that answers user questions based on provided document context.'
              },
              {
                role: 'user',
                content: `I have the following documents:\n\n${documentContext}\n\nBased on these documents, please answer the following question: ${params}`
              }
            ];

            // MAJOR CHANGE: Instead of returning data for client-side LLM request,
            // directly make the LLM request on the server side
            logger.info('Making direct LLM request from server with messages');

            const request = { messages: messages };
            logger.info('Calling llmService.sendMessage');
            const result = await llmService.sendMessage(request);
            logger.info('LLM request completed');

            if (result && result.choices && result.choices[0]?.message?.content) {
              logger.info('Got LLM response from direct request');
              // Return the LLM response with the document buttons
              const response = `${result.choices[0].message.content}\n\n[DOCUMENT_RESULTS_START]${JSON.stringify(documentResults)}[DOCUMENT_RESULTS_END]`;
              return response;
            } else {
              logger.error('Invalid response from LLM:', result);
              return `I couldn't analyze the documents properly. Please try again.\n\n[DOCUMENT_RESULTS_START]${JSON.stringify(documentResults)}[DOCUMENT_RESULTS_END]`;
            }
          } catch (llmError) {
            logger.error('Error preparing LLM request:', llmError);
            logger.error('LLM error details:', { message: llmError.message, stack: llmError.stack });
            
            // Still return the document results with an error message
            return `Analyzing documents for query: "${params}"\n\nI found some relevant documents but couldn't generate an AI response. ${llmError.message || 'The AI service is unavailable.'}\n\n[DOCUMENT_RESULTS_START]${JSON.stringify(documentResults)}[DOCUMENT_RESULTS_END]\n\n(Try again later when the AI service is available. You can still view the documents by clicking the icons above.)`;
          }
        } else {
          logger.info('No documents found in search results')
          return `No relevant documents found for: "${params}". Please try a different query.`;
        }

      } catch (importError) {
        logger.error('Error importing documentService via relative path:', importError)

        // Try alternative import approaches
        try {
          // Try importing from root services
          const { documentService } = await import('@services/documentService')
          logger.info('Successfully imported documentService via @services alias')

          // Search and format results
          const searchResults = await documentService.searchDocuments(params, 5)

          if (searchResults && searchResults.documents && searchResults.documents.length > 0) {
            hasFoundDocuments = true;
            logger.info(`Found ${searchResults.documents.length} documents via alias path`)

            // Create a clean list of document results for display
            documentResults = searchResults.documents.map((doc, index) => {
              return {
                id: index + 1,
                title: doc.title || doc.filename || `Document ${index + 1}`,
                content: doc.content || 'No content available'
              };
            });

            // Create a response with the document buttons for display
            let displayResponse = `Analyzing documents to answer: "${params}"\n\n[DOCUMENT_RESULTS_START]${JSON.stringify(documentResults)}[DOCUMENT_RESULTS_END]`;

            // Create the context to send to the LLM
            let documentContext = "";
            documentResults.forEach((doc, index) => {
              documentContext += `DOCUMENT ${index + 1}: ${doc.title}\n${doc.content}\n\n`;
            });

            try {
              // Get the LLM service
              logger.info('Attempting to get LLM service (alias path)');
              const llmService = await getLLMService();
              logger.info('Successfully retrieved LLM service (alias path)');

              // Verify the service has the necessary methods
              if (!llmService || typeof llmService.sendMessage !== 'function') {
                logger.error('LLM service missing sendMessage method (alias path)', { service: !!llmService });
                throw new Error('LLM service is not properly initialized');
              }

              // Prepare the messages with document context
              const messages = [
                {
                  role: 'system',
                  content: 'You are a helpful assistant that answers user questions based on provided document context.'
                },
                {
                  role: 'user',
                  content: `I have the following documents:\n\n${documentContext}\n\nBased on these documents, please answer the following question: ${params}`
                }
              ];

              // MAJOR CHANGE: Instead of returning data for client-side LLM request,
              // directly make the LLM request on the server side
              logger.info('Making direct LLM request from server with messages (alias path)');

              const request = { messages: messages };
              logger.info('Calling llmService.sendMessage (alias path)');
              const result = await llmService.sendMessage(request);
              logger.info('LLM request completed (alias path)');

              if (result && result.choices && result.choices[0]?.message?.content) {
                logger.info('Got LLM response from direct request (alias path)');
                // Return the LLM response with the document buttons
                const response = `${result.choices[0].message.content}\n\n[DOCUMENT_RESULTS_START]${JSON.stringify(documentResults)}[DOCUMENT_RESULTS_END]`;
                return response;
              } else {
                logger.error('Invalid response from LLM (alias path):', result);
                return `I couldn't analyze the documents properly. Please try again.\n\n[DOCUMENT_RESULTS_START]${JSON.stringify(documentResults)}[DOCUMENT_RESULTS_END]`;
              }
            } catch (llmError) {
              logger.error('Error preparing LLM request (alias path):', llmError);
              logger.error('LLM error details (alias path):', { message: llmError.message, stack: llmError.stack });
              
              // Still return the document results with an error message
              return `Analyzing documents for query: "${params}"\n\nI found some relevant documents but couldn't generate an AI response. ${llmError.message || 'The AI service is unavailable.'}\n\n[DOCUMENT_RESULTS_START]${JSON.stringify(documentResults)}[DOCUMENT_RESULTS_END]\n\n(Try again later when the AI service is available. You can still view the documents by clicking the icons above.)`;
            }
          } else {
            logger.info('No documents found in search results (alias path)')
            return `No relevant documents found for: "${params}". Please try a different query.`;
          }
        } catch (aliasImportError) {
          logger.error('Error importing documentService via alias:', aliasImportError)
          throw aliasImportError
        }
      }
    } catch (error) {
      logger.error('Error executing answer command:', error)
      return `Error processing query: "${params}"\n\n(${error.message || 'Unknown error'})`
    }
  }
})