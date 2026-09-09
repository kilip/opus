import type { UIDataTypes, UIMessage, UITools } from 'ai'
import type { PageLoad } from './$types'

export const load: PageLoad = async ({ fetch }) => {
  const response = await fetch('/api/chat')
  const initialMessages = (await response.json()) as UIMessage<unknown, UIDataTypes, UITools>[]
  return { initialMessages }
}