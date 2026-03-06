import { FC, useEffect, useRef, useState, useCallback } from 'react'
import { useQuery } from '@tanstack/react-query'
import { fetchMessages, fetchMessagesSince } from '../api/client'
import { MessageBubble } from './MessageBubble'
import type { Message } from '../types'

export const MessageFeed: FC = () => {
  const [messages, setMessages] = useState<Message[]>([])
  const [newIds, setNewIds] = useState<Set<string>>(new Set())
  const [, setTick] = useState(0)
  const bottomRef = useRef<HTMLDivElement>(null)
  const containerRef = useRef<HTMLDivElement>(null)
  const isNearBottom = useRef(true)
  const initialLoadDone = useRef(false)

  // Initial load
  const { data: initialMessages } = useQuery({
    queryKey: ['messages-initial'],
    queryFn: () => fetchMessages(80),
    staleTime: Infinity,
    refetchOnWindowFocus: false,
  })

  useEffect(() => {
    if (initialMessages && !initialLoadDone.current) {
      // Reverse because API returns newest first
      setMessages([...initialMessages].reverse())
      initialLoadDone.current = true
      setTimeout(() => bottomRef.current?.scrollIntoView(), 50)
    }
  }, [initialMessages])

  // Periodic tick to keep timestamps fresh
  useEffect(() => {
    const timer = setInterval(() => setTick(t => t + 1), 60000)
    return () => clearInterval(timer)
  }, [])

  // Poll for new messages
  const lastId = messages.length > 0 ? messages[messages.length - 1].id : null

  useQuery({
    queryKey: ['messages-poll', lastId],
    queryFn: () => fetchMessagesSince(lastId!),
    enabled: !!lastId,
    refetchInterval: 3000,
    select: useCallback((data: Message[]) => {
      if (data.length > 0) {
        setMessages(prev => {
          const existingIds = new Set(prev.map(m => m.id))
          const fresh = data.filter(m => !existingIds.has(m.id))
          if (fresh.length === 0) return prev
          setNewIds(new Set(fresh.map(m => m.id)))
          return [...prev, ...fresh]
        })
      }
      return data
    }, []),
  })

  // Clear "new" status after animation
  useEffect(() => {
    if (newIds.size > 0) {
      const timer = setTimeout(() => setNewIds(new Set()), 500)
      return () => clearTimeout(timer)
    }
  }, [newIds])

  // Track scroll position
  const handleScroll = useCallback(() => {
    const el = containerRef.current
    if (!el) return
    isNearBottom.current = el.scrollHeight - el.scrollTop - el.clientHeight < 100
  }, [])

  // Auto-scroll when new messages arrive and user is near bottom
  useEffect(() => {
    if (isNearBottom.current) {
      bottomRef.current?.scrollIntoView({ behavior: 'smooth' })
    }
  }, [messages.length])

  return (
    <div
      ref={containerRef}
      onScroll={handleScroll}
      className="flex-1 overflow-y-auto"
    >
      {messages.length === 0 ? (
        <div className="flex items-center justify-center h-full">
          <p className="text-fog/40 font-serif italic text-lg">The hall awaits its first words...</p>
        </div>
      ) : (
        <div className="py-4 space-y-1">
          {messages.map(msg => (
            <MessageBubble key={msg.id} message={msg} isNew={newIds.has(msg.id)} />
          ))}
        </div>
      )}
      <div ref={bottomRef} />
    </div>
  )
}
