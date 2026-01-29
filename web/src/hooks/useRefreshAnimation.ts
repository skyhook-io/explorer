import { useState, useCallback, useRef } from 'react'

const MIN_ANIMATION_DURATION = 400 // ms

/**
 * Hook that ensures a refresh animation runs for a minimum duration,
 * providing visual feedback even when the operation completes immediately.
 *
 * @param refetchFn - The actual refetch function to call
 * @returns [wrappedRefetch, isAnimating] - A wrapped function and animation state
 */
export function useRefreshAnimation(refetchFn: () => void | Promise<unknown>): [() => void, boolean] {
  const [isAnimating, setIsAnimating] = useState(false)
  const animationTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null)

  const wrappedRefetch = useCallback(() => {
    // Clear any existing timeout
    if (animationTimeoutRef.current) {
      clearTimeout(animationTimeoutRef.current)
    }

    // Start animation
    setIsAnimating(true)

    // Call the actual refetch
    const result = refetchFn()

    // Ensure minimum animation duration
    const startTime = Date.now()

    const stopAnimation = () => {
      const elapsed = Date.now() - startTime
      const remaining = MIN_ANIMATION_DURATION - elapsed

      if (remaining > 0) {
        animationTimeoutRef.current = setTimeout(() => {
          setIsAnimating(false)
        }, remaining)
      } else {
        setIsAnimating(false)
      }
    }

    if (result instanceof Promise) {
      result.finally(stopAnimation)
    } else {
      stopAnimation()
    }
  }, [refetchFn])

  return [wrappedRefetch, isAnimating]
}
