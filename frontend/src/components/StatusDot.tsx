import { FC } from 'react'

interface StatusDotProps {
  isRunning: boolean
}

export const StatusDot: FC<StatusDotProps> = ({ isRunning }) => (
  <span className="inline-flex items-center gap-2 text-xs font-mono tracking-wider uppercase">
    <span
      className={`w-2 h-2 rounded-full ${isRunning ? 'bg-emerald-400 pulse-glow' : 'bg-red-500/60'}`}
    />
    <span className="text-fog">
      {isRunning ? 'Conversing' : 'Silent'}
    </span>
  </span>
)
