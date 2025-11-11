/**
 * Pack Optimizer Frontend Application
 * 
 * This React application provides a user interface for the pack optimizer API.
 * It allows users to:
 * - Manage pack sizes (add/remove)
 * - Calculate optimal pack distributions for order amounts
 * - View detailed breakdowns of pack combinations
 * 
 * The UI uses inline styles with a Deep Navy/Indigo color scheme and provides
 * real-time validation, error handling, and user feedback.
 */

import React, { useMemo, useState } from 'react'

// API base URL - uses environment variable or defaults to localhost
const API = import.meta.env.VITE_API_URL || 'http://localhost:8080/api/v1'

/**
 * Parses API error response and returns a user-friendly error message.
 * Handles both JSON error objects and plain text errors.
 */
async function parseErrorResponse(res: Response): Promise<string> {
  // Clone response so we can read it multiple times if needed
  const clonedRes = res.clone()
  
  try {
    const errorData = await res.json()
    // Use message from API error, or construct from details
    if (errorData.message) {
      let message = errorData.message
      // Add details if available for better context
      if (errorData.details && errorData.details.reason) {
        message += `: ${errorData.details.reason}`
      }
      return message
    } else if (errorData.error) {
      return errorData.error
    }
    return 'An error occurred'
  } catch {
    // If JSON parsing fails, try text from cloned response
    try {
      const txt = await clonedRes.text()
      return txt || 'An error occurred'
    } catch {
      return 'An error occurred'
    }
  }
}

