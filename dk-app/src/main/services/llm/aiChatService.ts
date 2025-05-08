import fs from 'fs'
import path from 'path'
import { app } from 'electron'
import * as SharedTypes from '@shared/types'
type AIMessage = SharedTypes.AIMessage
import logger from '@shared/logging'

// Define the path for storing AI chat history
const getAIChatHistoryPath = (): string => {
  const userDataPath = app.getPath('userData')
  return path.join(userDataPath, 'ai-chat-history.json')
}

// Get the AI Chat History from file
export const getAIChatHistory = async (): Promise<AIMessage[]> => {
  try {
    const historyPath = getAIChatHistoryPath()

    // If the file doesn't exist yet, return an empty array
    if (!fs.existsSync(historyPath)) {
      return []
    }

    // Read the file and parse the JSON
    const historyData = fs.readFileSync(historyPath, 'utf8')
    const messages: AIMessage[] = JSON.parse(historyData)

    // Convert timestamp strings back to Date objects
    return messages.map((msg) => ({
      ...msg,
      timestamp: new Date(msg.timestamp)
    }))
  } catch (error) {
    logger.error('Error loading AI chat history:', error)
    return []
  }
}

// Save AI Chat History to file
export const saveAIChatHistory = async (messages: AIMessage[]): Promise<boolean> => {
  try {
    const historyPath = getAIChatHistoryPath()

    // Convert to JSON and write to file
    const historyData = JSON.stringify(messages, null, 2)
    fs.writeFileSync(historyPath, historyData, 'utf8')

    return true
  } catch (error) {
    logger.error('Error saving AI chat history:', error)
    return false
  }
}

// Clear AI Chat History
export const clearAIChatHistory = async (): Promise<boolean> => {
  try {
    const historyPath = getAIChatHistoryPath()

    // Create a welcome message
    const welcomeMessage: AIMessage = {
      id: crypto.randomUUID(),
      role: 'assistant',
      content: "Hello! I'm your AI assistant. How can I help you today?",
      timestamp: new Date()
    }

    // Save just the welcome message
    return await saveAIChatHistory([welcomeMessage])
  } catch (error) {
    logger.error('Error clearing AI chat history:', error)
    return false
  }
}
