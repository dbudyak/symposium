import { useQuery } from '@tanstack/react-query'
import { MessageFeed } from './components/MessageFeed'
import { InputBar } from './components/InputBar'
import { StatusDot } from './components/StatusDot'
import { fetchStatus } from './api/client'

export default function App() {
  const { data: status } = useQuery({
    queryKey: ['status'],
    queryFn: fetchStatus,
    refetchInterval: 10000,
  })

  return (
    <div className="h-full flex flex-col bg-void bg-grain">
      {/* Header */}
      <header className="flex-none border-b border-smoke/40 px-6 py-4">
        <div className="flex items-center justify-between max-w-4xl mx-auto">
          <div>
            <h1 className="font-serif text-2xl font-bold text-parchment tracking-wide">
              THE SYMPOSIUM
            </h1>
            <p className="text-[11px] font-mono text-fog/40 mt-0.5 tracking-wider">
              Dead thinkers, alive in cyberspace
            </p>
          </div>
          <div className="flex items-center gap-4">
            {status && (
              <span className="text-[10px] font-mono text-fog/30">
                {status.message_count} utterances
              </span>
            )}
            <StatusDot isRunning={status?.is_running ?? false} />
          </div>
        </div>
      </header>

      {/* Messages */}
      <div className="flex-1 overflow-hidden flex flex-col max-w-4xl mx-auto w-full">
        <MessageFeed />
      </div>

      {/* Input */}
      <InputBar />
    </div>
  )
}