const styles = {
  app: {
    minHeight: '100vh',
    background: 'linear-gradient(135deg, #2C3E50 0%, #4A5568 100%)',
    padding: '2rem 1rem',
    fontFamily: '-apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif',
  },
  container: {
    maxWidth: 900,
    margin: '0 auto',
  },
  header: {
    textAlign: 'center' as const,
    color: '#ffffff',
    marginBottom: '2.5rem',
  },
  title: {
    fontSize: '2.5rem',
    fontWeight: 700,
    margin: 0,
    textShadow: '0 2px 10px rgba(0,0,0,0.2)',
    letterSpacing: '-0.02em',
  },
  subtitle: {
    fontSize: '1.1rem',
    opacity: 0.9,
    marginTop: '0.5rem',
    fontWeight: 400,
  },
  card: {
    background: '#ffffff',
    borderRadius: 16,
    padding: '2rem',
    marginBottom: '1.5rem',
    boxShadow: '0 10px 40px rgba(0,0,0,0.15)',
    transition: 'transform 0.2s, box-shadow 0.2s',
  },
  cardHover: {
    transform: 'translateY(-2px)',
    boxShadow: '0 15px 50px rgba(0,0,0,0.2)',
  },
  sectionTitle: {
    fontSize: '1.5rem',
    fontWeight: 600,
    color: '#2d3748',
    margin: '0 0 1rem 0',
  },
  sectionText: {
    color: '#718096',
    fontSize: '0.95rem',
    marginBottom: '1rem',
  },
  chipsContainer: {
    display: 'flex',
    flexWrap: 'wrap' as const,
    gap: '0.75rem',
    marginBottom: '1.5rem',
  },
  chip: {
    background: 'linear-gradient(135deg, #2C3E50 0%, #4A5568 100%)',
    color: '#ffffff',
    padding: '0.5rem 1rem',
    borderRadius: 20,
    display: 'inline-flex',
    alignItems: 'center',
    gap: '0.5rem',
    fontSize: '0.9rem',
    fontWeight: 500,
    boxShadow: '0 4px 12px rgba(44, 62, 80, 0.3)',
    transition: 'transform 0.2s, box-shadow 0.2s',
  },
  chipHover: {
    transform: 'scale(1.05)',
    boxShadow: '0 6px 16px rgba(44, 62, 80, 0.4)',
  },
  deleteBtn: {
    border: 'none',
    background: 'rgba(255,255,255,0.25)',
    color: '#ffffff',
    cursor: 'pointer',
    borderRadius: '50%',
    width: 20,
    height: 20,
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
    fontSize: '1.2rem',
    lineHeight: 1,
    padding: 0,
    transition: 'background 0.2s',
  },
  deleteBtnHover: {
    background: 'rgba(255,255,255,0.4)',
  },
  inputGroup: {
    display: 'flex',
    gap: '0.75rem',
    marginBottom: '1rem',
  },
  input: {
    flex: 1,
    padding: '0.875rem 1rem',
    fontSize: '1rem',
    border: '2px solid #e2e8f0',
    borderRadius: 10,
    outline: 'none',
    transition: 'border-color 0.2s, box-shadow 0.2s',
    fontFamily: 'inherit',
  },
  inputFocus: {
    borderColor: '#2C3E50',
    boxShadow: '0 0 0 3px rgba(44, 62, 80, 0.1)',
  },
  btn: {
    padding: '0.875rem 1.75rem',
    fontSize: '1rem',
    fontWeight: 600,
    border: 'none',
    borderRadius: 10,
    cursor: 'pointer',
    transition: 'all 0.2s',
    fontFamily: 'inherit',
    whiteSpace: 'nowrap' as const,
  },
  btnPrimary: {
    background: 'linear-gradient(135deg, #2C3E50 0%, #4A5568 100%)',
    color: '#ffffff',
    boxShadow: '0 4px 12px rgba(44, 62, 80, 0.3)',
  },
  btnPrimaryHover: {
    transform: 'translateY(-2px)',
    boxShadow: '0 6px 20px rgba(44, 62, 80, 0.4)',
  },
  btnSecondary: {
    background: '#f7fafc',
    color: '#4a5568',
    border: '2px solid #e2e8f0',
  },
  btnSecondaryHover: {
    background: '#edf2f7',
    borderColor: '#cbd5e0',
  },
  message: {
    padding: '0.75rem 1rem',
    borderRadius: 8,
    fontSize: '0.9rem',
    marginTop: '0.75rem',
    fontWeight: 500,
  },
  messageSuccess: {
    background: '#f0fff4',
    color: '#22543d',
    border: '1px solid #9ae6b4',
  },
  messageError: {
    background: '#fff5f5',
    color: '#742a2a',
    border: '1px solid #fc8181',
  },
  resultCard: {
    background: 'linear-gradient(135deg, #f6f8ff 0%, #f0f4ff 100%)',
    borderRadius: 12,
    padding: '1.5rem',
    marginTop: '1.5rem',
    border: '1px solid #e2e8f0',
  },
  resultStats: {
    display: 'flex',
    gap: '2rem',
    marginBottom: '1.5rem',
    flexWrap: 'wrap' as const,
  },
  stat: {
    display: 'flex',
    flexDirection: 'column' as const,
  },
  statLabel: {
    fontSize: '0.85rem',
    color: '#718096',
    fontWeight: 500,
    marginBottom: '0.25rem',
  },
  statValue: {
    fontSize: '1.5rem',
    fontWeight: 700,
    color: '#2d3748',
  },
  table: {
    width: '100%',
    borderCollapse: 'collapse' as const,
    marginTop: '1rem',
    background: '#ffffff',
    borderRadius: 8,
    overflow: 'hidden',
    boxShadow: '0 2px 8px rgba(0,0,0,0.05)',
  },
  tableHeader: {
    background: 'linear-gradient(135deg, #2C3E50 0%, #4A5568 100%)',
    color: '#ffffff',
  },
  tableHeaderCell: {
    padding: '0.875rem 1rem',
    textAlign: 'left' as const,
    fontWeight: 600,
    fontSize: '0.9rem',
  },
  tableCell: {
    padding: '0.875rem 1rem',
    borderTop: '1px solid #e2e8f0',
    color: '#2d3748',
  },
  tableRow: {
    transition: 'background 0.15s',
  },
  tableRowHover: {
    background: '#f7fafc',
  },
}

