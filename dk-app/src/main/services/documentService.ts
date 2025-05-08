import { createServiceLogger } from '../../shared/logging'
import axios from 'axios'
import { appConfig } from './config'
import { RAGDocument } from '../../shared/types'

// Create a dedicated logger for the document service
const serviceLogger = createServiceLogger('documentService')

export interface DocumentStats {
  count: number
  error?: string
}

export interface DocumentSearchResults {
  documents: RAGDocument[]
}

/**
 * Service to periodically fetch document data from the RAG server
 * and maintain a local cache for quick retrieval
 */
export class DocumentService {
  private intervalId: NodeJS.Timeout | null = null
  private intervalMs = 5000 // 5 seconds polling interval
  private isServerAvailable = false
  private ragServerBaseUrl: string = ''

  // Cache for document data
  private documentStats: DocumentStats = { count: 0 }
  private lastSearchResults: Map<string, DocumentSearchResults> = new Map()
  private lastRefreshTime: Date = new Date()

  constructor() {
    // Initialize with empty values
    this.documentStats = { count: 0 }
    serviceLogger.info('Document service initialized')
  }

  /**
   * Start the periodic document data fetching service
   */
  public startDocumentDataFetch(): void {
    // Clear any existing interval
    if (this.intervalId) {
      clearInterval(this.intervalId)
    }

    // Update the RAG server URL
    this.updateRagServerUrl()

    serviceLogger.info(`Starting document data fetch service with ${this.intervalMs}ms interval`)

    // Perform an initial fetch immediately
    this.fetchDocumentData().catch((error) => {
      serviceLogger.error('Initial document data fetch failed:', error)
    })

    // Set up periodic fetching
    this.intervalId = setInterval(() => {
      this.fetchDocumentData().catch((error) => {
        serviceLogger.error('Periodic document data fetch failed:', error)
      })
    }, this.intervalMs)

    serviceLogger.info('Document data fetch service started')
  }

  /**
   * Stop the periodic fetching service
   */
  public stopDocumentDataFetch(): void {
    if (this.intervalId) {
      clearInterval(this.intervalId)
      this.intervalId = null
      serviceLogger.info('Document data fetch service stopped')
    }
  }

  /**
   * Update the RAG server URL from application config
   */
  private updateRagServerUrl(): void {
    // Check for dk_api first as it's the primary endpoint used in the original implementation
    if (appConfig.dk_api) {
      this.ragServerBaseUrl = appConfig.dk_api
    } else {
      // Fallback to a default URL
      // TS error fix: Add rag_server_url to AppConfig type or use a type assertion
      const ragServerUrl = (appConfig as any).rag_server_url
      this.ragServerBaseUrl =
        (ragServerUrl ? ragServerUrl.replace(/\/rag$/, '') : '') || 'http://localhost:4232'
    }
    serviceLogger.info(`RAG server URL set to: ${this.ragServerBaseUrl}`)
  }

  /**
   * Fetch document data from the RAG server
   */
  private async fetchDocumentData(): Promise<void> {
    try {
      // Update the URL in case it changed
      this.updateRagServerUrl()

      // First, check if the RAG server is available
      await this.checkServerAvailability()

      // If server is not available, just return and maintain existing cache
      if (!this.isServerAvailable) {
        this.documentStats = {
          count: this.documentStats.count || 0,
          error: 'RAG server is not available'
        }
        return
      }

      // Fetch current document count
      const response = await axios.get(`${this.ragServerBaseUrl}/rag/count`, {
        timeout: 5000
      })

      if (response.data && typeof response.data.count === 'number') {
        this.documentStats = { count: response.data.count }
        this.lastRefreshTime = new Date()
        serviceLogger.debug(`Updated document count: ${this.documentStats.count}`)
      } else {
        serviceLogger.warn('Invalid response from RAG server count endpoint')
        this.documentStats.error = 'Invalid response from RAG server'
      }
    } catch (error) {
      // Type guard for axios errors
      if (axios.isAxiosError(error)) {
        if (error.response) {
          // The request was made and the server responded with a status code
          // that falls out of the range of 2xx
          serviceLogger.error(`Server error (${error.response.status}): ${error.response.data}`)
          this.documentStats.error = `Server error (${error.response.status}): ${error.response.statusText}`
        } else if (error.request) {
          // The request was made but no response was received
          serviceLogger.error('No response from RAG server:', this.ragServerBaseUrl)
          this.documentStats.error = 'No response from RAG server'
        } else {
          // Something happened in setting up the request that triggered an Error
          const errorMessage = error.message || 'Unknown error'
          serviceLogger.error('Error making request:', errorMessage)
          this.documentStats.error = `Request error: ${errorMessage}`
        }
      } else {
        // For non-axios errors
        const errorMessage = error instanceof Error ? error.message : String(error)
        serviceLogger.error('Error making request:', errorMessage)
        this.documentStats.error = `Request error: ${errorMessage}`
      }
    }
  }

  /**
   * Get the cached document count statistics
   */
  public getDocumentCount(): DocumentStats {
    return { ...this.documentStats }
  }

  /**
   * Check if the RAG server is available
   */
  private async checkServerAvailability(): Promise<boolean> {
    try {
      // Check if server is available by making a simple request to rag/count
      // This avoids adding a new health endpoint requirement
      const response = await axios.get(`${this.ragServerBaseUrl}/rag/count`, {
        timeout: 2000
      })

      this.isServerAvailable = response.status === 200
      return this.isServerAvailable
    } catch (error) {
      this.isServerAvailable = false
      if (axios.isAxiosError(error) && error.response) {
        // The request was made and the server responded with a status code that falls out of the range of 2xx
        serviceLogger.warn(
          `RAG server at ${this.ragServerBaseUrl} returned status ${error.response.status}`
        )
      } else if (axios.isAxiosError(error) && error.request) {
        // The request was made but no response was received
        serviceLogger.warn(
          `RAG server at ${this.ragServerBaseUrl} did not respond (connection refused or timeout)`
        )
      } else {
        // Something happened in setting up the request
        const errorMessage = error instanceof Error ? error.message : String(error)
        serviceLogger.warn(
          `Error connecting to RAG server at ${this.ragServerBaseUrl}: ${errorMessage}`
        )
      }
      return false
    }
  }

