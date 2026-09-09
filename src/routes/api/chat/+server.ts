import type { RequestHandler } from './$types'
import { handleChatStream } from '@mastra/ai-sdk'
import { toAISdkV5Messages } from '@mastra/ai-sdk/ui'
import { createUIMessageStreamResponse, type UIMessageChunk } from 'ai'
import { mastra } from '../../../mastra'

const THREAD_ID = 'example-user-id'
const RESOURCE_ID = 'weather-chat'

export const POST: RequestHandler = async ({ request }) => {
  const params = await request.json()
  const stream = await handleChatStream({
    mastra,
    agentId: 'weather-agent',
    params: {
      ...params,
      memory: {
        ...params.memory,
        thread: THREAD_ID,
        resource: RESOURCE_ID,
      },
    },
  })

  return createUIMessageStreamResponse({
    stream: stream as unknown as ReadableStream<UIMessageChunk>,
  })
}

export const GET: RequestHandler = async () => {
  const memory = await mastra.getAgentById('weather-agent').getMemory()
  let response = null

  try {
    response = await memory?.recall({
      threadId: THREAD_ID,
      resourceId: RESOURCE_ID,
    })
  } catch {
    console.log('No previous messages found.')
  }

  const uiMessages = toAISdkV5Messages(response?.messages || [])

  return Response.json(uiMessages)
}