/**
 * Main App component - root component of the application.
 * Renders the header and two main sections: PackSizes and Calculator.
 */
export default function App() {
  return (
    <div style={styles.app}>
      <div style={styles.container}>
        <header style={styles.header}>
          <h1 style={styles.title}>📦 Pack Optimizer</h1>
          <p style={styles.subtitle}>Calculate optimal pack combinations for your orders</p>
        </header>
        <PackSizes />
        <Calculator />
      </div>
    </div>
  )
}

/**
 * Custom hook for managing active pack sizes.
 * Fetches pack sizes from the API on mount and provides a refresh function.
 * 
 * @returns Object containing:
 *   - sizes: Current array of pack sizes
 *   - refresh: Function to refetch sizes from API
 *   - setSizes: Function to update sizes directly (for testing)
 */
function useActiveSizes() {
  const [sizes, setSizes] = React.useState<number[]>([])
  const fetchSizes = React.useCallback(async () => {
    const res = await fetch(`${API}/packs`)
    const data = await res.json()
    setSizes(data.sizes || [])
  }, [])
  React.useEffect(() => { fetchSizes() }, [fetchSizes])
  return { sizes, refresh: fetchSizes, setSizes }
}

/**
 * PackSizes component - manages pack size configuration.
 * Allows users to:
 * - View current pack sizes as interactive chips
 * - Add new pack sizes (one at a time)
 * - Delete pack sizes by clicking the × button on each chip
 * 
 * Validates input: pack sizes must be positive integers <= 10,000.
 * Shows success/error messages inline below the input.
 */
function PackSizes() {
  const { sizes, refresh } = useActiveSizes()
  const [editing, setEditing] = useState<string>('') // Current input value
  const [msg, setMsg] = useState<{ kind:'ok'|'err'; text:string }|null>(null) // Status message
  const [hoverChip, setHoverChip] = useState<number | null>(null) // Hover state for chips
  const [hoverDelete, setHoverDelete] = useState<number | null>(null) // Hover state for delete buttons

  /**
   * Adds a new pack size to the active set.
   * Validates the input, checks for duplicates, and sends PUT request to API.
   * Updates the UI with success/error feedback.
   */
  const addOne = async () => {
    const val = parseInt(editing, 10)
    if (!Number.isFinite(val) || val <= 0) {
      setMsg({ kind:'err', text:'Enter a positive integer size' })
      return
    }
    if (val > 10000) {
      setMsg({ kind:'err', text:'Pack size cannot exceed 10,000 items' })
      return
    }
    if (sizes.includes(val)) {
      setMsg({ kind:'err', text:`Size ${val} already exists` })
      return
    }
    const uniq = Array.from(new Set([...sizes, val])).sort((a,b)=>a-b)
    const res = await fetch(`${API}/packs`, { method: 'PUT', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ sizes: uniq }) })
    if (res.ok) { 
      await refresh()
      setMsg({ kind:'ok', text:`✓ Added pack size ${val}` })
      setTimeout(() => setMsg(null), 3000)
    } else {
      const errorMsg = await parseErrorResponse(res)
      setMsg({ kind:'err', text:errorMsg || 'Failed to add size' })
      setTimeout(() => setMsg(null), 3000)
    }
    setEditing('')
  }

  /**
   * Removes a pack size from the active set.
   * Sends DELETE request to API and refreshes the pack sizes list.
   * Prevents event propagation to avoid unintended side effects.
   * 
   * @param val - The pack size value to remove
   * @param e - Optional mouse event (used to prevent default behavior)
   */
  const remove = async (val:number, e?: React.MouseEvent) => {
    if (e) {
      e.preventDefault()
      e.stopPropagation()
    }
    const res = await fetch(`${API}/packs/${val}`, { method: 'DELETE' })
    if (res.ok) { 
      await refresh()
      setMsg({ kind:'ok', text:`✓ Removed pack size ${val}` })
      setTimeout(() => setMsg(null), 3000)
    } else {
      const errorMsg = await parseErrorResponse(res)
      setMsg({ kind:'err', text:errorMsg || 'Failed to remove pack size' })
      setTimeout(() => setMsg(null), 3000)
    }
  }

  return (
    <div style={styles.card}>
      <h2 style={styles.sectionTitle}>Pack Sizes</h2>
      <p style={styles.sectionText}>Manage available pack sizes. Current sizes:</p>
      {sizes.length > 0 && (
        <div style={styles.chipsContainer}>
          {sizes.map(s => (
            <span 
              key={s} 
              style={{
                ...styles.chip,
                ...(hoverChip === s ? styles.chipHover : {})
              }}
              onMouseEnter={() => setHoverChip(s)}
              onMouseLeave={() => setHoverChip(null)}
            >
              {s} items
              <button 
                onClick={(e)=>remove(s, e)} 
                title="Delete" 
                type="button"
                style={{
                  ...styles.deleteBtn,
                  ...(hoverDelete === s ? styles.deleteBtnHover : {})
                }}
                onMouseEnter={() => setHoverDelete(s)}
                onMouseLeave={() => setHoverDelete(null)}
              >
                ×
              </button>
            </span>
          ))}
        </div>
      )}
      <div style={styles.inputGroup}>
        <input
          value={editing}
          onChange={e => setEditing(e.target.value.replace(/[^0-9]/g, ''))}
          onKeyPress={e => e.key === 'Enter' && addOne()}
          style={styles.input}
          inputMode="numeric"
          pattern="[0-9]*"
        />
        <button 
          onClick={addOne}
          style={{...styles.btn, ...styles.btnPrimary}}
          onMouseEnter={e => e.currentTarget.style.transform = 'translateY(-2px)'}
          onMouseLeave={e => e.currentTarget.style.transform = 'translateY(0)'}
        >
          Add Size
        </button>
      </div>
      {msg && (
        <div style={{
          ...styles.message,
          ...(msg.kind === 'ok' ? styles.messageSuccess : styles.messageError)
        }}>
          {msg.text}
        </div>
      )}
    </div>
  )
}

