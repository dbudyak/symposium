export interface Message {
  id: string
  agent_id: string
  agent_name: string
  content: string
  created_at: string
  reply_to?: string
}

export interface AgentInfo {
  slug: string
  name: string
  color: string
}

export interface StatusResponse {
  is_running: boolean
  message_count: number
  agents: AgentInfo[]
  cooldown_seconds: number
}

export interface ErrorResponse {
  error: string
  retry_after?: number
}

export const AGENT_COLORS: Record<string, string> = {
  diogenes: '#E8A838',
  hypatia: '#7EB8DA',
  tesla: '#B088F9',
  curie: '#5DE8A0',
  cioran: '#F25C54',
  turing: '#6EC8C8',
  ada: '#F2A2C0',
  camus: '#D4D4D4',
  sagan: '#4A90D9',
  hawking: '#1CA3EC',
  jung: '#C77DBA',
  freud: '#D4A574',
  lynch: '#E84040',
  dali: '#FFD700',
  koda: '#CC6A2B',
  human: '#8B8B8B',
}
