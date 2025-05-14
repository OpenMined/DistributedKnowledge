/**
 * Utility functions for formatting data in the renderer
 */

/**
 * Convert a Date object to a readable string format
 * Renderer-safe version that doesn't depend on Node.js OS module
 */
export function formatMessageTimestamp(date: Date | string): string {
  // Convert string dates to Date objects for consistent formatting
  let dateObj: Date

  if (typeof date === 'string') {
    // Check if it's already a formatted string (not an ISO date)
    if (!/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}/.test(date)) {
      return date // Return already formatted strings
    }
    // Parse ISO date string into Date object
    dateObj = new Date(date)
  } else {
    dateObj = date
  }

  const now = new Date()
  const today = new Date(now.getFullYear(), now.getMonth(), now.getDate())
  const yesterday = new Date(today)
  yesterday.setDate(yesterday.getDate() - 1)

  // Date is today
  if (dateObj >= today) {
    return dateObj.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
  }

  // Date is yesterday
  if (dateObj >= yesterday) {
    return 'Yesterday'
  }

  // Date is within last 7 days
  const lastWeek = new Date(today)
  lastWeek.setDate(lastWeek.getDate() - 6)
  if (dateObj >= lastWeek) {
    const options: Intl.DateTimeFormatOptions = { weekday: 'long' }
    return dateObj.toLocaleDateString(undefined, options)
  }

  // Date is older than a week but in the current year
  if (dateObj.getFullYear() === now.getFullYear()) {
    const options: Intl.DateTimeFormatOptions = { month: 'short', day: 'numeric' }
    return dateObj.toLocaleDateString(undefined, options)
  }

  // Date is from a different year
  const options: Intl.DateTimeFormatOptions = { year: 'numeric', month: 'short', day: 'numeric' }
  return dateObj.toLocaleDateString(undefined, options)
}

/**
 * Format file size for display
 */
export function formatFileSize(bytes?: number): string {
  if (bytes === undefined) return 'Unknown size'
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  if (bytes < 1024 * 1024 * 1024) return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
  return `${(bytes / (1024 * 1024 * 1024)).toFixed(1)} GB`
}