/**
 * Calculator component - calculates optimal pack distribution.
 * Allows users to:
 * - Enter an order amount
 * - Calculate optimal pack combination using active pack sizes
 * - View detailed breakdown with total items, overage, and pack quantities
 * 
 * Validates input: amount must be positive integer <= 1,000,000.
 * Shows loading state during calculation and error messages on failure.
 */
function Calculator() {
  const [amount, setAmount] = useState('500000') // Order amount input (defaults to edge case value)
  const [result, setResult] = useState<any>(null) // Calculation result from API
  const [loading, setLoading] = useState(false) // Loading state during API call
  const [inputFocused, setInputFocused] = useState(false) // Input focus state for styling
  const [error, setError] = useState<string | null>(null) // Error message state
  const MAX_AMOUNT = 1_000_000 // Maximum allowed order amount

  /**
   * Performs pack calculation by sending POST request to API.
   * Validates amount before sending, handles errors, and updates UI state.
   */
  const calc = async () => {
    if (!amount) {
      return
    }
    const numAmount = parseInt(amount, 10)
    if (numAmount > MAX_AMOUNT) {
      setError(`Amount cannot exceed ${MAX_AMOUNT.toLocaleString()} items`)
      setResult(null)
      return
    }
    setError(null)
    setLoading(true)
    try {
      const res = await fetch(`${API}/calculate`, { 
        method: 'POST', 
        headers: { 'Content-Type': 'application/json' }, 
        body: JSON.stringify({ amount: numAmount }) 
      })
      if (!res.ok) {
        const errorMessage = await parseErrorResponse(res)
        setError(errorMessage || 'Calculation failed')
        setResult(null)
        return
      }
      setResult(await res.json())
    } catch (err) {
      setError('Failed to calculate. Please try again.')
      setResult(null)
    } finally {
      setLoading(false)
    }
  }

  return (
    <div style={styles.card}>
      <h2 style={styles.sectionTitle}>Calculate Packs</h2>
      <p style={styles.sectionText}>Enter the number of items to calculate optimal pack distribution</p>
      <div style={styles.inputGroup}>
        <input
          value={amount}
          onChange={e => {
            const val = e.target.value.replace(/[^0-9]/g, '')
            setAmount(val)
            if (val) {
              const num = parseInt(val, 10)
              if (num > MAX_AMOUNT) {
                setError(`Amount cannot exceed ${MAX_AMOUNT.toLocaleString()} items`)
              } else {
                setError(null)
              }
            } else {
              setError(null)
            }
          }}
          onKeyPress={e => e.key === 'Enter' && calc()}
          onFocus={() => setInputFocused(true)}
          onBlur={() => setInputFocused(false)}
          style={{
            ...styles.input,
            ...(inputFocused ? styles.inputFocus : {})
          }}
          inputMode="numeric"
          pattern="[0-9]*"
        />
        <button 
          onClick={calc}
          disabled={loading || !amount}
          style={{
            ...styles.btn, 
            ...styles.btnPrimary,
            opacity: (loading || !amount) ? 0.6 : 1,
            cursor: (loading || !amount) ? 'not-allowed' : 'pointer'
          }}
          onMouseEnter={e => !loading && amount && (e.currentTarget.style.transform = 'translateY(-2px)')}
          onMouseLeave={e => e.currentTarget.style.transform = 'translateY(0)'}
        >
          {loading ? 'Calculating...' : 'Calculate'}
        </button>
      </div>
      {error && (
        <div style={{
          ...styles.message,
          ...styles.messageError
        }}>
          {error}
        </div>
      )}
      {result && <Result res={result} />}
    </div>
  )
}

