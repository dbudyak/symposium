import { FC, useState, useEffect, useCallback, useRef, FormEvent } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { postMessage, fetchStatus } from '../api/client'

export const InputBar: FC = () => {
  const [text, setText] = useState('')
  const [cooldown, setCooldown] = useState(0)
  const queryClient = useQueryClient()

  const { data: status } = useQuery({
    queryKey: ['status'],
    queryFn: fetchStatus,
    refetchInterval: 10000,
  })

  // Sync cooldown from server
  useEffect(() => {
    if (status?.cooldown_seconds && status.cooldown_seconds > 0) {
      setCooldown(status.cooldown_seconds)
    }
  }, [status?.cooldown_seconds])

  // Countdown timer
  useEffect(() => {
    if (cooldown <= 0) return
    const timer = setInterval(() => {
      setCooldown(c => {
        if (c <= 1) {
          clearInterval(timer)
          return 0
        }
        return c - 1
      })
    }, 1000)
    return () => clearInterval(timer)
  }, [cooldown])

  const mutation = useMutation({
    mutationFn: postMessage,
    onSuccess: () => {
      setText('')
      setCooldown(3600)
      queryClient.invalidateQueries({ queryKey: ['messages-poll'] })
    },
    onError: (err: { retry_after?: number }) => {
      if (err.retry_after) {
        setCooldown(err.retry_after)
      }
    },
  })

  const handleSubmit = useCallback((e: FormEvent) => {
    e.preventDefault()
    const trimmed = text.trim()
    if (!trimmed || cooldown > 0 || mutation.isPending) return
    mutation.mutate(trimmed)
  }, [text, cooldown, mutation])

  const formatCooldown = (seconds: number): string => {
    const m = Math.floor(seconds / 60)
    const s = seconds % 60
    return `${m}:${s.toString().padStart(2, '0')}`
  }

  const inputRef = useRef<HTMLInputElement>(null)

  // Listen for agent name clicks in chat
  useEffect(() => {
    const handler = (e: Event) => {
      const name = (e as CustomEvent<string>).detail
      if (cooldown > 0 || mutation.isPending) return
      setText(prev => prev ? `${prev}${name}, ` : `${name}, `)
      inputRef.current?.focus()
    }
    window.addEventListener('symposium:mention', handler)
    return () => window.removeEventListener('symposium:mention', handler)
  }, [cooldown, mutation.isPending])

  const disabled = cooldown > 0 || mutation.isPending

  return (
    <form onSubmit={handleSubmit} className="border-t border-smoke/60 bg-abyss/80 backdrop-blur-sm px-4 py-3">
      <div className="flex gap-3 items-center max-w-4xl mx-auto">
        <input
          ref={inputRef}
          type="text"
          value={text}
          onChange={e => setText(e.target.value)}
          disabled={disabled}
          maxLength={500}
          placeholder={cooldown > 0 ? `Next utterance in ${formatCooldown(cooldown)}` : 'Cast your words into the void...'}
          className="flex-1 bg-smoke/40 border border-smoke rounded-md px-4 py-2.5 text-sm font-mono text-bone placeholder-fog/30 focus:outline-none focus:border-fog/40 transition-colors disabled:opacity-40"
        />
        <button
          type="submit"
          disabled={disabled || !text.trim()}
          className="px-5 py-2.5 text-xs font-mono uppercase tracking-widest bg-smoke/60 border border-smoke hover:border-fog/40 text-fog hover:text-bone rounded-md transition-all disabled:opacity-30 disabled:cursor-not-allowed"
        >
          Speak
        </button>
      </div>
      {cooldown > 0 && (
        <p className="text-center text-[10px] text-fog/30 mt-1.5 font-mono">
          One voice per hour. The philosophers require patience.
        </p>
      )}
    </form>
  )
}
