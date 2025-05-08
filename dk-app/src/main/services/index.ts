// Export all services
export * from './client'
export * from './clientService'
export * from './config'
export * from './llm'
export * from './database'
export * from './appService'
export * from './trackerService'
export * from './documentService'

// Initialize services in the correct order
import { loadConfig, getOnboardingStatus, getConfigFilePath, appConfig } from './config'
import { initDatabaseService } from './database'
import { loadSyftboxConfig } from './appService'
import { trackerService } from './trackerService'
import { documentService } from './documentService'
import logger from '../../shared/logging'
import { existsSync } from 'fs'
import { join } from 'path'
import { app } from 'electron'

/**
 * Initialize all services in the correct order to avoid circular dependencies
 */
export function initializeServices(): void {
  logger.info('Initializing all services...')

  // 1. First load configuration
  loadConfig()

  // 2. Load SyftBox configuration if available (uses environment variable set by loadConfig)
  loadSyftboxConfig()

  // 3. Initialize database (this will also call initializeAppTrackers)
  initDatabaseService()

  // 4. Check if we should start services that require complete configuration
  const configPath = getConfigFilePath()
  const isConfigExists = existsSync(configPath)
  const onboardingStatus = getOnboardingStatus()

  // Only start services when:
  // 1. Config file exists AND
  // 2. UserID is not empty or the default (to ensure it's a valid config) AND
  // 3. Onboarding is not in first run state
  const isValidConfig = isConfigExists && appConfig.userID && appConfig.userID !== 'default-user'
  const shouldStartServices = isValidConfig && !onboardingStatus.isFirstRun

  // 5. Start tracker scanning and document data services if configuration is complete
  if (shouldStartServices) {
    logger.info('Configuration exists and onboarding completed - starting background services')

    // Start tracker scanning service
    trackerService.startTrackerScan()

    // Start document data fetching service
    documentService.startDocumentDataFetch()
  } else {
    logger.info(
      'Configuration not set up or onboarding not completed - background services will not start'
    )
  }

  logger.info('All services initialized')
}