/**
 * Result component - displays calculation results.
 * Shows:
 * - Total items, overage, and total packs as statistics
 * - Detailed breakdown table with pack sizes and quantities
 * 
 * Sorts breakdown by pack size (descending) for better readability.
 * 
 * @param res - Calculation result object from API
 */
function Result({ res }:{ res:any }) {
  const [hoverRow, setHoverRow] = useState<string | null>(null) // Hover state for table rows
  // Sort breakdown entries by pack size (descending) for display
  const entries = Object.entries(res.breakdown || {}).sort(([a], [b]) => parseInt(b) - parseInt(a))
  
  return (
    <div style={styles.resultCard}>
      <div style={styles.resultStats}>
        <div style={styles.stat}>
          <div style={styles.statLabel}>Total Items</div>
          <div style={styles.statValue}>{res.totalItems.toLocaleString()}</div>
        </div>
        <div style={styles.stat}>
          <div style={styles.statLabel}>Overage</div>
          <div style={styles.statValue}>{res.overage.toLocaleString()}</div>
        </div>
        <div style={styles.stat}>
          <div style={styles.statLabel}>Total Packs</div>
          <div style={styles.statValue}>{res.totalPacks}</div>
        </div>
      </div>
      {entries.length > 0 && (
        <table style={styles.table}>
          <thead style={styles.tableHeader}>
            <tr>
              <th style={styles.tableHeaderCell}>Pack Size</th>
              <th style={styles.tableHeaderCell}>Quantity</th>
            </tr>
          </thead>
          <tbody>
            {entries.map(([k,v]) => (
              <tr 
                key={k}
                style={{
                  ...styles.tableRow,
                  ...(hoverRow === k ? styles.tableRowHover : {})
                }}
                onMouseEnter={() => setHoverRow(k)}
                onMouseLeave={() => setHoverRow(null)}
              >
                <td style={styles.tableCell}>{k} items</td>
                <td style={styles.tableCell}><strong>{v as any}</strong></td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </div>
  )
}