  /**
   * Search documents using the RAG server
   * First tries the cache for recent identical searches
   */
  public async searchDocuments(
    query: string,
    numResults: number = 5
  ): Promise<DocumentSearchResults> {
    try {
      // Update the URL in case it changed
      this.updateRagServerUrl()

      // Create a cache key based on query and numResults
      const cacheKey = `${query}:${numResults}`

      // Check if we have a recent cached result for this exact query
      const cachedResult = this.lastSearchResults.get(cacheKey)
      if (cachedResult) {
        const cacheAge = new Date().getTime() - this.lastRefreshTime.getTime()
        // Only use cache if it's less than 30 seconds old
        if (cacheAge < 30000) {
          serviceLogger.debug(`Using cached result for query: "${query}" (${cacheAge}ms old)`)
          return cachedResult
        }
      }

      // If server is not available, return empty results
      if (!this.isServerAvailable) {
        return { documents: [] }
      }

      let response
      // For empty query, get all active documents
      if (!query.trim()) {
        serviceLogger.info(`Getting all active documents`)

        // Simple GET request without any parameters - match original URL structure
        response = await axios.get(`${this.ragServerBaseUrl}/rag/active/true`, {
          timeout: 10000 // 10 second timeout
        })
      } else {
        // Normal search with query
        serviceLogger.info(
          `Searching RAG documents with query: "${query}", numResults: ${numResults}`
        )

        // Use the same URL structure as the original implementation
        response = await axios.get(`${this.ragServerBaseUrl}/rag`, {
          params: {
            query,
            num_results: numResults
          },
          timeout: 10000 // 10 second timeout
        })
      }

      // Process the response to ensure required fields exist
      if (response.data && response.data.documents) {
        response.data.documents.forEach((doc: any) => {
          // If the document doesn't have metadata, create an empty one
          if (!doc.metadata) {
            doc.metadata = { date: new Date().toLocaleString() }
          }
        })
      }

      // Cache this result for future use
      const result = { documents: response.data?.documents || [] }
      this.lastSearchResults.set(cacheKey, result)

      return result
    } catch (error) {
      serviceLogger.error(`Failed to search or get RAG documents:`, error)
      // Return empty results on error
      return { documents: [] }
    }
  }

  /**
   * Delete a document from the RAG server
   */
  public async deleteDocument(filename: string): Promise<{ success: boolean; message: string }> {
    try {
      // Update the URL in case it changed
      this.updateRagServerUrl()

      if (!this.isServerAvailable) {
        return {
          success: false,
          message: 'RAG server is not available'
        }
      }

      if (!filename) {
        return {
          success: false,
          message: 'Filename is required'
        }
      }

      serviceLogger.info(`Deleting document with filename: "${filename}"`)

      // Send DELETE request to the RAG server with the filename as a query parameter
      const response = await axios.delete(`${this.ragServerBaseUrl}/rag`, {
        params: {
          filename: filename
        },
        timeout: 10000 // 10 second timeout
      })

      if (response.status === 200) {
        // Update the document count after deletion
        this.fetchDocumentData().catch((error) => {
          serviceLogger.error('Failed to update document count after deletion:', error)
        })

        serviceLogger.info(`Successfully deleted document: ${filename}`)
        return {
          success: true,
          message: 'Document deleted successfully'
        }
      } else {
        serviceLogger.error(`Failed to delete document. Status: ${response.status}`)
        return {
          success: false,
          message: `Failed to delete document. Server returned: ${response.status}`
        }
      }
    } catch (error) {
      serviceLogger.error(`Failed to delete document:`, error)
      const errorMessage = error instanceof Error ? error.message : String(error)
      return {
        success: false,
        message: `Error deleting document: ${errorMessage}`
      }
    }
  }

  /**
   * Request cleanup of all documents from the RAG server
   */
  public async cleanupDocuments(): Promise<{ success: boolean; message: string }> {
    try {
      // Update the URL in case it changed
      this.updateRagServerUrl()

      if (!this.isServerAvailable) {
        return {
          success: false,
          message: 'RAG server is not available'
        }
      }

      // Send DELETE request to cleanup all documents
      const response = await axios.delete(`${this.ragServerBaseUrl}/rag/all`, {
        timeout: 10000 // 10 second timeout
      })

      if (response.status === 200) {
        // Reset local document count
        this.documentStats = { count: 0 }

        // Clear the search cache
        this.lastSearchResults.clear()

        serviceLogger.info('Successfully cleaned up all documents')
        return {
          success: true,
          message: 'All documents have been successfully removed.'
        }
      } else {
        const errorData = await response.data
        serviceLogger.error(
          `Failed to cleanup documents. Status: ${response.status}. Error: ${errorData}`
        )
        return {
          success: false,
          message: `Failed to cleanup documents. Server returned: ${response.status} ${errorData}`
        }
      }
    } catch (error) {
      serviceLogger.error('Error in cleanupDocuments function:', error)
      const errorMessage = error instanceof Error ? error.message : String(error)
      return {
        success: false,
        message: `Unexpected error while cleaning up documents: ${errorMessage}`
      }
    }
  }
}

// Create a singleton instance of DocumentService
export const documentService = new DocumentService()
