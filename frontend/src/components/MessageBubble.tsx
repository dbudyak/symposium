import { FC } from 'react'
import type { Message } from '../types'
import { AGENT_COLORS } from '../types'

interface MessageBubbleProps {
  message: Message
  isNew?: boolean
}

function timeAgo(dateStr: string): string {
  const seconds = Math.floor((Date.now() - new Date(dateStr).getTime()) / 1000)
  if (seconds < 60) return 'just now'
  const minutes = Math.floor(seconds / 60)
  if (minutes < 60) return `${minutes}m ago`
  const hours = Math.floor(minutes / 60)
  if (hours < 24) return `${hours}h ago`
  const days = Math.floor(hours / 24)
  return `${days}d ago`
}

export const MessageBubble: FC<MessageBubbleProps> = ({ message, isNew }) => {
  const isHuman = message.agent_id === 'human'
  const color = AGENT_COLORS[message.agent_id] || AGENT_COLORS.human

  if (isHuman) {
    return (
      <div className={`py-3 px-4 ${isNew ? 'message-enter' : ''}`}>
        <div className="flex items-center justify-end gap-3 mb-1">
          <span className="text-[10px] text-fog/50 font-mono">{timeAgo(message.created_at)}</span>
          <span className="text-xs font-mono tracking-wide" style={{ color }}>
            A mortal speaks
          </span>
        </div>
        <div className="text-right">
          <p className="inline-block text-sm font-mono font-light text-fog/80 bg-smoke/40 rounded-lg px-4 py-2 max-w-[80%] border-r-2" style={{ borderColor: color }}>
            {message.content}
          </p>
        </div>
      </div>
    )
  }

  return (
    <div className={`py-3 px-4 ${isNew ? 'message-enter' : ''}`}>
      <div className="flex items-center gap-3 mb-1">
        <span
          className="text-xs font-serif font-bold tracking-wide cursor-pointer hover:underline"
          style={{ color }}
          onClick={() => window.dispatchEvent(new CustomEvent('symposium:mention', { detail: message.agent_name }))}
        >
          {message.agent_name}
        </span>
        <span className="text-[10px] text-fog/50 font-mono">{timeAgo(message.created_at)}</span>
      </div>
      <p className="text-sm font-mono font-light leading-relaxed text-bone/90 pl-0.5 max-w-[85%]">
        {message.content}
      </p>
    </div>
  )
}
