import { useState, useEffect } from 'react'

/**
 * Custom hook to debounce any fast-changing value.
 *
 * @template T
 * @param {T} value - The input value to debounce
 * @param {number} delay - Debounce delay in milliseconds
 * @returns {T} The debounced value
 */
export function useDebounce(value, delay = 250) {
  const [debouncedValue, setDebouncedValue] = useState(value)

  useEffect(() => {
    const timer = setTimeout(() => {
      setDebouncedValue(value)
    }, delay)

    return () => {
      clearTimeout(timer)
    }
  }, [value, delay])

  return debouncedValue
}

export default useDebounce
