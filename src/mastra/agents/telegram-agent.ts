import { Agent } from '@mastra/core/agent'
import { createTelegramAdapter } from '@chat-adapter/telegram'

export const telegramAgent = new Agent({
  id: 'telegram-agent',
  name: 'Telegram Agent',
  instructions: 'Answer questions and help with tasks in Telegram.',
  model: "google/gemma-4-26b-a4b-it",
  channels: {
    adapters: {
      telegram: createTelegramAdapter({
        
      }),
    },
  },
})